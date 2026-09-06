package user

import (
	"encoding/json"
	"fmt"
	"os"
)

// EmailBranding è il mittente mostrato nelle email di invito: dati del
// comitato, non un segreto, per questo vive in un file versionato nel fork
// (`config/<slug>/registry/email.json`) invece che in una variabile
// d'ambiente — stesso principio già seguito per i ruoli (ADR-0012).
type EmailBranding struct {
	FromName    string `json:"from_name"`
	FromAddress string `json:"from_address"`

	// PrimaryColor colora l'intestazione dell'email (formato "#RRGGBB",
	// stesso formato di "colors.primary" in config/<slug>/app/theme.json —
	// duplicato qui invece di far leggere quel file anche al backend,
	// perché sono due file con due consumer distinti (l'app Flutter e
	// questo servizio) e non vale la pena accoppiarli per un solo campo.
	// Facoltativo: vuoto = un grigio neutro invece del colore del brand.
	PrimaryColor string `json:"primary_color"`
}

// LoadEmailBranding legge il file di branding al path indicato. Path vuoto
// o file assente non sono un errore: risultano in un `EmailBranding` vuoto,
// che il chiamante userà per non costruire alcun `Mailer` — un comitato
// che non vuole ancora configurare l'invio email non deve fare nulla di
// diverso da oggi.
func LoadEmailBranding(path string) (EmailBranding, error) {
	if path == "" {
		return EmailBranding{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return EmailBranding{}, fmt.Errorf("load email branding: read %s: %w", path, err)
	}
	var branding EmailBranding
	if err := json.Unmarshal(raw, &branding); err != nil {
		return EmailBranding{}, fmt.Errorf("load email branding: parse %s: %w", path, err)
	}
	return branding, nil
}
