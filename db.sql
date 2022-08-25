create table orders
(
    id           serial primary key,
    order_id     bigint,
    symbol       text,
    side         text,
    quantity     real,
    actual_price real,
    price        real,
    stop_price   real,
    status       text,
    type         text,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP
);