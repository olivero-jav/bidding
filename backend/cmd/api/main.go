package main

import (
	"context"
	"log"
	stdhttp "net/http"
	"time"

	auctionhttp "bidding/internal/auction/adapter/http"
	auctionpg "bidding/internal/auction/adapter/postgres"
	"bidding/internal/auction/app"
	"bidding/internal/platform"
	"bidding/internal/platform/config"
	platformpg "bidding/internal/platform/postgres"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := platformpg.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	repo := auctionpg.NewAuctionRepository(pool)
	bidRepo := auctionpg.NewBidRepository(pool)
	svc := app.NewService(repo, bidRepo, platform.SystemClock{})
	handler := auctionhttp.NewHandler(svc)

	srv := &stdhttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("auction api listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
