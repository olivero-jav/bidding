package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"bidding/internal/auction/adapter/postgres/sqlc"
	"bidding/internal/auction/domain"
	"bidding/internal/auction/port"
)

// lockTimeout bounds how long a bid waits for the auction row lock. It is a
// safety valve, not a tuning knob: without it a pathological hold could pin a
// pool connection indefinitely. On expiry Postgres raises lock_not_available
// (55P03), which we surface as a retryable error — the server does not retry.
const lockTimeout = "3s"

// pgLockNotAvailable is the SQLSTATE Postgres returns when lock_timeout fires.
const pgLockNotAvailable = "55P03"

// BidRepository implements port.BidRepository. It owns the pool directly (not
// just *sqlc.Queries) because placing a bid spans a transaction: several
// statements under one FOR UPDATE lock.
type BidRepository struct {
	pool *pgxpool.Pool
}

func NewBidRepository(pool *pgxpool.Pool) *BidRepository {
	return &BidRepository{pool: pool}
}

func (r *BidRepository) PlaceBid(ctx context.Context, auctionID uuid.UUID, decide port.BidDecision) (*domain.Bid, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// Rollback is a no-op once Commit has succeeded; safe to always defer.
	defer tx.Rollback(ctx)

	// SET LOCAL scopes the timeout to this transaction only.
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '"+lockTimeout+"'"); err != nil {
		return nil, err
	}

	q := sqlc.New(tx)

	row, err := q.GetAuctionForUpdate(ctx, toPgUUID(auctionID))
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, domain.ErrAuctionNotFound
		case isLockTimeout(err):
			return nil, domain.ErrBiddingUnavailable
		default:
			return nil, err
		}
	}

	auction := toDomain(row)

	// Domain validation runs here, under the lock, against the locked row.
	bid, err := decide(auction)
	if err != nil {
		return nil, err
	}

	if err := q.InsertBid(ctx, sqlc.InsertBidParams{
		ID:        toPgUUID(bid.ID),
		AuctionID: toPgUUID(bid.AuctionID),
		BidderID:  toPgUUID(bid.BidderID),
		Amount:    int64(bid.Amount),
		CreatedAt: toPgTimestamptz(bid.CreatedAt),
	}); err != nil {
		return nil, err
	}

	// decide advanced the aggregate's runtime state; persist it.
	if err := q.UpdateAuctionHighestBid(ctx, sqlc.UpdateAuctionHighestBidParams{
		ID:               toPgUUID(auction.ID),
		HighestBidAmount: toNullableInt64(auction.HighestBidAmount),
		HighestBidderID:  toNullableUUID(auction.HighestBidderID),
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return bid, nil
}

func isLockTimeout(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgLockNotAvailable
}
