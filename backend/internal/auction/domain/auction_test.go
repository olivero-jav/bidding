package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func validParams() NewAuctionParams {
	return NewAuctionParams{
		SellerID:     uuid.New(),
		Title:        "Charizard 1st edition",
		Description:  "PSA 9",
		Category:     "pokemon",
		StartPrice:   10000,
		MinIncrement: 500,
		Cap:          nil,
		EndsAt:       time.Now().Add(24 * time.Hour),
	}
}

func TestNewAuction_Valid(t *testing.T) {
	now := time.Now()
	a, err := NewAuction(validParams(), now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.Status != StatusActive {
		t.Errorf("expected status %q, got %q", StatusActive, a.Status)
	}
	if a.ID == uuid.Nil {
		t.Error("expected a generated id")
	}
	if a.CreatedAt.Location() != time.UTC {
		t.Error("expected CreatedAt in UTC")
	}
}

func TestNewAuction_Invariants(t *testing.T) {
	cap5000 := Money(5000)
	cap20000 := Money(20000)

	cases := []struct {
		name    string
		mutate  func(*NewAuctionParams)
		wantErr error
	}{
		{"empty title", func(p *NewAuctionParams) { p.Title = "" }, ErrEmptyTitle},
		{"zero increment", func(p *NewAuctionParams) { p.MinIncrement = 0 }, ErrNonPositiveIncrement},
		{"negative increment", func(p *NewAuctionParams) { p.MinIncrement = -1 }, ErrNonPositiveIncrement},
		{"negative start price", func(p *NewAuctionParams) { p.StartPrice = -1 }, ErrNegativeStartPrice},
		{"ends in past", func(p *NewAuctionParams) { p.EndsAt = time.Now().Add(-time.Hour) }, ErrEndsInPast},
		{"cap equal to start", func(p *NewAuctionParams) { c := Money(10000); p.Cap = &c }, ErrCapNotAboveStart},
		{"cap below start", func(p *NewAuctionParams) { p.Cap = &cap5000 }, ErrCapNotAboveStart},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validParams()
			tc.mutate(&p)
			_, err := NewAuction(p, time.Now())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}

	// A cap above the start price (Type B) is valid.
	t.Run("cap above start is valid", func(t *testing.T) {
		p := validParams()
		p.Cap = &cap20000
		if _, err := NewAuction(p, time.Now()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
