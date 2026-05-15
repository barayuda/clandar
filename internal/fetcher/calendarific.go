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

const calendarificBaseURL = "https://calendarific.com/api/v2/holidays"

// calendarificResponse is the top-level JSON response from the Calendarific API.
type calendarificResponse struct {
	Meta struct {
		Code int `json:"code"`
	} `json:"meta"`
	Response struct {
		Holidays []calendarificHoliday `json:"holidays"`
	} `json:"response"`
}

// calendarificHoliday is a single holiday entry from the Calendarific API.
type calendarificHoliday struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Country     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"country"`
	Date struct {
		ISO string `json:"iso"`
	} `json:"date"`
	Type []string `json:"type"`
}

// CalendarificFetcher fetches holidays from the Calendarific API.
type CalendarificFetcher struct {
	apiKey string
	client *http.Client
}

// newCalendarificFetcher creates a CalendarificFetcher with a 15-second timeout.
func newCalendarificFetcher(apiKey string) *CalendarificFetcher {
	return &CalendarificFetcher{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Fetch retrieves holidays from the Calendarific API for the given country and
// year. If the API key is empty the method returns an empty slice and no error,
// allowing Nager.Date to serve as the sole data source.
func (f *CalendarificFetcher) Fetch(ctx context.Context, countryCode string, year int) ([]store.Holiday, error) {
	if f.apiKey == "" {
		return nil, nil
	}

	url := fmt.Sprintf("%s?api_key=%s&country=%s&year=%d",
		calendarificBaseURL, f.apiKey, strings.ToUpper(countryCode), year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetcher: calendarific: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetcher: calendarific: do request for %s/%d: %w", countryCode, year, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetcher: calendarific: unexpected status %d for %s/%d", resp.StatusCode, countryCode, year)
	}

	var payload calendarificResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("fetcher: calendarific: decode response for %s/%d: %w", countryCode, year, err)
	}

	if payload.Meta.Code != http.StatusOK {
		return nil, fmt.Errorf("fetcher: calendarific: API error code %d for %s/%d", payload.Meta.Code, countryCode, year)
	}

	raw := payload.Response.Holidays
	holidays := make([]store.Holiday, 0, len(raw))
	for _, h := range raw {
		// The ISO date may include time; take just the date portion.
		date := h.Date.ISO
		if len(date) > 10 {
			date = date[:10]
		}
		holidays = append(holidays, store.Holiday{
			CountryCode: strings.ToUpper(countryCode),
			Date:        date,
			Name:        h.Name,
			Description: h.Description,
			Type:        calendarificTypeToInternal(h.Type),
			Year:        year,
			Source:      "calendarific",
		})
	}
	return holidays, nil
}

// calendarificTypeToInternal maps the Calendarific type array to our internal
// type string. Precedence: religious > school > observance > cultural > public.
func calendarificTypeToInternal(types []string) string {
	religiousKeywords := []string{"Religious", "Christian", "Muslim", "Jewish", "Hindu", "Buddhist"}
	for _, t := range types {
		for _, kw := range religiousKeywords {
			if strings.EqualFold(t, kw) {
				return "religious"
			}
		}
	}
	for _, t := range types {
		if strings.EqualFold(t, "School") {
			return "school"
		}
	}
	for _, t := range types {
		if strings.EqualFold(t, "Observance") ||
			strings.EqualFold(t, "Season") ||
			strings.EqualFold(t, "United Nations observance") {
			return "observance"
		}
	}
	for _, t := range types {
		if strings.EqualFold(t, "Cultural") {
			return "cultural"
		}
	}
	for _, t := range types {
		if strings.EqualFold(t, "National holiday") ||
			strings.EqualFold(t, "Common local holiday") {
			return "public"
		}
	}
	return "public"
}
