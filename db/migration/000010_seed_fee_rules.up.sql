INSERT INTO fee_rules (
    operation_type,
    currency,
    fee_type,
    flat_amount,
    percentage_bps,
    min_fee,
    max_fee,
    is_active
)
VALUES
    -- Transfers
    ('transfer', 'MAD', 'flat',       1000, 0,   0,    NULL, TRUE),
    ('transfer', 'USD', 'flat',        100, 0,   0,    NULL, TRUE),
    ('transfer', 'EUR', 'flat',        100, 0,   0,    NULL, TRUE),

    -- Currency conversions
    ('conversion', 'MAD', 'percentage', 0,  100, 500,  5000, TRUE),
    ('conversion', 'USD', 'percentage', 0,  100,  50,  1000, TRUE),
    ('conversion', 'EUR', 'percentage', 0,  100,  50,  1000, TRUE),

    -- Card operations
    ('card', 'MAD', 'flat',            2000, 0,   0,    NULL, TRUE),
    ('card', 'USD', 'flat',             200, 0,   0,    NULL, TRUE),
    ('card', 'EUR', 'flat',             200, 0,   0,    NULL, TRUE); 