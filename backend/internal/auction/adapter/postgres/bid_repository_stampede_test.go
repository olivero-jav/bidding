package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"bidding/internal/auction/adapter/postgres/sqlc"
	"bidding/internal/auction/domain"
)

// TestPlaceBid_Stampede is the correctness proof for the concurrency core. It
// fires N concurrent, identical, valid bids at one auction against a real
// Postgres (the row lock is Postgres behavior, not something a mock can stand in
// for) and asserts the outcome is serialized: exactly one bid wins that price
// level, every other is rejected for not meeting the increment, and the auction
// row plus the bid ledger end up consistent.
func TestPlaceBid_Stampede(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stampede test (needs Docker) in -short mode")
	}

	ctx := context.Background()
	pool := startPostgres(ctx, t)

	auctionRepo := NewAuctionRepository(pool)
	bidRepo := NewBidRepository(pool)

	// One active auction: start price 1000, increment 100, no bids yet, so the
	// minimum first bid is exactly 1000.
	now := time.Now()
	auction, err := domain.NewAuction(domain.NewAuctionParams{
		SellerID:     uuid.New(),
		Title:        "Stampede subject",
		Category:     "pokemon",
		StartPrice:   1000,
		MinIncrement: 100,
		EndsAt:       now.Add(24 * time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("build auction: %v", err)
	}
	if err := auctionRepo.Create(ctx, auction); err != nil {
		t.Fatalf("create auction: %v", err)
	}

	const N = 30
	results := make([]error, N)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bidder := uuid.New()
			<-start // release all goroutines at once
			_, results[i] = bidRepo.PlaceBid(ctx, auction.ID, func(a *domain.Auction) (*domain.Bid, error) {
				return a.PlaceBid(bidder, 1000, now)
			})
		}(i)
	}
	close(start)
	wg.Wait()

	// Tally the outcomes.
	var won, rejected int
	for i, err := range results {
		switch {
		case err == nil:
			won++
		case errors.Is(err, domain.ErrBidBelowMinimum):
			rejected++
		default:
			t.Errorf("goroutine %d: unexpected error %v", i, err)
		}
	}

	if won != 1 {
		t.Errorf("expected exactly 1 winning bid, got %d", won)
	}
	if rejected != N-1 {
		t.Errorf("expected %d rejections for below-minimum, got %d", N-1, rejected)
	}

	// The auction row reflects exactly one accepted bid at 1000.
	got, err := auctionRepo.Get(ctx, auction.ID)
	if err != nil {
		t.Fatalf("reload auction: %v", err)
	}
	if got.HighestBidAmount == nil || *got.HighestBidAmount != 1000 {
		t.Errorf("highest_bid_amount = %v, want 1000", got.HighestBidAmount)
	}

	// The append-only ledger holds exactly one bid — no lost or phantom inserts.
	count, err := sqlc.New(pool).CountBidsForAuction(ctx, toPgUUID(auction.ID))
	if err != nil {
		t.Fatalf("count bids: %v", err)
	}
	if count != 1 {
		t.Errorf("bid ledger has %d rows, want 1", count)
	}
}

// startPostgres spins up an ephemeral Postgres with the schema applied from the
// real migration files, and returns a ready pool. Container and pool are torn
// down via t.Cleanup.
func startPostgres(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	scripts := migrationUpScripts(t)

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithInitScripts(scripts...),
		tcpostgres.WithDatabase("bidding"),
		tcpostgres.WithUsername("bidding"),
		tcpostgres.WithPassword("bidding"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// migrationUpScripts returns the absolute, ordered paths of the .up.sql files.
func migrationUpScripts(t *testing.T) []string {
	t.Helper()
	// This test lives in internal/auction/adapter/postgres; migrations are at the
	// backend root under db/migrations.
	dir := filepath.Join("..", "..", "..", "..", "db", "migrations")
	scripts, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(scripts) == 0 {
		t.Fatalf("no migration scripts found under %s", dir)
	}
	sort.Strings(scripts) // 0001_..., 0002_... apply in order
	abs := make([]string, len(scripts))
	for i, s := range scripts {
		p, err := filepath.Abs(s)
		if err != nil {
			t.Fatalf("abs path: %v", err)
		}
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("stat migration %s: %v", p, err)
		}
		abs[i] = p
	}
	return abs
}
