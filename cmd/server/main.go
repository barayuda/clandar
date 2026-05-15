package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/barayuda/clandar/internal/api"
	"github.com/barayuda/clandar/internal/config"
	"github.com/barayuda/clandar/internal/fetcher"
	"github.com/barayuda/clandar/internal/scheduler"
	"github.com/barayuda/clandar/internal/seeder"
	"github.com/barayuda/clandar/internal/store"
)

// schemaPath is the path to the SQL schema file, read at startup so the
// binary remains runnable from the project root without embedding.
//
// TODO: consider moving schema.sql into an internal/schema package so it can
// be embedded with //go:embed, making the binary fully self-contained. For now
// the Dockerfile copies db/ alongside the binary, so the runtime file read works.
const schemaPath = "db/schema.sql"

func main() {
	cfg := config.Load()

	// --- Logger setup ---
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	log.Info().
		Str("port", cfg.Port).
		Str("log_level", level.String()).
		Str("db_path", cfg.DBPath).
		Msg("starting clandar")

	// --- Schema ---
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", schemaPath).Msg("read schema file")
	}

	// --- Database ---
	st, err := store.Open(cfg.DBPath, string(schemaSQL))
	if err != nil {
		log.Fatal().Err(err).Msg("open database")
	}
	defer func() {
		if err := st.Close(); err != nil {
			log.Error().Err(err).Msg("close database")
		}
	}()

	log.Info().Str("db_path", cfg.DBPath).Msg("database ready")

	// --- Fetcher ---
	f := fetcher.New(cfg.CalendarificAPIKey)

	// --- Seeder ---
	sd := seeder.New(st, f, log.Logger)

	// --- Scheduler ---
	sc := scheduler.New(sd, log.Logger)

	// --- Router ---
	router := api.NewRouter("web", st)

	// --- HTTP server ---
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine so the main goroutine can wait for the
	// shutdown signal.
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("http server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Start the scheduler (runs an immediate sync + schedules annual re-syncs).
	// Use a root context that lives for the duration of the process.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	sc.Start(rootCtx)

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-serverErr:
		log.Error().Err(err).Msg("server error")
	}

	// Cancel the root context to stop scheduler goroutines.
	rootCancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
		os.Exit(1)
	}

	log.Info().Msg("server stopped")
}
