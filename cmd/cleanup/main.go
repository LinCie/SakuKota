// Command cleanup deletes one batch of storage objects queued for deletion and
// exits. Run it on a schedule, such as every 15 minutes from cron.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"rantau-go/internal/config"
	"rantau-go/internal/database"
	"rantau-go/internal/module/storagecleanup"
	"rantau-go/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx := context.Background()

	flush, err := setupTelemetry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	runCtx, span := otel.Tracer("rantau-go/cmd/cleanup").Start(ctx, "storage cleanup")
	err = run(runCtx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(runCtx, "clean up storage", "err", err)
	}
	span.End()

	// Flush here, not in a defer: os.Exit skips deferred calls, and the
	// telemetry of a failed run is the most worth keeping.
	if err := flush(ctx); err != nil {
		log.Printf("flush telemetry: %v", err)
	}
	if err != nil {
		os.Exit(1)
	}
}

// run deletes one batch of queued storage objects.
func run(ctx context.Context) error {
	databaseURL, err := config.DatabaseURL()
	if err != nil {
		return err
	}

	storageCfg, err := config.NewStorageConfig()
	if err != nil {
		return err
	}

	db, err := database.NewPostgresDB(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("create database connection: %w", err)
	}
	defer func() { _ = db.Close() }()

	module := storagecleanup.New(db, storage.NewS3(storageCfg))
	return module.Service.Run(ctx)
}

// setupTelemetry sends traces and slog records over OTLP when their endpoints
// are set, as in cmd/api: each run becomes a trace, with its queries as child
// spans and its log lines linked to it. Call the returned func before exiting
// to send what is still buffered.
func setupTelemetry(ctx context.Context) (flush func(context.Context) error, err error) {
	var flushers []func(context.Context) error
	flush = func(ctx context.Context) error {
		var errs []error
		for _, f := range flushers {
			errs = append(errs, f(ctx))
		}
		return errors.Join(errs...)
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create trace exporter: %w", err)
		}
		provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
		otel.SetTracerProvider(provider)
		flushers = append(flushers, provider.Shutdown)
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") != "" {
		exporter, err := otlploghttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("create log exporter: %w", err)
		}
		provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
		otlpHandler := otelslog.NewHandler("rantau-go/cmd/cleanup", otelslog.WithLoggerProvider(provider))
		slog.SetDefault(slog.New(slog.NewMultiHandler(slog.Default().Handler(), otlpHandler)))
		flushers = append(flushers, provider.Shutdown)
	}

	return flush, nil
}
