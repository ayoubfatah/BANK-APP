DELETE FROM banking_partners
WHERE provider_name IN (
    'Demo SEPA Provider',
    'Demo UK Payments Provider',
    'Demo US Payments Provider',
    'Demo Morocco Payments Provider',
    'Demo Canada Payments Provider',
    'Demo Japan Payments Provider'
);