package crm

import "os"

const (
	defaultPort         = "4000"
	defaultDatabasePath = "data/crm.sqlite"
)

// Port is the port the API listens on (PORT, default 4000).
func Port() string {
	return envOrDefault("PORT", defaultPort)
}

// DatabasePath is the SQLite file (DATABASE_PATH, default data/crm.sqlite relative to the working directory).
func DatabasePath() string {
	return envOrDefault("DATABASE_PATH", defaultDatabasePath)
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
