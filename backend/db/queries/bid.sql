-- name: InsertBid :exec
INSERT INTO bid (id, auction_id, bidder_id, amount, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: CountBidsForAuction :one
SELECT count(*) FROM bid WHERE auction_id = $1;
