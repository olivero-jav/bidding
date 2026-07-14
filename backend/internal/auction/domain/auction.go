package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// AuctionStatus is the lifecycle state of an auction. In slice 1 the only
// value is StatusActive; the closing slice introduces the real transitions.
type AuctionStatus string

const StatusActive AuctionStatus = "active"

// Auction is the aggregate root. It carries the publication fields (what the
// seller enters) plus the runtime state of the highest bid, kept on the row so
// the bid transaction can validate under a lock without scanning the bid log.
type Auction struct {
	ID           uuid.UUID
	SellerID     uuid.UUID
	Title        string
	Description  string
	Category     string
	StartPrice   Money
	MinIncrement Money
	Cap          *Money // nil = Type A; set = Type B (buy-it-now cap)
	EndsAt       time.Time
	Status       AuctionStatus
	CreatedAt    time.Time

	// Runtime state — both nil until the first accepted bid.
	HighestBidAmount *Money
	HighestBidderID  *uuid.UUID
}

// Publication invariants.
var (
	ErrEmptyTitle           = errors.New("auction: title must not be empty")
	ErrNonPositiveIncrement = errors.New("auction: min increment must be greater than zero")
	ErrNegativeStartPrice   = errors.New("auction: start price must not be negative")
	ErrEndsInPast           = errors.New("auction: ends_at must be in the future")
	ErrCapNotAboveStart     = errors.New("auction: cap must be greater than start price")
)

// ErrAuctionNotFound is returned by the repository when no auction matches.
var ErrAuctionNotFound = errors.New("auction: not found")

// NewAuctionParams carries the seller-provided fields at publication time.
type NewAuctionParams struct {
	SellerID     uuid.UUID
	Title        string
	Description  string
	Category     string
	StartPrice   Money
	MinIncrement Money
	Cap          *Money
	EndsAt       time.Time
}

// NewAuction validates the publication invariants and builds an active auction.
// now is the authoritative server clock: ends_at is checked against it, and the
// auction's timestamps are stored in UTC.
func NewAuction(p NewAuctionParams, now time.Time) (*Auction, error) {
	if p.Title == "" {
		return nil, ErrEmptyTitle
	}
	if p.MinIncrement <= 0 {
		return nil, ErrNonPositiveIncrement
	}
	if p.StartPrice < 0 {
		return nil, ErrNegativeStartPrice
	}
	if !p.EndsAt.After(now) {
		return nil, ErrEndsInPast
	}
	if p.Cap != nil && *p.Cap <= p.StartPrice {
		return nil, ErrCapNotAboveStart
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &Auction{
		ID:           id,
		SellerID:     p.SellerID,
		Title:        p.Title,
		Description:  p.Description,
		Category:     p.Category,
		StartPrice:   p.StartPrice,
		MinIncrement: p.MinIncrement,
		Cap:          p.Cap,
		EndsAt:       p.EndsAt.UTC(),
		Status:       StatusActive,
		CreatedAt:    now.UTC(),
	}, nil
}
