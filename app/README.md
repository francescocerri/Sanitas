# app — Sanitas (Flutter)

Frontend Flutter/Dart di Sanitas: un solo codebase per web, iOS e Android (vedi [ADR-0022](../docs/adr/0022-frontend-flutter.md)). Perimetro attuale: login, profilo self-service, cambio password, attivazione account da link di invito e creazione utenti per chi dispone di `users:manage`.

## Setup una tantum

Servono Flutter (`flutter --version`, questo progetto è stato scaffoldato con la 3.47.2 stable) e, per iOS/Android, i rispettivi toolchain nativi (Xcode/Android SDK) — non indispensabili per lavorare solo sulla build web.

```bash
flutter pub get
```

**Tema del comitato**: prima di lanciare l'app va sincronizzato il file di branding da `config/<slug>/app/theme.json` (vedi `CLAUDE.md` per la convenzione di fork):

```bash
./scripts/sync_committee_config.sh   # legge COMMITTEE_CONFIG_DIR, default ../config/pavullo
```

**URL delle API**: copia `env.example.json` in `env.json` (ignorato da git) e aggiorna `REGISTRY_API_URL` se il backend non gira sulla porta di default (`http://localhost:8090`, coerente con `REGISTRY_HOST_PORT` in `deploy/docker-compose.yml`):

```bash
cp env.example.json env.json
```

## Eseguire in locale

Con `services/registry` già in esecuzione (vedi il suo README, o `docker compose up registry` dalla cartella `deploy/`):

```bash
flutter run -d chrome --web-port=5173 --web-hostname=localhost --dart-define-from-file=env.json
```

`--web-port`/`--web-hostname` fissano l'indirizzo su cui gira il web server di sviluppo (`http://localhost:5173`): senza, Flutter ne sceglie uno casuale ad ogni avvio, costringendo ad aggiornare `CORS_ALLOWED_ORIGIN` lato `registry` (e qualunque bookmark) ogni volta. `5173` è lo stesso valore già usato come default in `services/registry/.env.example` — nessuna configurazione aggiuntiva necessaria in locale.

Per iOS/Android, sostituire `-d chrome --web-port=5173 --web-hostname=localhost` con il device/simulatore desiderato (`flutter devices` per l'elenco) — quei due flag sono specifici del target web.

## Test

```bash
dart format --output=none --set-exit-if-changed .
flutter analyze
flutter test
```

## Build

```bash
flutter build web --dart-define-from-file=env.json
```

Le build native (`flutter build apk`/`flutter build ios`) richiedono firma/provisioning e non sono ancora parte del flusso di questo progetto — vedi la voce di backlog "strategia di hosting/distribuzione".

## Struttura

```
lib/
  main.dart              # bootstrap: tema del comitato, i18n, Riverpod
  router.dart            # go_router: mappa delle route + redirect di autenticazione
  core/
    auth/                 # sessione (access token in memoria, refresh token sicuro)
    theme/                # tema per-comitato (colori, font) + preferenza chiaro/scuro
    widgets/               # componenti condivisi: monogramma, layout auth, banner, toggle tema
    api_client.dart        # client HTTP autenticato (Bearer + refresh automatico su 401)
    raw_api_client.dart     # client HTTP "nudo" per login/refresh/logout/attivazione
    jwt.dart               # decodifica claim JWT (nessuna libreria esterna)
  features/
    login/
    activate_account/
    profile/
assets/
  translations/          # it.json (default), en.json — easy_localization
  committee/              # tema sincronizzato da config/<slug>/app/ (non committato)
scripts/
  sync_committee_config.sh
```

## Multilingua e tema

### Icona dell'app

L'icona Sanitas (S intrecciata con dettaglio del battito) deriva da `../design/sanitas-icon/sanitas-icon-pulse-v3.png`. Le esportazioni PNG opache sono incluse in `web/` (favicon 32 px, icone PWA 192/512 px e Apple Touch 180 px), `android/app/src/main/res/mipmap-*/` (48–192 px) e `ios/Runner/Assets.xcassets/AppIcon.appiconset/` (dimensioni definite da `Contents.json`, inclusa l'icona App Store da 1024 px). Per sostituirla, rigenerare queste esportazioni dal nuovo master mantenendo nomi e dimensioni; il tema per-comitato non modifica automaticamente le icone installate.

- **Lingua**: `easy_localization`, italiano di default (sovrascrivibile per comitato via `default_locale` in `config/<slug>/app/theme.json`). Ogni stringa visibile all'utente passa da `assets/translations/*.json`, mai testo hardcoded in una lingua nel codice.
- **Colori**: personalizzabili per comitato (vedi sopra), applicati anche a tipografia (`google_fonts`, "Plus Jakarta Sans") e forma di campi/bottoni in `committee_theme.dart` — non solo la palette di Material di default. Chiaro/scuro è invece una preferenza personale dell'utente, persistita separatamente (vedi `theme_mode_controller.dart`).
- **Identità visiva**: nessun comitato è tenuto a fornire un logo — `BrandMark` (`core/widgets/brand_mark.dart`) mostra l'iniziale del nome del comitato in un cerchio colorato, sostituibile in futuro con un'immagine vera.
- **Animazioni**: transizioni fra schermate e ingresso dei form con `flutter_animate`, sobrie e coerenti invece che assenti o eccessive.

## Creazione utenti

Dal profilo, il pulsante «Crea utente» è visibile solo con `users:manage`; anche la route `/users/new` è protetta. Il form carica i ruoli con `GET /v1/roles` e invia email, username e slug dei ruoli a `POST /v1/users`. Dopo la creazione mostra il link restituito dal backend, selezionabile e copiabile. L’invio automatico dell’email è pianificato in `docs/backlog.md`.
