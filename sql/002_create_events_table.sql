CREATE TABLE processed_events (
    event_id VARCHAR(50) PRIMARY KEY,
    card_id VARCHAR(50) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);