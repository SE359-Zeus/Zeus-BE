CREATE TABLE IF NOT EXISTS carriers (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL UNIQUE
);

INSERT INTO carriers (name, code) VALUES
  ('DHL Express',   'DHL'),
  ('FedEx',         'FEDEX'),
  ('UPS',           'UPS'),
  ('USPS',          'USPS'),
  ('Maersk',        'MAERSK'),
  ('Evergreen',     'EVERGREEN'),
  ('MSC',           'MSC'),
  ('CEVA Logistics','CEVA'),
  ('Kuehne+Nagel',  'KN'),
  ('DB Schenker',   'DBS');
