// Package api wires together the HTTP router and all request handlers for
// the Clandar public holiday API.
package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/barayuda/clandar/internal/store"
)

// NewRouter builds and returns a fully configured chi router.
// staticDir is the path to the directory that should be served as static files
// at the root URL (e.g. "./web"). st is stored on handlers for use in Phase 3.
func NewRouter(staticDir string, st *store.Store) http.Handler {
	h := newHandler(st)

	r := chi.NewRouter()

	// --- Global middleware ---
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(zerologMiddleware)
	r.Use(middleware.Recoverer)

	// --- Health check (no auth, no logging noise) ---
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- API routes ---
	r.Route("/api", func(r chi.Router) {
		r.Get("/holidays", h.GetHolidays)
		r.Get("/countries", h.GetCountries)
		r.Get("/regions", h.GetRegions)
	})

	// --- Static files (serves web/ at /) ---
	fs := http.FileServer(http.Dir(staticDir))
	r.Handle("/*", fs)

	return r
}

// zerologMiddleware is a chi-compatible middleware that logs each request as a
// structured zerolog JSON entry.
func zerologMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			var event *zerolog.Event
			status := ww.Status()
			switch {
			case status >= 500:
				event = log.Error()
			case status >= 400:
				event = log.Warn()
			default:
				event = log.Info()
			}

			event.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("request_id", middleware.GetReqID(r.Context())).
				Int("status", status).
				Int("bytes", ww.BytesWritten()).
				Dur("duration_ms", time.Since(start)).
				Msg("request")
		}()

		next.ServeHTTP(ww, r)
	})
}
