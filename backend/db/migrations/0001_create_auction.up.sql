CREATE TABLE auction (
    id            uuid        PRIMARY KEY,
    seller_id     uuid        NOT NULL,
    title         text        NOT NULL,
    description   text        NOT NULL DEFAULT '',
    category      text        NOT NULL DEFAULT '',
    start_price   bigint      NOT NULL,
    min_increment bigint      NOT NULL,
    cap           bigint,
    ends_at       timestamptz NOT NULL,
    status        text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT start_price_non_negative CHECK (start_price >= 0),
    CONSTRAINT min_increment_positive   CHECK (min_increment > 0),
    CONSTRAINT cap_above_start          CHECK (cap IS NULL OR cap > start_price)
);
