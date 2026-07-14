DROP TABLE bid;

ALTER TABLE auction
    DROP COLUMN highest_bid_amount,
    DROP COLUMN highest_bidder_id;
