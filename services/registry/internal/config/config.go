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

	JWTPrivateKeyPath    string
	RolesSeedPath        string
	InviteURLBase        string
	PasswordResetURLBase string

	// Only used to bootstrap the first account when the users table is
	// empty (see internal/user.Bootstrap) — not required afterwards.
	InitialAdminEmail    string
	InitialAdminUsername string
	InitialAdminPassword string

	// SMTP per l'invio automatico dell'email di invito (vedi
	// docs/adr/0023-invio-email-invito-smtp.md). Tutti opzionali: se
	// SMTPHost è vuoto, l'invio resta disabilitato e il comportamento è
	// identico a prima di questa funzionalità (link da copiare a mano) —
	// nessun fork è costretto a configurare Gmail per continuare a
	// funzionare.
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string

	// Percorso del file di branding (mittente) per-comitato, non un
	// segreto — vedi internal/user.LoadEmailBranding.
	EmailConfigPath string
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
		PasswordResetURLBase: getEnv("PASSWORD_RESET_URL_BASE", "http://localhost:5173/#/reset-password"),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminUsername: os.Getenv("INITIAL_ADMIN_USERNAME"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),

		EmailConfigPath: os.Getenv("EMAIL_CONFIG_PATH"),
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
		os.Setenv(key, unquote(strings.TrimSpace(value)))
	}
}

// unquote strips a single matching pair of surrounding quotes ('"' or "'")
// from an .env value, if present. This parser only treats a line as a
// comment when "#" is its very first character (see loadDotEnv above), so
// a value containing "#" — e.g. a URL fragment like ".../#/user-activation"
// — never needed quoting in the first place; but quoting values is such a
// standard .env convention that someone will reach for it anyway, and
// without this the literal quote characters would end up baked into the
// value (here: a broken invite link with stray '"' in it).
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
