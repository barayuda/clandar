package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                string
	LogLevel            string

	// Local SQLite fallback (used when TursoDatabaseURL is empty).
	DBPath              string

	// Turso cloud SQLite — takes priority over DBPath when both are set.
	// Set via TURSO_DATABASE_URL and TURSO_AUTH_TOKEN env vars.
	TursoDatabaseURL    string
	TursoAuthToken      string

	CalendarificAPIKey  string
}

// Load reads configuration from environment variables and applies defaults
// for any values that are not set.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		DBPath:           getEnv("DB_PATH", "./data/clandar.db"),
		TursoDatabaseURL: getEnv("TURSO_DATABASE_URL", ""),
		TursoAuthToken:   getEnv("TURSO_AUTH_TOKEN", ""),
		CalendarificAPIKey: getEnv("CALENDARIFIC_API_KEY", ""),
	}
}

// IsRemoteDB returns true when Turso cloud credentials are configured.
func (c *Config) IsRemoteDB() bool {
	return c.TursoDatabaseURL != ""
}

// getEnv returns the value of the named environment variable, or the provided
// fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
