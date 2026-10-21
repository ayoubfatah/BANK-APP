CREATE TABLE card_spending_limits (
    card_id BIGINT PRIMARY KEY
        REFERENCES cards(id) ON DELETE RESTRICT,

    daily_limit BIGINT CHECK (
        daily_limit IS NULL OR daily_limit > 0
    ),

    monthly_limit BIGINT CHECK (
        monthly_limit IS NULL OR monthly_limit > 0
    ),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);