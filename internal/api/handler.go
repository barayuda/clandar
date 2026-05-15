package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/barayuda/clandar/internal/store"
)

// Handler holds shared dependencies (store, etc.) that API handlers need.
type Handler struct {
	store *store.Store
}

// newHandler creates a Handler with the given store.
func newHandler(st *store.Store) *Handler {
	return &Handler{store: st}
}

// validHolidayTypes is the set of accepted holiday type values.
var validHolidayTypes = map[string]bool{
	"public":      true,
	"religious":   true,
	"cultural":    true,
	"school":      true,
	"observance":  true,
}

// knownRegions lists the regions supported by Clandar, in display order.
var knownRegions = []string{"ASEAN", "Asia-Pacific", "Europe", "Americas"}

// writeJSON encodes v as JSON and writes it to w with the given HTTP status
// code. Any encoding error is logged and a 500 is returned instead.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error body with the given status and message.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// parseYear extracts and validates the "year" query param.
// Returns the current year when the param is absent, and an error string
// (suitable for writeError) when the value is present but invalid.
func parseYear(r *http.Request) (int, string) {
	raw := r.URL.Query().Get("year")
	if raw == "" {
		return time.Now().Year(), ""
	}
	y, err := strconv.Atoi(raw)
	if err != nil || y < 2000 || y > 2100 {
		return 0, "invalid year: must be between 2000 and 2100"
	}
	return y, ""
}

// holidayJSON is the JSON representation of a holiday returned by the API.
type holidayJSON struct {
	Date        string `json:"date"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	CountryCode string `json:"country_code"`
	Year        int    `json:"year"`
}

// toHolidayJSON converts a store.Holiday to its API representation.
func toHolidayJSON(h store.Holiday) holidayJSON {
	return holidayJSON{
		Date:        h.Date,
		Name:        h.Name,
		Description: h.Description,
		Type:        h.Type,
		CountryCode: h.CountryCode,
		Year:        h.Year,
	}
}

// GetHolidays handles GET /api/holidays.
//
// Query params:
//   - country  — ISO country code (optional)
//   - region   — region name (optional)
//   - year     — 4-digit year, defaults to current year
//   - type     — holiday type filter (optional)
func (h *Handler) GetHolidays(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	year, errMsg := parseYear(r)
	if errMsg != "" {
		writeError(w, http.StatusBadRequest, errMsg)
		return
	}

	holidayType := q.Get("type")
	if holidayType != "" && !validHolidayTypes[holidayType] {
		writeError(w, http.StatusBadRequest,
			"invalid type: must be one of public, religious, cultural, school, observance")
		return
	}

	country := q.Get("country")
	region := q.Get("region")

	var holidays []store.Holiday
	var storeErr error

	switch {
	case country != "":
		if holidayType != "" {
			holidays, storeErr = h.store.GetHolidaysByCountryYearAndType(ctx, country, year, holidayType)
		} else {
			holidays, storeErr = h.store.GetHolidaysByCountryAndYear(ctx, country, year)
		}

	case region != "":
		holidays, storeErr = h.store.GetHolidaysByRegionAndYear(ctx, region, year)
		// Apply in-Go type filter when the store method doesn't support it.
		if storeErr == nil && holidayType != "" {
			filtered := holidays[:0]
			for _, hol := range holidays {
				if hol.Type == holidayType {
					filtered = append(filtered, hol)
				}
			}
			holidays = filtered
		}

	default:
		holidays, storeErr = h.store.GetAllHolidaysByYear(ctx, year)
		// Apply in-Go type filter for the all-countries path.
		if storeErr == nil && holidayType != "" {
			filtered := holidays[:0]
			for _, hol := range holidays {
				if hol.Type == holidayType {
					filtered = append(filtered, hol)
				}
			}
			holidays = filtered
		}
	}

	if storeErr != nil {
		log.Error().Err(storeErr).Msg(fmt.Sprintf("GetHolidays: store error (country=%q region=%q year=%d type=%q)",
			country, region, year, holidayType))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Guarantee a non-null JSON array even when there are no results.
	out := make([]holidayJSON, 0, len(holidays))
	for _, hol := range holidays {
		out = append(out, toHolidayJSON(hol))
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, map[string]any{
		"year":     year,
		"count":    len(out),
		"holidays": out,
	})
}

// countryJSON is the JSON representation of a country returned by the API.
type countryJSON struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	FlagEmoji string `json:"flag_emoji"`
}

// GetCountries handles GET /api/countries.
//
// Query params:
//   - region — optional filter by region name
func (h *Handler) GetCountries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	region := r.URL.Query().Get("region")

	var countries []store.Country
	var storeErr error

	if region != "" {
		countries, storeErr = h.store.GetCountriesByRegion(ctx, region)
	} else {
		countries, storeErr = h.store.GetCountries(ctx)
	}

	if storeErr != nil {
		log.Error().Err(storeErr).Msg(fmt.Sprintf("GetCountries: store error (region=%q)", region))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	out := make([]countryJSON, 0, len(countries))
	for _, c := range countries {
		out = append(out, countryJSON{
			Code:      c.Code,
			Name:      c.Name,
			Region:    c.Region,
			FlagEmoji: c.FlagEmoji,
		})
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(out),
		"countries": out,
	})
}

// regionJSON is the JSON representation of a region returned by the API.
type regionJSON struct {
	Name         string `json:"name"`
	CountryCount int    `json:"country_count"`
}

// GetRegions handles GET /api/regions.
// Returns the three known regions with their country counts.
func (h *Handler) GetRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	counts, err := h.store.GetRegionCounts(ctx)
	if err != nil {
		log.Error().Err(err).Msg("GetRegions: store error fetching region counts")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	regions := make([]regionJSON, 0, len(knownRegions))
	for _, name := range knownRegions {
		regions = append(regions, regionJSON{
			Name:         name,
			CountryCount: counts[name], // zero if not in DB yet
		})
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	writeJSON(w, http.StatusOK, map[string]any{
		"regions": regions,
	})
}
