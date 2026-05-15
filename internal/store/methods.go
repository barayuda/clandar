package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// sqliteTimeFormats lists the formats SQLite uses for DATETIME columns. We try
// each in order when scanning a DATETIME string into a time.Time value.
var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseTime parses a SQLite DATETIME string into time.Time.
// Returns the zero time if s is empty.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, format := range sqliteTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// InsertCountry inserts a country record into the countries table.
// Uses INSERT OR IGNORE so duplicate codes are silently skipped.
func (s *Store) InsertCountry(ctx context.Context, code, name, region, flagEmoji string) error {
	const q = `
		INSERT OR IGNORE INTO countries (code, name, region, flag_emoji)
		VALUES (?, ?, ?, ?)`
	if _, err := s.DB.ExecContext(ctx, q, code, name, region, flagEmoji); err != nil {
		return fmt.Errorf("store: insert country %q: %w", code, err)
	}
	return nil
}

// GetCountries returns all countries ordered by name.
func (s *Store) GetCountries(ctx context.Context) ([]Country, error) {
	const q = `SELECT code, name, region, flag_emoji, created_at FROM countries ORDER BY name`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: get countries: %w", err)
	}
	defer rows.Close()
	return scanCountries(rows)
}

// GetCountriesByRegion returns all countries in the given region, ordered by name.
func (s *Store) GetCountriesByRegion(ctx context.Context, region string) ([]Country, error) {
	const q = `SELECT code, name, region, flag_emoji, created_at FROM countries WHERE region = ? ORDER BY name`
	rows, err := s.DB.QueryContext(ctx, q, region)
	if err != nil {
		return nil, fmt.Errorf("store: get countries by region %q: %w", region, err)
	}
	defer rows.Close()
	return scanCountries(rows)
}

