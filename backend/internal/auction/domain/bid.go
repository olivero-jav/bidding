package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Bid is one accepted offer on an auction. Bids are append-only: once accepted a
// bid is final (no retract in the MVP), so this struct is built once and never
// mutated.
type Bid struct {
	ID        uuid.UUID
	AuctionID uuid.UUID
	BidderID  uuid.UUID
	Amount    Money
	CreatedAt time.Time
}

// Bid rejection reasons. These describe a conflict with the auction's current
// state (not malformed input), except ErrNonPositiveAmount.
var (
	ErrAuctionNotActive   = errors.New("bid: auction is not active")
	ErrAuctionEnded       = errors.New("bid: auction has already ended")
	ErrBidBelowMinimum    = errors.New("bid: amount is below the minimum next bid")
	ErrNonPositiveAmount  = errors.New("bid: amount must be greater than zero")
	ErrBiddingUnavailable = errors.New("bid: auction is momentarily busy, retry")
)

// MinimumBid is the smallest amount the next bid may have: the start price while
// there are no bids, otherwise the current highest plus the minimum increment.
func (a *Auction) MinimumBid() Money {
	if a.HighestBidAmount != nil {
		return *a.HighestBidAmount + a.MinIncrement
	}
	return a.StartPrice
}

// PlaceBid validates amount against the auction's current state and, if it wins,
// builds the Bid and advances the auction's runtime state (highest bid/bidder).
// It must be called on an auction row already locked FOR UPDATE: the checks read
// HighestBidAmount, so the lock is what makes concurrent bids serialize.
//
// now is the authoritative server clock; the client's clock is never trusted.
func (a *Auction) PlaceBid(bidderID uuid.UUID, amount Money, now time.Time) (*Bid, error) {
	if amount <= 0 {
		return nil, ErrNonPositiveAmount
	}
	if a.Status != StatusActive {
		return nil, ErrAuctionNotActive
	}
	if now.After(a.EndsAt) {
		return nil, ErrAuctionEnded
	}
	if amount < a.MinimumBid() {
		return nil, ErrBidBelowMinimum
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	bid := &Bid{
		ID:        id,
		AuctionID: a.ID,
		BidderID:  bidderID,
		Amount:    amount,
		CreatedAt: now.UTC(),
	}

	// Advance the aggregate so the adapter persists the new current price. Copy
	// into fresh locals to avoid aliasing the caller's arguments.
	winningAmount := amount
	winningBidder := bidderID
	a.HighestBidAmount = &winningAmount
	a.HighestBidderID = &winningBidder

	return bid, nil
}
