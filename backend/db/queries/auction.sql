-- name: CreateAuction :exec
INSERT INTO auction (
    id, seller_id, title, description, category,
    start_price, min_increment, cap, ends_at, status, created_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11
);

-- name: GetAuction :one
SELECT * FROM auction WHERE id = $1;

-- name: GetAuctionForUpdate :one
-- Locks the auction row for the bid transaction. Blocks until the row is free,
-- serializing concurrent bids on the same auction into a total order.
SELECT * FROM auction WHERE id = $1 FOR UPDATE;

-- name: UpdateAuctionHighestBid :exec
UPDATE auction
SET highest_bid_amount = $2,
    highest_bidder_id  = $3
WHERE id = $1;

-- name: ListAuctions :many
SELECT * FROM auction ORDER BY created_at DESC;
