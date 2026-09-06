import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// URL base del servizio `registry`. Flutter non ha un meccanismo di file
/// `.env` incorporato come Vite: il modo idiomatico per passare
/// configurazione a build/avvio time è `--dart-define`, letto qui con
/// `String.fromEnvironment`. Per comodità usiamo
/// `--dart-define-from-file=env.json` (vedi `env.example.json` e
/// `app/README.md`) invece di elencare ogni singola variabile da riga di
/// comando.
///
/// Il default (`http://localhost:8090`) coincide con `REGISTRY_HOST_PORT`
/// nel `docker-compose.yml` di sviluppo — funziona out-of-the-box con
/// `docker compose up registry` senza dover configurare nulla.
const registryApiBaseUrl = String.fromEnvironment(
  'REGISTRY_API_URL',
  defaultValue: 'http://localhost:8090',
);

/// Un client `dio` "nudo": nessun interceptor, nessuna gestione di
/// Authorization/refresh. Lo usa `AuthController` per le chiamate che non
/// richiedono (o non possono richiedere, essendo il login stesso) un access
/// token già valido: login, refresh, logout, attivazione account.
///
/// Le chiamate che INVECE richiedono un Bearer token già valido (`GET
/// /v1/me`, `POST /v1/password/change`) passano da `apiDioProvider` in
/// `api_client.dart`, che aggiunge l'header e gestisce il refresh
/// automatico su 401 — quel provider dipende da `AuthController`, mentre
/// questo qui no: separarli evita una dipendenza circolare fra i due.
final rawDioProvider = Provider<Dio>((ref) {
  return Dio(BaseOptions(baseUrl: registryApiBaseUrl));
});
