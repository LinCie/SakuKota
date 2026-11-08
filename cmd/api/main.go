package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
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

	db, err := database.NewDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer db.Close()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router.NewRouter(db, cfg),

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
