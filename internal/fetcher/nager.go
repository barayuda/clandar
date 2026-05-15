// Package fetcher provides clients for public holiday data sources.
package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/barayuda/clandar/internal/store"
)

const nagerBaseURL = "https://date.nager.at/api/v3/PublicHolidays"

// nagerHoliday is the JSON shape returned by the Nager.Date API.
type nagerHoliday struct {
	Date        string   `json:"date"`
	LocalName   string   `json:"localName"`
	Name        string   `json:"name"`
	CountryCode string   `json:"countryCode"`
	Fixed       bool     `json:"fixed"`
	Global      bool     `json:"global"`
	Counties    []string `json:"counties"`
	LaunchYear  *int     `json:"launchYear"`
	Types       []string `json:"types"`
}

// NagerFetcher fetches public holidays from the free Nager.Date API.
// Client and BaseURL are exported so tests can inject a httptest.Server without
// hitting the real network.
type NagerFetcher struct {
	Client  *http.Client
	BaseURL string
}

// newNagerFetcher creates a NagerFetcher with a 10-second timeout.
func newNagerFetcher() *NagerFetcher {
	return &NagerFetcher{
		Client:  &http.Client{Timeout: 10 * time.Second},
		BaseURL: nagerBaseURL,
	}
}

// Fetch retrieves public holidays for the given country code and year from
// the Nager.Date API and returns them as store.Holiday values.
func (f *NagerFetcher) Fetch(ctx context.Context, countryCode string, year int) ([]store.Holiday, error) {
	url := fmt.Sprintf("%s/%d/%s", f.BaseURL, year, strings.ToUpper(countryCode))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetcher: nager: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetcher: nager: do request for %s/%d: %w", countryCode, year, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent {
		// 404 = country not supported by Nager; 204 = no data for this year.
		// Both are graceful — return empty slice so Calendarific can supplement.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetcher: nager: unexpected status %d for %s/%d", resp.StatusCode, countryCode, year)
	}

	var raw []nagerHoliday
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("fetcher: nager: decode response for %s/%d: %w", countryCode, year, err)
	}

	holidays := make([]store.Holiday, 0, len(raw))
	for _, h := range raw {
		holidays = append(holidays, store.Holiday{
			CountryCode: strings.ToUpper(countryCode),
			Date:        h.Date,
			Name:        h.Name,
			Description: "",
			Type:        NagerTypeToInternal(h.Types),
			SubRegion:   strings.Join(h.Counties, ","),
			Year:        year,
			Source:      "nager",
		})
	}
	return holidays, nil
}

// NagerTypeToInternal maps the Nager.Date types array to our internal type string.
// Exported so it can be exercised directly in tests.
func NagerTypeToInternal(types []string) string {
	for _, t := range types {
		switch t {
		case "Public", "Bank", "Optional", "Authorities":
			return "public"
		case "School":
			return "school"
		case "Observance":
			return "observance"
		}
	}
	return "public"
}
