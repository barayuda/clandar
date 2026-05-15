CREATE TABLE IF NOT EXISTS countries (
    code       TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    region     TEXT NOT NULL,
    flag_emoji TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS holidays (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    country_code TEXT    NOT NULL REFERENCES countries(code),
    date         TEXT    NOT NULL,
    name         TEXT    NOT NULL,
    description  TEXT,
    type         TEXT    NOT NULL,
    sub_region   TEXT,
    year         INTEGER NOT NULL,
    source       TEXT,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(country_code, date, name)
);

CREATE TABLE IF NOT EXISTS sync_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    country_code  TEXT    NOT NULL,
    year          INTEGER NOT NULL,
    source        TEXT    NOT NULL,
    synced_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    status        TEXT    NOT NULL,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_holidays_country_year ON holidays(country_code, year);
CREATE INDEX IF NOT EXISTS idx_holidays_date         ON holidays(date);
CREATE INDEX IF NOT EXISTS idx_sync_log_country_year ON sync_log(country_code, year);
