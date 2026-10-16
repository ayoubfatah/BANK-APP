CREATE TABLE fee_rules (
    id             bigserial PRIMARY KEY,
    operation_type text        NOT NULL CHECK (operation_type IN ('transfer', 'conversion', 'card')),
    currency       char(3)     NOT NULL,
    fee_type       text        NOT NULL CHECK (fee_type IN ('flat', 'percentage')),
    flat_amount    bigint      NOT NULL DEFAULT 0 CHECK (flat_amount >= 0),
    percentage_bps integer     NOT NULL DEFAULT 0 CHECK (percentage_bps BETWEEN 0 AND 10000),
    min_fee        bigint      NOT NULL DEFAULT 0 CHECK (min_fee >= 0),
    max_fee        bigint      CHECK (max_fee >= 0),
    is_active      boolean     NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX fee_rules_one_active_per_operation_currency
    ON fee_rules (operation_type, currency)
    WHERE is_active;