INSERT INTO exchange_rates (base_currency, quote_currency, rate, source, fetched_at)
VALUES
  -- USD base
  ('USD', 'EUR', 0.9200000000, 'manual_seed', now()),
  ('USD', 'GBP', 0.7800000000, 'manual_seed', now()),
  ('USD', 'MAD', 9.9500000000, 'manual_seed', now()),
  ('USD', 'JPY', 149.5000000000, 'manual_seed', now()),
  ('USD', 'CAD', 1.3600000000, 'manual_seed', now()),

  -- EUR base
  ('EUR', 'USD', 1.0869565217, 'manual_seed', now()),
  ('EUR', 'GBP', 0.8478260870, 'manual_seed', now()),
  ('EUR', 'MAD', 10.8152173913, 'manual_seed', now()),
  ('EUR', 'JPY', 162.5000000000, 'manual_seed', now()),
  ('EUR', 'CAD', 1.4782608696, 'manual_seed', now()),

  -- GBP base
  ('GBP', 'USD', 1.2820512821, 'manual_seed', now()),
  ('GBP', 'EUR', 1.1794871795, 'manual_seed', now()),
  ('GBP', 'MAD', 12.7564102564, 'manual_seed', now()),
  ('GBP', 'JPY', 191.6666666667, 'manual_seed', now()),
  ('GBP', 'CAD', 1.7435897436, 'manual_seed', now()),

  -- MAD base
  ('MAD', 'USD', 0.1005025126, 'manual_seed', now()),
  ('MAD', 'EUR', 0.0924623116, 'manual_seed', now()),
  ('MAD', 'GBP', 0.0783919598, 'manual_seed', now()),
  ('MAD', 'JPY', 15.0251256281, 'manual_seed', now()),
  ('MAD', 'CAD', 0.1366834171, 'manual_seed', now()),

  -- JPY base
  ('JPY', 'USD', 0.0066889632, 'manual_seed', now()),
  ('JPY', 'EUR', 0.0061538462, 'manual_seed', now()),
  ('JPY', 'GBP', 0.0052173913, 'manual_seed', now()),
  ('JPY', 'MAD', 0.0665551839, 'manual_seed', now()),
  ('JPY', 'CAD', 0.0090969900, 'manual_seed', now()),

  -- CAD base
  ('CAD', 'USD', 0.7352941176, 'manual_seed', now()),
  ('CAD', 'EUR', 0.6764705882, 'manual_seed', now()),
  ('CAD', 'GBP', 0.5735294118, 'manual_seed', now()),
  ('CAD', 'MAD', 7.3161764706, 'manual_seed', now()),
  ('CAD', 'JPY', 109.9264705882, 'manual_seed', now());