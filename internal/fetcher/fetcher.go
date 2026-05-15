package fetcher

import (
	"context"
	"fmt"

	"github.com/barayuda/clandar/internal/store"
)

// Fetcher combines Nager.Date (always) and Calendarific (when an API key is
// configured) into a single data source. Nager is called first; Calendarific
// supplements it when available. DB-level UNIQUE constraints handle deduplication
// on insert.
type Fetcher struct {
	Nager        *NagerFetcher
	Calendarific *CalendarificFetcher
}

// New creates a Fetcher. calendarificKey may be empty — in that case the
// Calendarific client is created but will return empty results gracefully.
func New(calendarificKey string) *Fetcher {
	return &Fetcher{
		Nager:        newNagerFetcher(),
		Calendarific: newCalendarificFetcher(calendarificKey),
	}
}

// FetchAll retrieves holidays for the given country and year from all
// configured sources and returns the combined slice. The caller is responsible
// for persisting the results; this method performs no DB operations.
func (f *Fetcher) FetchAll(ctx context.Context, countryCode string, year int) ([]store.Holiday, error) {
	nagerHolidays, err := f.Nager.Fetch(ctx, countryCode, year)
	if err != nil {
		return nil, fmt.Errorf("fetcher: nager fetch %s/%d: %w", countryCode, year, err)
	}

	calHolidays, err := f.Calendarific.Fetch(ctx, countryCode, year)
	if err != nil {
		// Calendarific is supplemental — log-worthy but not fatal. Return Nager
		// results and propagate the error so the caller can decide.
		return nagerHolidays, fmt.Errorf("fetcher: calendarific fetch %s/%d: %w", countryCode, year, err)
	}

	combined := make([]store.Holiday, 0, len(nagerHolidays)+len(calHolidays))
	combined = append(combined, nagerHolidays...)
	combined = append(combined, calHolidays...)
	return combined, nil
}
