package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/barayuda/clandar/internal/api"
	"github.com/barayuda/clandar/internal/store"
)

// setupTestStore opens an in-memory SQLite store and seeds it with test data.
// The store is automatically closed when the test completes.
func setupTestStore(t *testing.T) *store.Store {
	t.Helper()

	schemaBytes, err := os.ReadFile("../../db/schema.sql")
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}

	st, err := store.Open(":memory:", string(schemaBytes))
	if err != nil {
		t.Fatalf("open in-memory store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := st.Close(); closeErr != nil {
			t.Errorf("close store: %v", closeErr)
		}
	})

	ctx := context.Background()

	// Seed countries.
	countries := []struct{ code, name, region, flag string }{
		{"AU", "Australia", "Asia-Pacific", "🇦🇺"},
		{"SG", "Singapore", "ASEAN", "🇸🇬"},
		{"DE", "Germany", "Europe", "🇩🇪"},
	}
	for _, c := range countries {
		if err := st.InsertCountry(ctx, c.code, c.name, c.region, c.flag); err != nil {
			t.Fatalf("seed country %s: %v", c.code, err)
		}
	}

	// Seed holidays.
	holidays := []store.Holiday{
		{CountryCode: "AU", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "AU", Date: "2026-04-25", Name: "Anzac Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "SG", Date: "2026-01-01", Name: "New Year's Day", Type: "public", Year: 2026, Source: "nager"},
		{CountryCode: "DE", Date: "2026-12-25", Name: "Christmas Day", Type: "public", Year: 2026, Source: "nager"},
	}
	for _, h := range holidays {
		if err := st.InsertHoliday(ctx, h); err != nil {
			t.Fatalf("seed holiday %s/%s: %v", h.CountryCode, h.Name, err)
		}
	}

	return st
}

// do sends a GET request against the router and returns the recorder.
func do(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// decodeJSON unmarshals the recorder body into v.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
}

func TestGetHolidays_ByCountry(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/holidays?country=AU&year=2026")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	count := int(body["count"].(float64))
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	holidays := body["holidays"].([]any)
	for _, raw := range holidays {
		h := raw.(map[string]any)
		if h["country_code"] != "AU" {
			t.Errorf("expected AU holiday, got country_code %q", h["country_code"])
		}
	}
}

func TestGetHolidays_ByRegion(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/holidays?region=ASEAN&year=2026")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	count := int(body["count"].(float64))
	if count != 1 {
		t.Errorf("count = %d, want 1 (SG holiday)", count)
	}

	holidays := body["holidays"].([]any)
	h := holidays[0].(map[string]any)
	if h["country_code"] != "SG" {
		t.Errorf("expected SG holiday, got country_code %q", h["country_code"])
	}
}

func TestGetHolidays_InvalidYear(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/holidays?year=1800")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	errMsg, ok := body["error"].(string)
	if !ok {
		t.Fatalf("expected error field in response, got %v", body)
	}
	if !strings.Contains(errMsg, "2000") {
		t.Errorf("error message %q should mention 2000", errMsg)
	}
}

func TestGetHolidays_InvalidType(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/holidays?type=garbage")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	errMsg, ok := body["error"].(string)
	if !ok {
		t.Fatalf("expected error field in response, got %v", body)
	}
	if !strings.Contains(errMsg, "observance") {
		t.Errorf("error message %q should mention 'observance'", errMsg)
	}
}

func TestGetHolidays_DefaultYear(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	// No year param — should default to current year and return 200 (possibly 0 results).
	rec := do(t, router, "/api/holidays?country=AU")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	if _, ok := body["holidays"]; !ok {
		t.Error("expected holidays field in response")
	}
}

func TestGetCountries_All(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/countries")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	count := int(body["count"].(float64))
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestGetCountries_ByRegion(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/countries?region=ASEAN")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	count := int(body["count"].(float64))
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	countries := body["countries"].([]any)
	c := countries[0].(map[string]any)
	if c["code"] != "SG" {
		t.Errorf("expected SG, got code %q", c["code"])
	}
}

func TestGetRegions(t *testing.T) {
	st := setupTestStore(t)
	router := api.NewRouter(".", st)

	rec := do(t, router, "/api/regions")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	decodeJSON(t, rec, &body)

	regions, ok := body["regions"].([]any)
	if !ok {
		t.Fatalf("expected regions array, got %T", body["regions"])
	}
	if len(regions) != 4 {
		t.Errorf("len(regions) = %d, want 4", len(regions))
	}

	// Verify all 4 known region names are present.
	wantNames := map[string]bool{
		"ASEAN":       true,
		"Asia-Pacific": true,
		"Europe":      true,
		"Americas":    true,
	}
	for _, raw := range regions {
		r := raw.(map[string]any)
		name, _ := r["name"].(string)
		delete(wantNames, name)
	}
	if len(wantNames) > 0 {
		t.Errorf("missing regions in response: %v", wantNames)
	}
}
