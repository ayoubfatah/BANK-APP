DELETE FROM fee_rules
WHERE (operation_type, currency) IN (
    ('transfer', 'MAD'),
    ('transfer', 'USD'),
    ('transfer', 'EUR'),
    ('conversion', 'MAD'),
    ('conversion', 'USD'),
    ('conversion', 'EUR'),
    ('card', 'MAD'),
    ('card', 'USD'),
    ('card', 'EUR')
);