# 0022. Frontend: Flutter al posto di React/Vite

Status: Accettata

## Contesto

Lo scaffold frontend esistente (`web/`, React + TypeScript + Vite, [ADR-0003](0003-stack-tecnico.md)) era minimo: un'unica pagina non autenticata che elenca i turni, mai collegata a `registry`. Il progetto ha ora bisogno di un vero frontend per login e gestione anagrafica utenti, con requisiti espliciti: dev'essere il punto di forza del progetto (UI curata e moderna), disponibile sia via web sia come app installabile su App Store/Google Play, multilingua (italiano di default) e con tema colori personalizzabile per ogni comitato che forka il progetto — coerente con il contratto di forkabilità già in `CLAUDE.md`.

Era stata inizialmente valutata la strada React (Vite, stack già scelto) + Capacitor per il packaging nativo, per riusare l'esperienza già maturata. È stata poi scelta esplicitamente Flutter: offre nativamente un solo codebase per iOS/Android/Web senza bisogno di un guscio WebView, un sistema di theming (Material 3 `ColorScheme`/`ThemeData`) più diretto per la personalizzazione per-comitato richiesta, e componenti/animazioni pensati per un risultato visivo curato out-of-the-box.

`deploy/docker-compose.yml` non contiene oggi alcun servizio frontend: il vecchio scaffold React non era servito in produzione da nessuna parte, quindi sostituirlo non rompe nulla di esistente.

## Decisione

**Il frontend diventa un'unica app Flutter** in una nuova directory di primo livello `app/` (sostituisce `web/`, rimosso). Target: `web`, `android`, `ios` (desktop non richiesto, non escluso in futuro).

Package scelti (tutti attivamente mantenuti e ampiamente adottati):
- `flutter_riverpod` — gestione stato/dependency injection (sessione auth, client HTTP, tema, locale).
- `go_router` — routing dichiarativo con redirect/guard per le route protette (pacchetto ufficiale del team Flutter).
- `dio` — client HTTP con interceptor nativi, necessari per il refresh automatico del token su risposta 401.
- `flutter_secure_storage` — persistenza del solo refresh token (Keychain/Keystore nativi; su web storage cifrato via WebCrypto).
- `easy_localization` — i18n basato su file JSON per lingua, italiano default.
- Nessuna libreria per il JWT: il decode dei claim (non serve verificarne la firma lato client) è scritto a mano con `dart:convert`, coerente con la filosofia di minimizzare le dipendenze già seguita nel backend.

**Sessione**: l'access token vive solo in memoria (mai persistito), il refresh token in `flutter_secure_storage`. Al bootstrap dell'app si tenta un refresh silenzioso; se fallisce, l'utente va al login. Un interceptor `dio` ripete automaticamente una richiesta fallita per 401 dopo un refresh riuscito.

**Tema per-comitato**: nuovo file di convenzione `config/<slug>/app/theme.json` (stesso pattern annidato di `config/pavullo/registry/roles.json`), con colori e nome del comitato. Flutter non supporta in modo affidabile asset dichiarati fuori dalla directory del progetto su tutte le piattaforme di build, quindi il file viene sincronizzato a build-time in `app/assets/committee/theme.json` da uno script (`app/scripts/sync_committee_config.sh`) che legge `COMMITTEE_CONFIG_DIR` (stessa variabile già usata dal backend, per coerenza). Light/dark mode resta un asse separato: preferenza utente persistita, non configurazione da fork.

**Perimetro della prima iterazione**: solo self-service (login, profilo da `GET /v1/me`, cambio password, attivazione account da link di invito). Il pannello admin per creare utenti/assegnare ruoli è rimandato (voce di backlog separata) e richiederà anche un nuovo endpoint backend (`GET /v1/roles`, oggi assente).

## Conseguenze

- Riscrittura completa del frontend: lo scaffold React minimo in `web/` viene rimosso, non migrato (non c'era nulla di significativo da portare avanti).
- CI (`.github/workflows/ci.yml`): il job `web` (npm build/lint/audit) è sostituito da un job equivalente basato su toolchain Flutter (`flutter analyze`, `flutter test`, `flutter build web`). Le build native iOS/Android non entrano in CI per ogni PR — richiedono firma e sono lente, restano un'attività futura separata quando si arriverà davvero alla pubblicazione sugli store.
- Chi in futuro forka il progetto deve conoscere anche Dart/Flutter oltre a Go — barriera d'ingresso più alta rispetto a uno stack interamente web, tradeoff accettato in cambio di un'unica base di codice per web e store nativi.
- `CLAUDE.md` aggiornato: la riga "Frontend" in "Stack tecnico" riflette ora Flutter/Dart al posto di React/TypeScript/Vite; questa parte di [ADR-0003](0003-stack-tecnico.md) è superata da questo ADR (il resto di ADR-0003, la scelta di Go per il backend, resta valido e non è toccato).
- Nessuna migrazione dati o rottura in produzione: il vecchio frontend non era distribuito da nessuna parte.
- La strategia di hosting della build web e di pubblicazione reale sugli store (account sviluppatore, firma, CI di release) resta esplicitamente aperta — voce di backlog separata, non affrontata da questa decisione.
