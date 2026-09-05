# 0003. Stack tecnico: Go + React

Status: Accettata

## Contesto

Serve uno stack per un ecosistema di microservizi backend più un frontend, mantenibile da un singolo sviluppatore esperto, con basso overhead operativo (coerente con il vincolo di costo bassissimo, vedi [ADR-0004](0004-target-di-deploy.md)).

## Decisione

- **Backend**: Go. Ogni microservizio è un modulo Go indipendente (`services/<nome>/go.mod` a sé stante).
- **Frontend**: React + TypeScript, build tool Vite.

## Conseguenze

- Go: binari statici piccoli, immagini Docker minimali, ottima concorrenza per servizi I/O-bound, toolchain di sicurezza integrata nell'ecosistema (`govulncheck`). Nessun framework web esterno di default: si preferisce la standard library (`net/http`) finché è sufficiente — vedi [ADR-0006](0006-policy-sicurezza-dipendenze.md).
- Ogni microservizio come modulo Go indipendente permette di aggiornarne le dipendenze e la versione di Go indipendentemente dagli altri, a costo di un minimo di duplicazione di configurazione (go.mod, Dockerfile) tra servizi.
- React + TypeScript + Vite: ecosistema maturo, build veloce, tipizzazione statica per ridurre errori di integrazione con le API. Vite richiede Node abbastanza recente (vedi nota in [ADR-0006](0006-policy-sicurezza-dipendenze.md) sull'allineamento a Node 24 LTS).
