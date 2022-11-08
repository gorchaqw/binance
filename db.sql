create table orders
(
    id           text primary key,
    order_id     bigint,
    session_id   text,
    symbol       text,
    side         text,
    quantity     real,
    actual_price real,
    price        real,
    stop_price   real,
    try          integer,
    status       text,
    type         text,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP
);

create table features_orders
(
    id           text primary key,
    order_id     bigint,
    session_id   text,
    symbol       text,
    side         text,
    quantity     real,
    actual_price real,
    price        real,
    stop_price   real,
    try          integer,
    status       text,
    type         text,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP
);