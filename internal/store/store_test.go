package store

import (
	"context"
	"os"
	"testing"
)

// openTestStore opens an in-memory SQLite store using the real schema file.
// The store is automatically closed when the test completes.
func openTestStore(t *testing.T) *Store {
	t.Helper()

	schemaBytes, err := os.ReadFile("../../db/schema.sql")
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}

	st, err := Open(":memory:", string(schemaBytes))
	if err != nil {
		t.Fatalf("open in-memory store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := st.Close(); closeErr != nil {
			t.Errorf("close store: %v", closeErr)
		}
	})
	return st
}

func TestInsertAndGetCountry(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry: %v", err)
	}

	countries, err := st.GetCountries(ctx)
	if err != nil {
		t.Fatalf("GetCountries: %v", err)
	}

	found := false
	for _, c := range countries {
		if c.Code == "AU" {
			found = true
			if c.Name != "Australia" {
				t.Errorf("Name = %q, want Australia", c.Name)
			}
			if c.Region != "Asia-Pacific" {
				t.Errorf("Region = %q, want Asia-Pacific", c.Region)
			}
			if c.FlagEmoji != "🇦🇺" {
				t.Errorf("FlagEmoji = %q, want 🇦🇺", c.FlagEmoji)
			}
		}
	}
	if !found {
		t.Error("country AU not found in GetCountries result")
	}
}

func TestInsertHolidayIgnoresDuplicate(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry: %v", err)
	}

	h := Holiday{
		CountryCode: "AU",
		Date:        "2026-01-01",
		Name:        "New Year's Day",
		Type:        "public",
		Year:        2026,
		Source:      "nager",
	}

	if err := st.InsertHoliday(ctx, h); err != nil {
		t.Fatalf("first InsertHoliday: %v", err)
	}
	// Second insert of same (country_code, date, name) should be silently ignored.
	if err := st.InsertHoliday(ctx, h); err != nil {
		t.Fatalf("second InsertHoliday (duplicate): %v", err)
	}

	holidays, err := st.GetHolidaysByCountryAndYear(ctx, "AU", 2026)
	if err != nil {
		t.Fatalf("GetHolidaysByCountryAndYear: %v", err)
	}
	if len(holidays) != 1 {
		t.Errorf("expected 1 holiday after duplicate insert, got %d", len(holidays))
	}
}

func TestGetHolidaysByCountryAndYear(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry: %v", err)
	}

	holidays := []Holiday{
		{CountryCode: "AU", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "AU", Date: "2026-04-25", Name: "Anzac Day", Type: "public", Year: 2026, Source: "nager"},
	}
	for _, h := range holidays {
		if err := st.InsertHoliday(ctx, h); err != nil {
			t.Fatalf("InsertHoliday %q: %v", h.Name, err)
		}
	}

	got, err := st.GetHolidaysByCountryAndYear(ctx, "AU", 2026)
	if err != nil {
		t.Fatalf("GetHolidaysByCountryAndYear: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 holidays, got %d", len(got))
	}
	if got[0].Date != "2026-01-01" {
		t.Errorf("got[0].Date = %q, want 2026-01-01", got[0].Date)
	}
	if got[1].Date != "2026-04-25" {
		t.Errorf("got[1].Date = %q, want 2026-04-25", got[1].Date)
	}
}

func TestGetHolidaysByRegionAndYear(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry AU: %v", err)
	}
	if err := st.InsertCountry(ctx, "DE", "Germany", "Europe", "🇩🇪"); err != nil {
		t.Fatalf("InsertCountry DE: %v", err)
	}

	hs := []Holiday{
		{CountryCode: "AU", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "DE", Date: "2026-12-25", Name: "Christmas Day", Type: "public", Year: 2026, Source: "nager"},
	}
	for _, h := range hs {
		if err := st.InsertHoliday(ctx, h); err != nil {
			t.Fatalf("InsertHoliday %q: %v", h.Name, err)
		}
	}

	got, err := st.GetHolidaysByRegionAndYear(ctx, "Asia-Pacific", 2026)
	if err != nil {
		t.Fatalf("GetHolidaysByRegionAndYear: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 holiday for Asia-Pacific, got %d", len(got))
	}
	if got[0].CountryCode != "AU" {
		t.Errorf("expected AU holiday, got country code %q", got[0].CountryCode)
	}
}

func TestGetAllHolidaysByYear(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry AU: %v", err)
	}
	if err := st.InsertCountry(ctx, "SG", "Singapore", "ASEAN", "🇸🇬"); err != nil {
		t.Fatalf("InsertCountry SG: %v", err)
	}

	hs := []Holiday{
		{CountryCode: "AU", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "SG", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
	}
	for _, h := range hs {
		if err := st.InsertHoliday(ctx, h); err != nil {
			t.Fatalf("InsertHoliday %q/%q: %v", h.CountryCode, h.Name, err)
		}
	}

	got, err := st.GetAllHolidaysByYear(ctx, 2026)
	if err != nil {
		t.Fatalf("GetAllHolidaysByYear: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 holidays, got %d", len(got))
	}

	codes := map[string]bool{}
	for _, h := range got {
		codes[h.CountryCode] = true
	}
	if !codes["AU"] || !codes["SG"] {
		t.Errorf("expected both AU and SG holidays, got country codes: %v", codes)
	}
}

func TestGetRegionCounts(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertCountry(ctx, "AU", "Australia", "Asia-Pacific", "🇦🇺"); err != nil {
		t.Fatalf("InsertCountry AU: %v", err)
	}
	if err := st.InsertCountry(ctx, "SG", "Singapore", "ASEAN", "🇸🇬"); err != nil {
		t.Fatalf("InsertCountry SG: %v", err)
	}

	counts, err := st.GetRegionCounts(ctx)
	if err != nil {
		t.Fatalf("GetRegionCounts: %v", err)
	}
	if counts["Asia-Pacific"] != 1 {
		t.Errorf("Asia-Pacific count = %d, want 1", counts["Asia-Pacific"])
	}
	if counts["ASEAN"] != 1 {
		t.Errorf("ASEAN count = %d, want 1", counts["ASEAN"])
	}
}

func TestGetLastSync_NoRows(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	sl, err := st.GetLastSync(ctx, "AU", 2026)
	if err != nil {
		t.Fatalf("GetLastSync: %v", err)
	}
	if sl != nil {
		t.Errorf("expected nil for no-rows case, got %+v", sl)
	}
}

func TestSyncLog(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.InsertSyncLog(ctx, "AU", 2026, "nager", "success", ""); err != nil {
		t.Fatalf("InsertSyncLog: %v", err)
	}

	sl, err := st.GetLastSync(ctx, "AU", 2026)
	if err != nil {
		t.Fatalf("GetLastSync: %v", err)
	}
	if sl == nil {
		t.Fatal("expected a SyncLog, got nil")
	}
	if sl.CountryCode != "AU" {
		t.Errorf("CountryCode = %q, want AU", sl.CountryCode)
	}
	if sl.Year != 2026 {
		t.Errorf("Year = %d, want 2026", sl.Year)
	}
	if sl.Source != "nager" {
		t.Errorf("Source = %q, want nager", sl.Source)
	}
	if sl.Status != "success" {
		t.Errorf("Status = %q, want success", sl.Status)
	}
	if sl.ErrorMessage != "" {
		t.Errorf("ErrorMessage = %q, want empty", sl.ErrorMessage)
	}
}
