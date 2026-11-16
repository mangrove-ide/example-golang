package web

import "os"

// Port is the port the web app listens on (PORT, default 5173).
func Port() string {
	return envOrDefault("PORT", "5173")
}

// APIURL is the REST API the web app calls (API_URL, else http://localhost:$API_PORT with API_PORT defaulting to 4000).
func APIURL() string {
	return envOrDefault("API_URL", "http://localhost:"+envOrDefault("API_PORT", "4000"))
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
