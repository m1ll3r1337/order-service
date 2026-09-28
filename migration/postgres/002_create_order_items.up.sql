CREATE TABLE IF NOT EXISTS order_items (
    id BIGSERIAL NOT NULL UNIQUE,
    guid UUID NOT NULL PRIMARY KEY,
    order_guid UUID NOT NULL REFERENCES orders(guid) ON DELETE CASCADE,
    product_guid UUID NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_guid ON order_items(order_guid);
