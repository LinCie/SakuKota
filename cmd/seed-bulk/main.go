package main

import (
	"context"
	_ "embed"
	"log"
	"log/slog"
	"os"
	"time"

	"rantau-go/internal/config"
	"rantau-go/internal/database"
)

//go:embed seed.sql
var seedSQL string

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer func() { _ = db.Close() }()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin seed transaction: %v", err)
	}

	if _, err := tx.ExecContext(ctx, seedSQL); err != nil {
		_ = tx.Rollback()
		log.Fatalf("run seeds: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit seeds: %v", err)
	}

	log.Println("bulk seeds applied")
}
