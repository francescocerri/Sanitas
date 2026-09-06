package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port              string
	DatabaseURL       string
	CORSAllowedOrigin string
	AuthJWKSURL       string
}

func Load() (Config, error) {
	loadDotEnv()

	cfg := Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		CORSAllowedOrigin: getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),
		AuthJWKSURL:       os.Getenv("AUTH_JWKS_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL not set")
	}
	if cfg.AuthJWKSURL == "" {
		return Config{}, fmt.Errorf("AUTH_JWKS_URL not set")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv loads ./.env into the process environment, if present — purely
// a local development convenience (go run without having to `source .env`
// in the same shell, a real footgun if run from a different terminal).
// Real environment variables (set by Docker/systemd/etc.) always win: a key
// already present in the environment is never overwritten. No error if the
// file doesn't exist: that's the normal case in Docker/production.
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, unquote(strings.TrimSpace(value)))
	}
}

// unquote strips a single matching pair of surrounding quotes ('"' or "'")
// from an .env value, if present — a standard .env convention this simple
// line-based parser should honor rather than silently baking the literal
// quote characters into the value (see registry's identical helper, same
// cross-cutting config-loading pattern per ADR-0010).
func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