// InsertHoliday inserts a holiday record into the holidays table.
// Uses INSERT OR IGNORE so duplicate (country_code, date, name) rows are silently skipped.
func (s *Store) InsertHoliday(ctx context.Context, h Holiday) error {
	const q = `
		INSERT OR IGNORE INTO holidays
			(country_code, date, name, description, type, sub_region, year, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := s.DB.ExecContext(ctx, q,
		h.CountryCode, h.Date, h.Name, h.Description,
		h.Type, h.SubRegion, h.Year, h.Source,
	); err != nil {
		return fmt.Errorf("store: insert holiday %q %q: %w", h.CountryCode, h.Date, err)
	}
	return nil
}

// GetHolidaysByCountryAndYear returns all holidays for a given country and year.
func (s *Store) GetHolidaysByCountryAndYear(ctx context.Context, countryCode string, year int) ([]Holiday, error) {
	const q = `
		SELECT id, country_code, date, name, description, type, sub_region, year, source, created_at
		FROM holidays
		WHERE country_code = ? AND year = ?
		ORDER BY date`
	rows, err := s.DB.QueryContext(ctx, q, countryCode, year)
	if err != nil {
		return nil, fmt.Errorf("store: get holidays by country %q year %d: %w", countryCode, year, err)
	}
	defer rows.Close()
	return scanHolidays(rows)
}

// GetHolidaysByRegionAndYear returns all holidays for countries in a given region and year.
func (s *Store) GetHolidaysByRegionAndYear(ctx context.Context, region string, year int) ([]Holiday, error) {
	const q = `
		SELECT h.id, h.country_code, h.date, h.name, h.description, h.type, h.sub_region, h.year, h.source, h.created_at
		FROM holidays h
		JOIN countries c ON c.code = h.country_code
		WHERE c.region = ? AND h.year = ?
		ORDER BY h.date`
	rows, err := s.DB.QueryContext(ctx, q, region, year)
	if err != nil {
		return nil, fmt.Errorf("store: get holidays by region %q year %d: %w", region, year, err)
	}
	defer rows.Close()
	return scanHolidays(rows)
}

// GetHolidaysByCountryYearAndType returns holidays filtered by country, year, and type.
func (s *Store) GetHolidaysByCountryYearAndType(ctx context.Context, countryCode string, year int, holidayType string) ([]Holiday, error) {
	const q = `
		SELECT id, country_code, date, name, description, type, sub_region, year, source, created_at
		FROM holidays
		WHERE country_code = ? AND year = ? AND type = ?
		ORDER BY date`
	rows, err := s.DB.QueryContext(ctx, q, countryCode, year, holidayType)
	if err != nil {
		return nil, fmt.Errorf("store: get holidays by country %q year %d type %q: %w", countryCode, year, holidayType, err)
	}
	defer rows.Close()
	return scanHolidays(rows)
}

// InsertSyncLog records the result of a sync operation.
func (s *Store) InsertSyncLog(ctx context.Context, countryCode string, year int, source, status, errMsg string) error {
	const q = `
		INSERT INTO sync_log (country_code, year, source, status, error_message)
		VALUES (?, ?, ?, ?, ?)`
	if _, err := s.DB.ExecContext(ctx, q, countryCode, year, source, status, errMsg); err != nil {
		return fmt.Errorf("store: insert sync log for %q %d: %w", countryCode, year, err)
	}
	return nil
}

// GetLastSync returns the most recent sync log entry for a country+year pair,
// or nil if no sync has been recorded yet.
func (s *Store) GetLastSync(ctx context.Context, countryCode string, year int) (*SyncLog, error) {
	const q = `
		SELECT id, country_code, year, source, synced_at, status, error_message
		FROM sync_log
		WHERE country_code = ? AND year = ?
		ORDER BY synced_at DESC
		LIMIT 1`
	row := s.DB.QueryRowContext(ctx, q, countryCode, year)

	var sl SyncLog
	var syncedAt string
	var errMsg sql.NullString
	err := row.Scan(
		&sl.ID, &sl.CountryCode, &sl.Year, &sl.Source,
		&syncedAt, &sl.Status, &errMsg,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get last sync for %q %d: %w", countryCode, year, err)
	}
	sl.SyncedAt = parseTime(syncedAt)
	sl.ErrorMessage = errMsg.String
	return &sl, nil
}

// GetAllHolidaysByYear returns all holidays across all countries for a given year,
// ordered by date then country code.
func (s *Store) GetAllHolidaysByYear(ctx context.Context, year int) ([]Holiday, error) {
	const q = `
		SELECT id, country_code, date, name, description, type, sub_region, year, source, created_at
		FROM holidays
		WHERE year = ?
		ORDER BY date, country_code`
	rows, err := s.DB.QueryContext(ctx, q, year)
	if err != nil {
		return nil, fmt.Errorf("store: get all holidays year %d: %w", year, err)
	}
	defer rows.Close()
	return scanHolidays(rows)
}

// GetRegionCounts returns the number of countries per region.
func (s *Store) GetRegionCounts(ctx context.Context) (map[string]int, error) {
	const q = `SELECT region, COUNT(*) FROM countries GROUP BY region`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: get region counts: %w", err)
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var region string
		var count int
		if err := rows.Scan(&region, &count); err != nil {
			return nil, fmt.Errorf("store: scan region count: %w", err)
		}
		counts[region] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate region counts: %w", err)
	}
	return counts, nil
}

// --- helpers ---

func scanCountries(rows *sql.Rows) ([]Country, error) {
	var out []Country
	for rows.Next() {
		var c Country
		var createdAt string
		if err := rows.Scan(&c.Code, &c.Name, &c.Region, &c.FlagEmoji, &createdAt); err != nil {
			return nil, fmt.Errorf("store: scan country: %w", err)
		}
		c.CreatedAt = parseTime(createdAt)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate countries: %w", err)
	}
	return out, nil
}

func scanHolidays(rows *sql.Rows) ([]Holiday, error) {
	var out []Holiday
	for rows.Next() {
		var h Holiday
		var desc, subRegion sql.NullString
		var createdAt string
		if err := rows.Scan(
			&h.ID, &h.CountryCode, &h.Date, &h.Name,
			&desc, &h.Type, &subRegion, &h.Year, &h.Source, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan holiday: %w", err)
		}
		h.Description = desc.String
		h.SubRegion = subRegion.String
		h.CreatedAt = parseTime(createdAt)
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate holidays: %w", err)
	}
	return out, nil
}
