package main

import (
	"context"
	"expvar"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on http.DefaultServeMux
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"sakukota-go/internal/config"
	"sakukota-go/internal/database"
	"sakukota-go/internal/router"
)

func main() {
	// Also the destination of the log package, so log.Printf shares the format
	// (and, once set up below, the OTLP export) of slog.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Metrics are opt-in and served apart from the API so they never go out
	// through the public proxy. The default Prometheus registry also carries Go
	// runtime and process metrics.
	if addr := os.Getenv("METRICS_ADDR"); addr != "" {
		exporter, err := otelprometheus.New()
		if err != nil {
			log.Fatalf("create prometheus exporter: %v", err)
		}
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter)))
		go func() {
			log.Printf("metrics listening on %s", addr)
			log.Println(http.ListenAndServe(addr, promhttp.Handler())) //nolint:gosec // opt-in, internal only
		}()
	}

	// Tracing is opt-in. The exporter reads the standard OTEL_EXPORTER_OTLP_*
	// variables, and the service name comes from OTEL_SERVICE_NAME.
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		exporter, err := otlptracehttp.New(context.Background())
		if err != nil {
			log.Fatalf("create trace exporter: %v", err)
		}
		// shortcut: spans still buffered when the process exits are lost; call
		// Shutdown on the provider once the server shuts down gracefully.
		otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter)))
	}

	// Logs also go to an OTLP endpoint, such as Loki, when one is set. Each
	// record there carries the trace and span of its context.
	if os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") != "" {
		exporter, err := otlploghttp.New(context.Background())
		if err != nil {
			log.Fatalf("create log exporter: %v", err)
		}
		// shortcut: like spans, buffered records are lost when the process exits.
		provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
		otlpHandler := otelslog.NewHandler("sakukota-go/cmd/api", otelslog.WithLoggerProvider(provider))
		slog.SetDefault(slog.New(slog.NewMultiHandler(slog.Default().Handler(), otlpHandler)))
	}

	db, err := database.NewPostgresDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer func() { _ = db.Close() }()
	expvar.Publish("dbstats", expvar.Func(func() any { return db.Stats() }))
	if err := prometheus.Register(collectors.NewDBStatsCollector(db.DB, "postgres")); err != nil {
		log.Fatalf("register database metrics: %v", err)
	}

	rdb, err := database.NewRedisClient(context.Background(), cfg.RedisURL)
	if err != nil {
		log.Fatalf("create redis connection: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	// Profiling is opt-in and served apart from the API, which has write
	// timeouts that would cut off long profiles.
	if addr := os.Getenv("PPROF_ADDR"); addr != "" {
		go func() {
			log.Printf("pprof listening on %s", addr)
			log.Println(http.ListenAndServe(addr, nil)) //nolint:gosec // opt-in, meant for localhost
		}()
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router.NewRouter(db, rdb, cfg),

		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
