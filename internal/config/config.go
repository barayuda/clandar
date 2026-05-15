package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port                string
	LogLevel            string
	DBPath              string
	CalendarificAPIKey  string
}

// Load reads configuration from environment variables and applies defaults
// for any values that are not set.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DBPath:             getEnv("DB_PATH", "./data/clandar.db"),
		CalendarificAPIKey: getEnv("CALENDARIFIC_API_KEY", ""),
	}
}

// getEnv returns the value of the named environment variable, or the provided
// fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
