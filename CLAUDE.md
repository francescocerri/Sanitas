# CLAUDE.md

Questo file guida chiunque (umano o Claude) lavori su questo repository, incluso chi lo forka per un altro comitato. È la fonte di verità condivisa per architettura, vincoli e convenzioni: va aggiornato ogni volta che una decisione rilevante cambia.

## Overview

Piattaforma a microservizi per la gestione operativa di un Comitato CRI: gestione turni, anagrafica volontari/soci, mezzi e magazzino, servizi ed emergenze. Sviluppata per il Comitato di Pavullo, progettata esplicitamente per essere forkata e riusata da altri comitati CRI.

Scope MVP (in ordine di priorità):
1. Gestione turni
2. Anagrafica volontari/soci
3. Gestione mezzi e magazzino
4. Servizi ed emergenze

## Contratto di forkabilità (regola non negoziabile)

**Nessun dato o identificativo specifico del Comitato di Pavullo deve mai finire hardcoded nel codice sorgente.** Nome del comitato, loghi, colori, contatti, indirizzi, elenco mezzi, utenti, qualunque dato reale del deployment: tutto vive in configurazione esterna al codice, mai in constant, default, o fixture di test che assomiglino a dati reali.

Convenzione di configurazione: `config/<committee-slug>/` contiene gli override per-comitato (branding, dati anagrafici del comitato, endpoint). Il codice dei servizi legge sempre da lì o da variabili d'ambiente, mai da valori inline.

Modello di fork: **fork + configurazione**, non multi-tenant single-deployment. Ogni comitato fa il fork del repo, personalizza `config/<nuovo-comitato>/`, branding e dati, e deploya una propria istanza indipendente. Non progettare per l'isolamento dati multi-tenant in un unico deployment condiviso — non è l'obiettivo di questo progetto.

Prima di scrivere codice per una nuova feature, chiediti: "questo assumerebbe qualcosa di specifico di Pavullo?" Se sì, va parametrizzato.

## Stack tecnico

- **Backend**: Go. Ogni microservizio è un modulo Go indipendente.
- **Frontend**: React + TypeScript.

## Struttura repo prevista

Non ancora creata (sarà oggetto di una sessione di scaffolding dedicata). Struttura di riferimento da rispettare quando verrà creata:

```
services/<nome-servizio>/   # es. turni, anagrafica, mezzi-magazzino, servizi-emergenze
                             # ciascuno Go module a sé stante
web/                         # app React
config/<committee-slug>/    # override per-comitato (branding, dati, endpoint)
docs/
deploy/                     # docker-compose / manifest, da definire
```

## Convenzioni di lavoro con Claude Code

- Usare **Plan Mode** prima di iniziare ogni nuovo microservizio o feature non banale.
- Passare da `/code-review` prima di ogni merge.
- Aggiornare questo file ogni volta che cambia una decisione architetturale rilevante.
- Commit: Conventional Commits. Branching: feature branch + PR.
- Test: unit test per servizio Go; test di integrazione via docker-compose quando più servizi comunicheranno tra loro.

## Nota per chi fa il fork

1. Fork del repository.
2. Creare `config/<nuovo-comitato>/` con branding e dati del proprio comitato.
3. Personalizzare eventuali override necessari.
4. Deploy indipendente (dettagli tecnici da definire quando si disegnerà `deploy/`).

## Licenza

MIT (vedi [LICENSE](LICENSE)). Permissiva: chi forka non è obbligato a ridare indietro le modifiche.
