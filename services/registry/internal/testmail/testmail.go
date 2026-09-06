// Package testmail avvia un server SMTP fittizio (mailpit) per i test di
// integrazione — stesso principio di internal/testdb per Postgres: una
// dipendenza reale e disponibile, non un mock/interfaccia (ADR-0010/0011),
// usa e getta per la durata dei test. Il codice che invia l'email
// (internal/user.Mailer) è esattamente lo stesso di produzione: cambia
// solo l'host SMTP a cui si connette.
package testmail

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Server espone l'host/porta SMTP (per configurare un internal/user.Mailer
// di test) e permette di interrogare l'API HTTP di mailpit per verificare
// che un'email sia stata effettivamente recapitata.
type Server struct {
	SMTPHost string
	SMTPPort int

	apiBaseURL string
}

// StartMailpit avvia un container mailpit, disponibile per tutta la durata
// dei test del package chiamante — va invocato una volta da TestMain,
// esattamente come testdb.StartPostgres.
func StartMailpit(ctx context.Context) (*Server, func(), error) {
	container, err := testcontainers.Run(ctx, "axllent/mailpit:v1.20",
		testcontainers.WithExposedPorts("1025/tcp", "8025/tcp"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("8025/tcp")),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("testmail: start mailpit container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("testmail: host: %w", err)
	}
	smtpPort, err := container.MappedPort(ctx, "1025/tcp")
	if err != nil {
		return nil, nil, fmt.Errorf("testmail: smtp port: %w", err)
	}
	httpPort, err := container.MappedPort(ctx, "8025/tcp")
	if err != nil {
		return nil, nil, fmt.Errorf("testmail: http port: %w", err)
	}

	server := &Server{
		SMTPHost:   host,
		SMTPPort:   int(smtpPort.Num()),
		apiBaseURL: fmt.Sprintf("http://%s:%s", host, httpPort.Port()),
	}

	cleanup := func() {
		_ = container.Terminate(context.Background())
	}
	return server, cleanup, nil
}

// HasMessageTo interroga l'API REST di mailpit e riporta se almeno un
// messaggio ricevuto finora ha `to` fra i destinatari. Cerca una semplice
// sottostringa nella risposta JSON grezza invece di derserializzarla in
// una struct tipizzata: lo schema esatto dell'API di mailpit non è
// qualcosa che controlliamo o vogliamo replicare campo per campo in questo
// repo, e per un test che deve solo confermare "l'email è arrivata al
// destinatario giusto" un controllo per sottostringa è sufficiente e più
// robusto a piccole differenze fra versioni dell'immagine.
func (s *Server) HasMessageTo(ctx context.Context, to string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiBaseURL+"/api/v1/messages", nil)
	if err != nil {
		return false, fmt.Errorf("testmail: build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("testmail: query mailpit: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("testmail: read response: %w", err)
	}
	return strings.Contains(string(body), to), nil
}
