package config

import (
	"os"
	"strings"
)

// Config holds runtime settings derived once at startup from the environment.
type Config struct {
	ListenAddr          string
	StaticDir           string
	AdminPassword       string
	ContestantPassword  string
	ContestantsJSONPath string
}

// Load reads configuration from environment variables (defaults applied where noted).
func Load() Config {
	return Config{
		ListenAddr:          listenAddr(),
		StaticDir:           getenvDefault("STATIC_DIR", "dist"),
		AdminPassword:       strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		ContestantPassword:  strings.TrimSpace(os.Getenv("CONTESTANT_PASSWORD")),
		ContestantsJSONPath: contestantsJSONPath(),
	}
}

func listenAddr() string {
	if v := os.Getenv("HTTP_LISTEN"); v != "" {
		return v
	}
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return ":1323"
}

func getenvDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func contestantsJSONPath() string {
	p := strings.TrimSpace(os.Getenv("CONTESTANTS_JSON_PATH"))
	if p == "" {
		return "/data/contestants.json"
	}
	return p
}
