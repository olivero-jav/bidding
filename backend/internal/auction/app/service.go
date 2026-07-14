package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"bidding/internal/auction/domain"
	"bidding/internal/auction/port"
)

// Service holds the auction use cases. It orchestrates the domain and the
// repository ports; it knows nothing about HTTP or SQL.
type Service struct {
	repo  port.AuctionRepository
	bids  port.BidRepository
	clock port.Clock
}

func NewService(repo port.AuctionRepository, bids port.BidRepository, clock port.Clock) *Service {
	return &Service{repo: repo, bids: bids, clock: clock}
}

// CreateAuctionInput is the seller-provided payload to publish an auction.
type CreateAuctionInput struct {
	SellerID     uuid.UUID
	Title        string
	Description  string
	Category     string
	StartPrice   domain.Money
	MinIncrement domain.Money
	Cap          *domain.Money
	EndsAt       time.Time
}

// CreateAuction validates the publication invariants (in the domain) against
// the server clock and persists the new auction.
func (s *Service) CreateAuction(ctx context.Context, in CreateAuctionInput) (*domain.Auction, error) {
	a, err := domain.NewAuction(domain.NewAuctionParams{
		SellerID:     in.SellerID,
		Title:        in.Title,
		Description:  in.Description,
		Category:     in.Category,
		StartPrice:   in.StartPrice,
		MinIncrement: in.MinIncrement,
		Cap:          in.Cap,
		EndsAt:       in.EndsAt,
	}, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// PlaceBidInput is the payload to bid on an auction.
type PlaceBidInput struct {
	AuctionID uuid.UUID
	BidderID  uuid.UUID
	Amount    domain.Money
}

// BidResult is the outcome of an accepted bid: the recorded bid plus the
// auction's new state (current price, highest bidder) for the caller to display.
type BidResult struct {
	Bid     *domain.Bid
	Auction *domain.Auction
}

// PlaceBid runs the bid under the repository's row lock. The domain validation
// (active, within deadline, meets the minimum) executes inside the locked
// transaction via the decision callback, against the server clock.
func (s *Service) PlaceBid(ctx context.Context, in PlaceBidInput) (*BidResult, error) {
	now := s.clock.Now()
	var auction *domain.Auction
	bid, err := s.bids.PlaceBid(ctx, in.AuctionID, func(a *domain.Auction) (*domain.Bid, error) {
		auction = a
		return a.PlaceBid(in.BidderID, in.Amount, now)
	})
	if err != nil {
		return nil, err
	}
	return &BidResult{Bid: bid, Auction: auction}, nil
}

// GetAuction returns a single auction, or domain.ErrAuctionNotFound.
func (s *Service) GetAuction(ctx context.Context, id uuid.UUID) (*domain.Auction, error) {
	return s.repo.Get(ctx, id)
}

// ListAuctions returns all auctions, newest first.
func (s *Service) ListAuctions(ctx context.Context) ([]*domain.Auction, error) {
	return s.repo.List(ctx)
}
