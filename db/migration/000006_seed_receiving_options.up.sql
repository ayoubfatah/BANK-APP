-- Seed default receiving-account provider configurations.
-- Fictional providers for development/testing only.

INSERT INTO banking_partners (
    currency_code,
    country_code,
    provider_name,
    detail_type,
    is_active
)
VALUES
    -- EUR: Eurozone
    ('EUR', 'FR', 'Demo SEPA Provider', 'IBAN', TRUE),
    ('EUR', 'DE', 'Demo SEPA Provider', 'IBAN', TRUE),
    ('EUR', 'ES', 'Demo SEPA Provider', 'IBAN', TRUE),
    ('EUR', 'NL', 'Demo SEPA Provider', 'IBAN', TRUE),

    -- GBP: United Kingdom
    ('GBP', 'GB', 'Demo UK Payments Provider', 'ACCOUNT_NUMBER', TRUE),
    ('GBP', 'GB', 'Demo UK Payments Provider', 'ROUTING_NUMBER', TRUE),

    -- USD: United States
    ('USD', 'US', 'Demo US Payments Provider', 'ACCOUNT_NUMBER', TRUE),
    ('USD', 'US', 'Demo US Payments Provider', 'ROUTING_NUMBER', TRUE),

    -- MAD: Morocco
    ('MAD', 'MA', 'Demo Morocco Payments Provider', 'ACCOUNT_NUMBER', TRUE),

    -- CAD: Canada
    ('CAD', 'CA', 'Demo Canada Payments Provider', 'ACCOUNT_NUMBER', TRUE),

    -- JPY: Japan
    ('JPY', 'JP', 'Demo Japan Payments Provider', 'ACCOUNT_NUMBER', TRUE)

ON CONFLICT (currency_code, country_code, provider_name)
DO NOTHING;