// Command cleanup deletes one batch of storage objects queued for deletion and
// exits. Run it on a schedule, such as every 15 minutes from cron.
package main

import (
	"context"
	"log"

	"rantau-go/internal/config"
	"rantau-go/internal/database"
	"rantau-go/internal/module/storagecleanup"
	"rantau-go/internal/storage"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	db, err := database.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database connection: %v", err)
	}
	defer db.Close()

	module := storagecleanup.NewModule(db, storage.NewS3(cfg.Storage))
	if err := module.Service.Run(ctx); err != nil {
		log.Fatalf("clean up storage: %v", err)
	}
}
