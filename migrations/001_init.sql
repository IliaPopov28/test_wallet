CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    balance DECIMAL(15, 2) NOT NULL DEFAULT 0 CHECK (balance >= 0)
);

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    type VARCHAR(10) NOT NULL CHECK (type IN ('DEPOSIT', 'WITHDRAW')),
    amount DECIMAL(15, 2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_transactions_wallet ON transactions(wallet_id); 