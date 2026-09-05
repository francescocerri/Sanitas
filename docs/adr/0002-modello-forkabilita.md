# 0002. Modello di forkabilità: fork + configurazione

Status: Accettata

## Contesto

L'obiettivo esplicito del progetto è essere riusato da più associazioni indipendenti (comitati CRI, Croce Verde, Croce Blu, ecc.), ciascuna con propri dati, branding e deployment. Due modelli possibili:

1. **Fork + configurazione**: ogni associazione forka il repo, personalizza una cartella di configurazione, deploya una propria istanza indipendente.
2. **Multi-tenant single-deployment**: un'unica installazione condivisa serve più associazioni, con isolamento dati a livello applicativo/DB.

## Decisione

Modello **fork + configurazione**. Ogni associazione fa il fork del repository, personalizza `config/<slug>/` (branding, dati anagrafici, endpoint) e deploya una propria istanza indipendente su propria infrastruttura.

## Conseguenze

- **Non** si progetta per l'isolamento dati multi-tenant in un deployment condiviso: niente tenant_id nelle tabelle, niente routing per-tenant, niente autenticazione cross-tenant. Chi propone questo tipo di design in futuro sta violando questa decisione.
- Nessun dato o identificativo specifico di un'associazione può finire hardcoded nel codice sorgente (vedi il contratto di forkabilità in `CLAUDE.md`): tutto ciò che è per-associazione vive in `config/<slug>/` o env var.
- Ogni associazione è responsabile della propria infrastruttura, backup, sicurezza operativa — il progetto fornisce il software, non un servizio gestito centralmente.
- Rende più semplice partire (nessuna infrastruttura multi-tenant da costruire) ma sposta il costo operativo (deploy, manutenzione, aggiornamenti) su ciascuna associazione che forka.
