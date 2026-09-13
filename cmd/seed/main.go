package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/rantaugo?sslmode=disable"

//go:embed seed.sql
var seedSQL string

func main() {
	databaseURL := getenv("DATABASE_URL", defaultDatabaseURL)

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

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

	log.Println("seeds applied")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
