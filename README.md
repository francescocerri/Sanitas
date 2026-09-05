# CRI Pavullo — Piattaforma gestionale

Insieme di microservizi per la gestione operativa di un Comitato CRI (turni, anagrafica volontari, mezzi e magazzino, servizi ed emergenze). Sviluppato per il Comitato di Pavullo, ma progettato per essere **forkato e riusato da altri comitati**.

Dettagli su architettura, convenzioni e contratto di forkabilità: vedi [CLAUDE.md](CLAUDE.md).

## Fork per altri comitati

Il progetto segue il modello *fork + configurazione*: ogni comitato fa il fork del repository, personalizza branding e dati in `config/<nome-comitato>/` e deploya la propria istanza indipendente. Nessun dato specifico del Comitato di Pavullo è hardcoded nel codice sorgente dei servizi.
