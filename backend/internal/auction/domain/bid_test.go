package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// activeAuction builds a fresh active auction (start 10000, increment 500, no
// bids yet) for the bid tests.
func activeAuction() *Auction {
	a, _ := NewAuction(validParams(), time.Now())
	return a
}

func TestMinimumBid(t *testing.T) {
	a := activeAuction() // start 10000, increment 500

	if got := a.MinimumBid(); got != 10000 {
		t.Errorf("with no bids, MinimumBid = %d, want 10000 (start price)", got)
	}

	highest := Money(12000)
	a.HighestBidAmount = &highest
	if got := a.MinimumBid(); got != 12500 {
		t.Errorf("with a highest bid, MinimumBid = %d, want 12500 (highest + increment)", got)
	}
}

func TestPlaceBid_Valid(t *testing.T) {
	a := activeAuction()
	bidder := uuid.New()
	now := time.Now()

	bid, err := a.PlaceBid(bidder, 10000, now) // exactly the start price
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if bid.Amount != 10000 || bid.BidderID != bidder || bid.AuctionID != a.ID {
		t.Errorf("unexpected bid: %+v", bid)
	}
	if bid.CreatedAt.Location() != time.UTC {
		t.Error("expected bid CreatedAt in UTC")
	}
	// The aggregate advanced to the new highest bid.
	if a.HighestBidAmount == nil || *a.HighestBidAmount != 10000 {
		t.Errorf("HighestBidAmount not advanced: %v", a.HighestBidAmount)
	}
	if a.HighestBidderID == nil || *a.HighestBidderID != bidder {
		t.Errorf("HighestBidderID not advanced: %v", a.HighestBidderID)
	}
}

func TestPlaceBid_SecondMustBeatIncrement(t *testing.T) {
	a := activeAuction()
	now := time.Now()

	if _, err := a.PlaceBid(uuid.New(), 10000, now); err != nil {
		t.Fatalf("first bid should win, got %v", err)
	}
	// Minimum is now 10500. Matching the current highest is not enough.
	if _, err := a.PlaceBid(uuid.New(), 10000, now); !errors.Is(err, ErrBidBelowMinimum) {
		t.Fatalf("expected ErrBidBelowMinimum, got %v", err)
	}
	if _, err := a.PlaceBid(uuid.New(), 10500, now); err != nil {
		t.Fatalf("a bid meeting the increment should win, got %v", err)
	}
}

func TestPlaceBid_Rejections(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(*Auction)
		bidder  uuid.UUID
		amount  Money
		now     time.Time
		wantErr error
	}{
		{
			name:    "below start price",
			amount:  9999,
			now:     time.Now(),
			wantErr: ErrBidBelowMinimum,
		},
		{
			name:    "non positive amount",
			amount:  0,
			now:     time.Now(),
			wantErr: ErrNonPositiveAmount,
		},
		{
			name:    "after deadline",
			amount:  10000,
			now:     time.Now().Add(48 * time.Hour), // ends_at is +24h
			wantErr: ErrAuctionEnded,
		},
		{
			name:    "not active",
			setup:   func(a *Auction) { a.Status = "closed" },
			amount:  10000,
			now:     time.Now(),
			wantErr: ErrAuctionNotActive,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := activeAuction()
			if tc.setup != nil {
				tc.setup(a)
			}
			_, err := a.PlaceBid(uuid.New(), tc.amount, tc.now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}
