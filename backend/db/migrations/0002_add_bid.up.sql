-- Runtime state on the auction row: the current highest bid. Denormalized on
-- purpose — the bid transaction reads and writes it under FOR UPDATE, so the
-- validation never needs MAX() over the bid log. Both NULL until the first bid.
ALTER TABLE auction
    ADD COLUMN highest_bid_amount bigint,
    ADD COLUMN highest_bidder_id  uuid;

-- Append-only ledger of every accepted bid. No updates, no deletes: a bid is
-- final (retracts are not allowed in the MVP). It is the audit trail; the
-- authoritative "current price" lives on auction.highest_bid_amount.
CREATE TABLE bid (
    id         uuid        PRIMARY KEY,
    auction_id uuid        NOT NULL REFERENCES auction (id),
    bidder_id  uuid        NOT NULL,
    amount     bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT bid_amount_positive CHECK (amount > 0)
);

CREATE INDEX bid_auction_id_idx ON bid (auction_id);
