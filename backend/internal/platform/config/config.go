package config

import "os"

// Config holds the runtime configuration, read from the environment with
// sensible local-dev defaults.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func Load() Config {
	return Config{
		DatabaseURL: getenv("DATABASE_URL", "postgres://bidding:bidding@localhost:55432/bidding?sslmode=disable"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
