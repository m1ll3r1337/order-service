package main

import (
	"context"
	"log"

	"github.com/m1ll3r1337/order-service/internal/app/config"
	rhealth "github.com/m1ll3r1337/order-service/internal/app/handler/http/health"
	rprocessor "github.com/m1ll3r1337/order-service/internal/app/processor/http"
	rcpostgres "github.com/m1ll3r1337/order-service/internal/app/repository/conn/postgres"
)

func main() {
	config.Load()

	cfg := config.Root

	ctx := context.Background()

	db, err := rcpostgres.NewClient(ctx, cfg.Repository.Postgres)
	if err != nil {
		log.Fatalf("failed to create db client: %v", err)
	}

	log.Printf("connected to database: %s", cfg.Repository.Postgres.Name)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	healthHandler := rhealth.NewHandler()

	proc := rprocessor.NewHTTP(healthHandler, cfg.Processor.WebServer)

	if err := proc.Serve(); err != nil {
		log.Printf("failed to start the server: %v", err)
	}
}
