CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,

    order_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    amount BIGINT NOT NULL CHECK (amount > 0),
    currency VARCHAR(10) NOT NULL,

    status VARCHAR(20) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    provider_transaction_id VARCHAR(255) NOT NULL,

    metadata JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_provider_tx UNIQUE (provider, provider_transaction_id)
);

CREATE TABLE IF NOT EXISTS payment_events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payments_provider_tx ON payments(provider_transaction_id);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();