CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    source_id TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    total REAL,
    status TEXT NOT NULL,
    notes TEXT,
    scheduled_for TEXT,
    restaurant TEXT,
    order_platform TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (source, source_id)
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_updated ON orders (updated_at);

CREATE TABLE IF NOT EXISTS line_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    item_name TEXT NOT NULL,
    source_line_id TEXT,
    category TEXT,
    price REAL,
    status TEXT
);

CREATE INDEX IF NOT EXISTS idx_line_items_order ON line_items (order_id);

CREATE TABLE IF NOT EXISTS order_events (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    payload TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_events_order ON order_events (order_id);

CREATE TABLE IF NOT EXISTS ingest_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_line_items_order_source_line
    ON line_items (order_id, source_line_id);
