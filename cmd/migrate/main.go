package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/pressly/goose/v3"

	"sakukota-go/internal/config"
	"sakukota-go/internal/database"
)

const defaultMigrationsDir = "migrations"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s {up|down|status|reset|version}", os.Args[0])
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		log.Fatal(err)
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = defaultMigrationsDir
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.NewPostgresDB(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer func() { _ = db.Close() }()

	var migrationErr error
	switch os.Args[1] {
	case "up":
		migrationErr = goose.UpContext(ctx, db.DB, migrationsDir)
	case "down":
		migrationErr = goose.DownContext(ctx, db.DB, migrationsDir)
	case "status":
		migrationErr = goose.StatusContext(ctx, db.DB, migrationsDir)
	case "reset":
		migrationErr = goose.ResetContext(ctx, db.DB, migrationsDir)
	case "version":
		migrationErr = goose.VersionContext(ctx, db.DB, migrationsDir)
	default:
		migrationErr = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if migrationErr != nil {
		log.Fatal(migrationErr)
	}
}
