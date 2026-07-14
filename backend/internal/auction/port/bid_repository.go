package port

import (
	"context"

	"github.com/google/uuid"

	"bidding/internal/auction/domain"
)

// BidDecision runs inside the bid transaction, once the auction row is locked.
// It receives the locked aggregate and returns the bid to persist, or an error
// to reject and roll back. This is where the domain validation happens, so it
// runs under the lock — never before it.
type BidDecision func(a *domain.Auction) (*domain.Bid, error)

// BidRepository is the persistence port for placing bids. Its single method owns
// the critical section: it loads the auction FOR UPDATE, calls decide with the
// locked aggregate, and — if decide returns a bid — appends the bid and writes
// the auction's advanced runtime state, all in one transaction. The lock, the
// transaction and lock_timeout are the adapter's concern; the app only supplies
// the decision.
type BidRepository interface {
	// PlaceBid returns domain.ErrAuctionNotFound if the auction does not exist,
	// domain.ErrBiddingUnavailable if the row lock could not be acquired in time,
	// or whatever error decide returned.
	PlaceBid(ctx context.Context, auctionID uuid.UUID, decide BidDecision) (*domain.Bid, error)
}
