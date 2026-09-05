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

	JWTPrivateKeyPath string
	RolesSeedPath     string
	InviteURLBase     string

	// Only used to bootstrap the first account when the users table is
	// empty (see internal/user.Bootstrap) — not required afterwards.
	InitialAdminEmail    string
	InitialAdminUsername string
	InitialAdminPassword string
}

func Load() (Config, error) {
	loadDotEnv()

	cfg := Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		CORSAllowedOrigin:    getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),
		JWTPrivateKeyPath:    os.Getenv("JWT_PRIVATE_KEY_PATH"),
		RolesSeedPath:        os.Getenv("ROLES_SEED_PATH"),
		InviteURLBase:        getEnv("INVITE_URL_BASE", "http://localhost:5173/attiva"),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminUsername: os.Getenv("INITIAL_ADMIN_USERNAME"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL not set")
	}
	if cfg.JWTPrivateKeyPath == "" {
		return Config{}, fmt.Errorf("JWT_PRIVATE_KEY_PATH not set")
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
		os.Setenv(key, strings.TrimSpace(value))
	}
}
