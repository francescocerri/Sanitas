# 0004. Target di deploy: Raspberry Pi + Cloudflare Tunnel, portabile ad AWS

Status: Accettata

## Contesto

Vincolo esplicito: costo di esercizio molto molto basso, tendente a zero, mantenuto nel tempo. L'utente è skillato su AWS ma consapevole che il free tier EC2/RDS scade dopo 12 mesi (poi si paga). L'utente possiede già un Raspberry Pi a casa (costo marginale ~zero, dato che l'hardware è già suo).

## Decisione

Produzione su **Docker Compose sul Raspberry Pi**, esposto via **Cloudflare Tunnel** (`cloudflared`): nessuna porta aperta sul router, niente IP dinamico da gestire, TLS gratuito gestito da Cloudflare. Il deploy continuo gira su un **GitHub Actions self-hosted runner installato sul Raspberry Pi stesso**: la CI (build/test/scan) gira su runner GitHub-hosted gratuiti, il job di deploy gira sul runner self-hosted, quindi build Docker nativa ARM senza cross-compilazione né necessità di esporre SSH verso casa.

Tutto containerizzato e configurato via env var, in modo che una futura migrazione ad AWS (EC2/ECS) richieda solo di ripuntare il job di deploy a una nuova macchina, senza toccare il codice dei servizi.

## Conseguenze

- Costo di hosting ricorrente reale: ~zero (solo elettricità del Raspberry Pi, già di proprietà).
- Uptime e banda dipendono dalla connessione domestica dell'utente — accettabile per la scala attuale (un singolo comitato locale), da rivalutare se il progetto crescesse.
- Nessun backup offsite automatico del Raspberry Pi stesso (solo il DB ha una nota di backup separata, vedi [ADR-0005](0005-database-postgres-self-hosted.md)) — se l'hardware si guasta si perde lo stato finché non si ripristina da un backup manuale.
- Setup fisico (Docker, token del tunnel Cloudflare, registrazione del runner) è un'attività che l'utente deve eseguire lui stesso fuori da Claude Code — tracciata in `docs/backlog.md`.
