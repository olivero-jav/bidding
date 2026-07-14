package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"bidding/internal/auction/adapter/postgres/sqlc"
	"bidding/internal/auction/domain"
)

// AuctionRepository implements port.AuctionRepository backed by Postgres via
// sqlc. All SQL and pgx types are confined to this package; the rest of the
// backend sees only domain types (that mapping is the seam that keeps the core
// database-agnostic).
type AuctionRepository struct {
	q *sqlc.Queries
}

func NewAuctionRepository(pool *pgxpool.Pool) *AuctionRepository {
	return &AuctionRepository{q: sqlc.New(pool)}
}

func (r *AuctionRepository) Create(ctx context.Context, a *domain.Auction) error {
	return r.q.CreateAuction(ctx, sqlc.CreateAuctionParams{
		ID:           toPgUUID(a.ID),
		SellerID:     toPgUUID(a.SellerID),
		Title:        a.Title,
		Description:  a.Description,
		Category:     a.Category,
		StartPrice:   int64(a.StartPrice),
		MinIncrement: int64(a.MinIncrement),
		Cap:          toNullableInt64(a.Cap),
		EndsAt:       toPgTimestamptz(a.EndsAt),
		Status:       string(a.Status),
		CreatedAt:    toPgTimestamptz(a.CreatedAt),
	})
}

func (r *AuctionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Auction, error) {
	row, err := r.q.GetAuction(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAuctionNotFound
		}
		return nil, err
	}
	return toDomain(row), nil
}

func (r *AuctionRepository) List(ctx context.Context) ([]*domain.Auction, error) {
	rows, err := r.q.ListAuctions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Auction, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

// --- mapping between domain and sqlc/pgx types ---

func toDomain(row sqlc.Auction) *domain.Auction {
	var cap *domain.Money
	if row.Cap != nil {
		m := domain.Money(*row.Cap)
		cap = &m
	}
	var highestAmount *domain.Money
	if row.HighestBidAmount != nil {
		m := domain.Money(*row.HighestBidAmount)
		highestAmount = &m
	}
	var highestBidder *uuid.UUID
	if row.HighestBidderID.Valid {
		u := uuid.UUID(row.HighestBidderID.Bytes)
		highestBidder = &u
	}
	return &domain.Auction{
		ID:           uuid.UUID(row.ID.Bytes),
		SellerID:     uuid.UUID(row.SellerID.Bytes),
		Title:        row.Title,
		Description:  row.Description,
		Category:     row.Category,
		StartPrice:   domain.Money(row.StartPrice),
		MinIncrement: domain.Money(row.MinIncrement),
		Cap:          cap,
		// pgx decodes timestamptz into the machine-local zone; normalize back to
		// UTC so the whole backend stays UTC-only (Chile time is a display concern).
		EndsAt:           row.EndsAt.Time.UTC(),
		Status:           domain.AuctionStatus(row.Status),
		CreatedAt:        row.CreatedAt.Time.UTC(),
		HighestBidAmount: highestAmount,
		HighestBidderID:  highestBidder,
	}
}

func toNullableUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toNullableInt64(m *domain.Money) *int64 {
	if m == nil {
		return nil
	}
	v := int64(*m)
	return &v
}
