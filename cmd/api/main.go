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

	"rantau-go/internal/config"
	"rantau-go/internal/database"
	"rantau-go/internal/router"
)

func main() {
	// Also the destination of the log package, so chi's request log and
	// log.Printf share the format of slog.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgresDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer func() { _ = db.Close() }()
	expvar.Publish("dbstats", expvar.Func(func() any { return db.Stats() }))

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
