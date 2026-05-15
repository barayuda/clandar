package fetcher_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/barayuda/clandar/internal/fetcher"
)

// newTestNagerFetcher builds a NagerFetcher whose HTTP client and BaseURL both
// point at the provided test server.
func newTestNagerFetcher(srv *httptest.Server) *fetcher.NagerFetcher {
	return &fetcher.NagerFetcher{
		Client:  srv.Client(),
		BaseURL: srv.URL,
	}
}

func TestNagerFetch_Success(t *testing.T) {
	payload := []map[string]any{
		{
			"date":        "2026-01-01",
			"localName":   "New Year's Day",
			"name":        "New Year's Day",
			"countryCode": "AU",
			"fixed":       true,
			"global":      true,
			"counties":    nil,
			"launchYear":  nil,
			"types":       []string{"Public"},
		},
		{
			"date":        "2026-04-25",
			"localName":   "Anzac Day",
			"name":        "Anzac Day",
			"countryCode": "AU",
			"fixed":       true,
			"global":      true,
			"counties":    nil,
			"launchYear":  nil,
			"types":       []string{"Public"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	f := newTestNagerFetcher(srv)
	holidays, err := f.Fetch(context.Background(), "AU", 2026)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(holidays) != 2 {
		t.Fatalf("expected 2 holidays, got %d", len(holidays))
	}

	h0 := holidays[0]
	if h0.Date != "2026-01-01" {
		t.Errorf("holiday[0].Date = %q, want 2026-01-01", h0.Date)
	}
	if h0.Name != "New Year's Day" {
		t.Errorf("holiday[0].Name = %q, want New Year's Day", h0.Name)
	}
	if h0.CountryCode != "AU" {
		t.Errorf("holiday[0].CountryCode = %q, want AU", h0.CountryCode)
	}
	if h0.Type != "public" {
		t.Errorf("holiday[0].Type = %q, want public", h0.Type)
	}
	if h0.Source != "nager" {
		t.Errorf("holiday[0].Source = %q, want nager", h0.Source)
	}

	h1 := holidays[1]
	if h1.Date != "2026-04-25" {
		t.Errorf("holiday[1].Date = %q, want 2026-04-25", h1.Date)
	}
	if h1.Name != "Anzac Day" {
		t.Errorf("holiday[1].Name = %q, want Anzac Day", h1.Name)
	}
}

func TestNagerFetch_NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	f := newTestNagerFetcher(srv)
	holidays, err := f.Fetch(context.Background(), "XX", 2026)
	if err != nil {
		t.Fatalf("expected nil error for 204, got: %v", err)
	}
	if holidays != nil {
		t.Fatalf("expected nil slice for 204, got %v", holidays)
	}
}

func TestNagerFetch_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := newTestNagerFetcher(srv)
	holidays, err := f.Fetch(context.Background(), "ZZ", 2026)
	if err != nil {
		t.Fatalf("expected nil error for 404, got: %v", err)
	}
	if holidays != nil {
		t.Fatalf("expected nil slice for 404, got %v", holidays)
	}
}

func TestNagerFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := newTestNagerFetcher(srv)
	_, err := f.Fetch(context.Background(), "AU", 2026)
	if err == nil {
		t.Fatal("expected an error for 500 response, got nil")
	}
}

func TestNagerTypeToInternal(t *testing.T) {
	cases := []struct {
		name     string
		input    []string
		expected string
	}{
		{"Public maps to public", []string{"Public"}, "public"},
		{"Bank maps to public", []string{"Bank"}, "public"},
		{"Optional maps to public", []string{"Optional"}, "public"},
		{"Authorities maps to public", []string{"Authorities"}, "public"},
		{"School maps to school", []string{"School"}, "school"},
		{"Observance maps to observance", []string{"Observance"}, "observance"},
		{"Unknown defaults to public", []string{"Whatever"}, "public"},
		{"Empty slice defaults to public", []string{}, "public"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := fetcher.NagerTypeToInternal(tc.input)
			if got != tc.expected {
				t.Errorf("NagerTypeToInternal(%v) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
