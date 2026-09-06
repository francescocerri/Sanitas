# 0006. Policy di sicurezza e dipendenze

Status: Accettata

## Contesto

Requisito esplicito: ogni servizio deve usare librerie mantenute attivamente e prive di vulnerabilità di lungo termine, allineate agli standard di sicurezza attuali.

## Decisione

- **Preferire sempre la standard library** quando è sufficiente. Il servizio `shifts` usa `net/http` per il routing, nessun router esterno.
- Per ogni dipendenza esterna necessaria, valutare esplicitamente prima di aggiungerla: manutenzione attiva (commit recenti, non archiviata), adozione/popolarità, storico CVE.
- Dipendenze esterne scelte finora e perché:
  - `github.com/jackc/pgx/v5` — driver Postgres per Go, standard de facto, attivamente mantenuto (preferito a `lib/pq`, in sola manutenzione).
- **CI obbligatoria prima del merge**: `govulncheck` (tool ufficiale del team Go) per i servizi Go, `npm audit --audit-level=high` per il frontend — nessuna vulnerabilità nota deve passare.
- **Aggiornamento automatico**: Renovate (`renovate.json`, preset `config:recommended`) apre PR per Go modules, npm, immagini Docker base, versioni delle GitHub Actions. Le PR di sicurezza vanno prioritizzate.
- **Toolchain allineata a versioni mantenute**: Node aggiornato a 24 LTS (da 20.11, troppo vecchio per Vite 8) invece di restare su una major di Vite più vecchia solo per compatibilità; stesso principio per Go, dove il `go.mod` fissa anche una direttiva `toolchain` esplicita (non solo `go`) — coerente con "dipendenze mantenute" applicato anche al runtime/compilatore, non solo alle librerie.

## Conseguenze

- Superficie di dipendenze minima: meno codice di terze parti da fidarsi, meno aggiornamenti da inseguire.
- Ogni nuova dipendenza è una decisione esplicita da giustificare (qui o in un nuovo ADR), non un'aggiunta silenziosa.
- Un CVE reale è già stato intercettato e risolto durante lo sviluppo (`golang.org/x/text` v0.29.0 → v0.39.0, dipendenza transitiva di `pgx`) grazie a `govulncheck` — prova che il processo funziona.
- Un secondo incidente reale: pinnare `go.mod` a `go 1.25.0` esatto (scritto da `go mod init`, mai più toccato) ha lasciato in CI un toolchain con diverse vulnerabilità note della standard library (`crypto/x509`, `crypto/tls`, ecc.), invisibili in locale perché lì `govulncheck` scaricava automaticamente un toolchain più recente. Fissare esplicitamente `go`/`toolchain` a una versione patchata (`go1.26.8`) e lasciare che Renovate lo tenga aggiornato chiude il buco.
