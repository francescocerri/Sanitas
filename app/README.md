# app — Sanitas (Flutter)

Frontend Flutter/Dart di Sanitas: un solo codebase per web, iOS e Android (vedi [ADR-0022](../docs/adr/0022-frontend-flutter.md)). Perimetro attuale: login, profilo self-service, cambio password, attivazione account da link di invito.

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
flutter run -d chrome --dart-define-from-file=env.json
```

Per iOS/Android, sostituire `-d chrome` con il device/simulatore desiderato (`flutter devices` per l'elenco).

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

- **Lingua**: `easy_localization`, italiano di default (sovrascrivibile per comitato via `default_locale` in `config/<slug>/app/theme.json`). Ogni stringa visibile all'utente passa da `assets/translations/*.json`, mai testo hardcoded in una lingua nel codice.
- **Colori**: personalizzabili per comitato (vedi sopra), applicati anche a tipografia (`google_fonts`, "Plus Jakarta Sans") e forma di campi/bottoni in `committee_theme.dart` — non solo la palette di Material di default. Chiaro/scuro è invece una preferenza personale dell'utente, persistita separatamente (vedi `theme_mode_controller.dart`).
- **Identità visiva**: nessun comitato è tenuto a fornire un logo — `BrandMark` (`core/widgets/brand_mark.dart`) mostra l'iniziale del nome del comitato in un cerchio colorato, sostituibile in futuro con un'immagine vera.
- **Animazioni**: transizioni fra schermate e ingresso dei form con `flutter_animate`, sobrie e coerenti invece che assenti o eccessive.
