// Package store provides database access for the Clandar application.
package store

import "time"

// Country represents a row in the countries table.
type Country struct {
	Code      string
	Name      string
	Region    string
	FlagEmoji string
	CreatedAt time.Time
}

// Holiday represents a row in the holidays table.
type Holiday struct {
	ID          int64
	CountryCode string
	Date        string // "YYYY-MM-DD"
	Name        string
	Description string
	Type        string // "public", "religious", "cultural", "school", "observance"
	SubRegion   string
	Year        int
	Source      string
	CreatedAt   time.Time
}

// SyncLog represents a row in the sync_log table.
type SyncLog struct {
	ID           int64
	CountryCode  string
	Year         int
	Source       string
	SyncedAt     time.Time
	Status       string
	ErrorMessage string
}
