// Package scheduler provides a background job that keeps holiday data
// up to date without requiring an external cron daemon.
package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/barayuda/clandar/internal/seeder"
)

// Scheduler runs periodic holiday sync jobs in the background.
type Scheduler struct {
	Seeder *seeder.Seeder
	Logger zerolog.Logger
}

// New creates a Scheduler.
func New(s *seeder.Seeder, log zerolog.Logger) *Scheduler {
	return &Scheduler{Seeder: s, Logger: log}
}

// Start launches background goroutines that:
//  1. Run SyncAll immediately (non-blocking with respect to the HTTP server).
//  2. Schedule a re-sync on each subsequent January 1st UTC to pull the new
//     year's data. The timer re-arms itself after each trigger so it fires
//     every year indefinitely.
//
// The goroutines respect ctx cancellation so they shut down cleanly.
func (sc *Scheduler) Start(ctx context.Context) {
	// 1. Immediate startup sync — runs in its own goroutine so the HTTP server
	//    starts accepting requests without waiting for the potentially slow
	//    holiday fetch.
	go func() {
		sc.Logger.Info().Msg("scheduler: running startup sync")
		if err := sc.Seeder.SyncAll(ctx); err != nil {
			sc.Logger.Error().Err(err).Msg("scheduler: startup sync failed")
		}
	}()

	// 2. Annual re-sync — fires on Jan 1 UTC each year.
	go sc.scheduleAnnualSync(ctx)
}

// scheduleAnnualSync waits until the next Jan 1 UTC midnight and then runs
// SyncAll. After each run it recalculates the next trigger so the goroutine
// stays alive for the lifetime of the process.
func (sc *Scheduler) scheduleAnnualSync(ctx context.Context) {
	for {
		d := durationUntilNextJan1UTC()
		sc.Logger.Info().
			Dur("in", d).
			Msg("scheduler: next annual sync scheduled")

		select {
		case <-ctx.Done():
			sc.Logger.Info().Msg("scheduler: annual sync goroutine stopped")
			return
		case <-time.After(d):
		}

		sc.Logger.Info().Msg("scheduler: running annual sync")
		if err := sc.Seeder.SyncAll(ctx); err != nil {
			sc.Logger.Error().Err(err).Msg("scheduler: annual sync failed")
		}
	}
}

// durationUntilNextJan1UTC returns the duration from now until the next
// January 1st 00:00:00 UTC. If today is Jan 1 the next occurrence is
// Jan 1 of the following year.
func durationUntilNextJan1UTC() time.Duration {
	now := time.Now().UTC()
	nextJan1 := time.Date(now.Year()+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	// If today happens to be Jan 1, still schedule for the next year.
	if now.Month() == time.January && now.Day() == 1 {
		nextJan1 = time.Date(now.Year()+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return nextJan1.Sub(now)
}
