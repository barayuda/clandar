// Package seeder populates the database with priority countries and their
// public holidays, fetching from external APIs on first run.
package seeder

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/barayuda/clandar/internal/fetcher"
	"github.com/barayuda/clandar/internal/store"
)

// priorityCountry holds the static metadata for one of the 12 seed countries.
type priorityCountry struct {
	Code      string
	Name      string
	Region    string
	FlagEmoji string
}

// priorityCountries is the canonical list of countries Clandar tracks on launch (20 total).
var priorityCountries = []priorityCountry{
	{"AU", "Australia", "Asia-Pacific", "🇦🇺"},
	{"JP", "Japan", "Asia-Pacific", "🇯🇵"},
	{"CN", "China", "Asia-Pacific", "🇨🇳"},
	// ASEAN
	{"BN", "Brunei", "ASEAN", "🇧🇳"},
	{"KH", "Cambodia", "ASEAN", "🇰🇭"},
	{"ID", "Indonesia", "ASEAN", "🇮🇩"},
	{"LA", "Laos", "ASEAN", "🇱🇦"},
	{"MY", "Malaysia", "ASEAN", "🇲🇾"},
	{"MM", "Myanmar", "ASEAN", "🇲🇲"},
	{"PH", "Philippines", "ASEAN", "🇵🇭"},
	{"SG", "Singapore", "ASEAN", "🇸🇬"},
	{"TH", "Thailand", "ASEAN", "🇹🇭"},
	{"VN", "Vietnam", "ASEAN", "🇻🇳"},
	{"US", "United States", "Americas", "🇺🇸"},
	{"GB", "United Kingdom", "Europe", "🇬🇧"},
	{"DE", "Germany", "Europe", "🇩🇪"},
	{"FR", "France", "Europe", "🇫🇷"},
	{"IT", "Italy", "Europe", "🇮🇹"},
	{"ES", "Spain", "Europe", "🇪🇸"},
	{"NL", "Netherlands", "Europe", "🇳🇱"},
}

// chinaHolidays2026 contains the hardcoded Chinese public holidays for 2026.
// Nager.Date does not support CN; Calendarific will supplement if a key is set.
var chinaHolidays2026 = []store.Holiday{
	{CountryCode: "CN", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-02-17", Name: "Spring Festival / Chinese New Year", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-04-05", Name: "Qingming Festival", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-05-01", Name: "Labour Day", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-06-19", Name: "Dragon Boat Festival", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-09-25", Name: "Mid-Autumn Festival", Type: "public", Year: 2026, Source: "hardcoded"},
	{CountryCode: "CN", Date: "2026-10-01", Name: "National Day", Type: "public", Year: 2026, Source: "hardcoded"},
}

// Seeder orchestrates country seeding and holiday synchronisation.
type Seeder struct {
	Store   *store.Store
	Fetcher *fetcher.Fetcher
	Logger  zerolog.Logger
}

// New creates a Seeder.
func New(st *store.Store, f *fetcher.Fetcher, log zerolog.Logger) *Seeder {
	return &Seeder{Store: st, Fetcher: f, Logger: log}
}

// SeedCountries inserts all 20 priority countries into the countries table.
// Existing rows are silently skipped.
func (s *Seeder) SeedCountries(ctx context.Context) error {
	for _, c := range priorityCountries {
		if err := s.Store.InsertCountry(ctx, c.Code, c.Name, c.Region, c.FlagEmoji); err != nil {
			return fmt.Errorf("seeder: seed country %s: %w", c.Code, err)
		}
	}
	s.Logger.Info().Int("count", len(priorityCountries)).Msg("seeder: countries seeded")
	return nil
}

// SyncCountry fetches and stores holidays for a single country+year pair.
// If a successful sync already exists in sync_log the operation is skipped.
func (s *Seeder) SyncCountry(ctx context.Context, countryCode string, year int) error {
	last, err := s.Store.GetLastSync(ctx, countryCode, year)
	if err != nil {
		return fmt.Errorf("seeder: check last sync for %s/%d: %w", countryCode, year, err)
	}
	if last != nil && last.Status == "success" {
		s.Logger.Debug().
			Str("country", countryCode).
			Int("year", year).
			Time("last_sync", last.SyncedAt).
			Msg("seeder: already synced, skipping")
		return nil
	}

	holidays, fetchErr := s.fetchHolidays(ctx, countryCode, year)

	// Persist what we have even if Calendarific errored (fetchErr may be partial).
	insertCount := 0
	for _, h := range holidays {
		if insertErr := s.Store.InsertHoliday(ctx, h); insertErr != nil {
			s.Logger.Warn().
				Err(insertErr).
				Str("country", countryCode).
				Str("date", h.Date).
				Str("name", h.Name).
				Msg("seeder: insert holiday failed")
		} else {
			insertCount++
		}
	}

	status := "success"
	errMsg := ""
	if fetchErr != nil {
		status = "failed"
		errMsg = fetchErr.Error()
	}

	if logErr := s.Store.InsertSyncLog(ctx, countryCode, year, "combined", status, errMsg); logErr != nil {
		s.Logger.Warn().Err(logErr).Str("country", countryCode).Int("year", year).Msg("seeder: write sync log failed")
	}

	if fetchErr != nil {
		return fmt.Errorf("seeder: fetch holidays for %s/%d: %w", countryCode, year, fetchErr)
	}

	s.Logger.Info().
		Str("country", countryCode).
		Int("year", year).
		Int("holidays", insertCount).
		Msg("seeder: sync complete")
	return nil
}

// fetchHolidays delegates to the appropriate data source(s) for the given
// country and year. China (CN) uses hardcoded data for 2026 plus Calendarific
// if a key is set; all other countries use the combined Fetcher.
func (s *Seeder) fetchHolidays(ctx context.Context, countryCode string, year int) ([]store.Holiday, error) {
	if countryCode == "CN" {
		return s.fetchChina(ctx, year)
	}
	return s.Fetcher.FetchAll(ctx, countryCode, year)
}

// fetchChina returns Chinese public holidays. For 2026 it uses a hardcoded
// list; for other years it relies solely on Calendarific (empty if no key).
func (s *Seeder) fetchChina(ctx context.Context, year int) ([]store.Holiday, error) {
	var holidays []store.Holiday

	if year == 2026 {
		holidays = append(holidays, chinaHolidays2026...)
	}

	// Supplement with Calendarific for any year (handles non-2026 and enriches 2026).
	calHolidays, err := s.Fetcher.Calendarific.Fetch(ctx, "CN", year)
	if err != nil {
		// Non-fatal: return whatever we have plus the error.
		return holidays, fmt.Errorf("seeder: calendarific CN/%d: %w", year, err)
	}
	holidays = append(holidays, calHolidays...)
	return holidays, nil
}

// SyncAll seeds countries then syncs all 12 for the current and next year.
func (s *Seeder) SyncAll(ctx context.Context) error {
	if err := s.SeedCountries(ctx); err != nil {
		return fmt.Errorf("seeder: seed countries: %w", err)
	}

	currentYear := time.Now().UTC().Year()
	years := []int{currentYear, currentYear + 1}

	for _, c := range priorityCountries {
		for _, y := range years {
			if err := s.SyncCountry(ctx, c.Code, y); err != nil {
				// Log and continue — a single country failure should not abort everything.
				s.Logger.Error().
					Err(err).
					Str("country", c.Code).
					Int("year", y).
					Msg("seeder: sync country failed")
			}
		}
	}
	s.Logger.Info().Msg("seeder: SyncAll complete")
	return nil
}
