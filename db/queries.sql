-- name: InsertCountry :exec
INSERT OR IGNORE INTO countries (code, name, region, flag_emoji)
VALUES (?, ?, ?, ?);

-- name: GetCountries :many
SELECT code, name, region, flag_emoji, created_at
FROM countries
ORDER BY name;

-- name: GetCountriesByRegion :many
SELECT code, name, region, flag_emoji, created_at
FROM countries
WHERE region = ?
ORDER BY name;

-- name: InsertHoliday :exec
INSERT OR IGNORE INTO holidays (country_code, date, name, description, type, sub_region, year, source)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetHolidaysByCountryAndYear :many
SELECT id, country_code, date, name, description, type, sub_region, year, source, created_at
FROM holidays
WHERE country_code = ?
  AND year = ?
ORDER BY date;

-- name: GetHolidaysByRegionAndYear :many
SELECT h.id, h.country_code, h.date, h.name, h.description, h.type, h.sub_region, h.year, h.source, h.created_at
FROM holidays h
JOIN countries c ON c.code = h.country_code
WHERE c.region = ?
  AND h.year = ?
ORDER BY h.date, h.country_code;

-- name: GetHolidaysByCountryYearAndType :many
SELECT id, country_code, date, name, description, type, sub_region, year, source, created_at
FROM holidays
WHERE country_code = ?
  AND year = ?
  AND type = ?
ORDER BY date;

-- name: InsertSyncLog :exec
INSERT INTO sync_log (country_code, year, source, status, error_message)
VALUES (?, ?, ?, ?, ?);

-- name: GetLastSync :one
SELECT id, country_code, year, source, synced_at, status, error_message
FROM sync_log
WHERE country_code = ?
  AND year = ?
  AND status = 'success'
ORDER BY synced_at DESC
LIMIT 1;
