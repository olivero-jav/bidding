package port

import (
	"context"

	"github.com/google/uuid"

	"bidding/internal/auction/domain"
)

// AuctionRepository is the persistence port for auctions. The application layer
// depends on this interface, never on pgx or SQL: all database concerns live in
// the adapter that implements it. Methods speak in domain types.
type AuctionRepository interface {
	Create(ctx context.Context, a *domain.Auction) error
	// Get returns domain.ErrAuctionNotFound when no auction matches the id.
	Get(ctx context.Context, id uuid.UUID) (*domain.Auction, error)
	List(ctx context.Context) ([]*domain.Auction, error)
}
