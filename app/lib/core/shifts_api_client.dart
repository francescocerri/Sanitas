import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'authenticated_dio.dart';

/// URL base del servizio `shifts` — stesso meccanismo di
/// `registryApiBaseUrl` in `raw_api_client.dart` (`--dart-define-from-file`,
/// vedi `env.example.json`). Il default (`http://localhost:8080`) coincide
/// con `SHIFTS_HOST_PORT` nel `docker-compose.yml` di sviluppo.
const shiftsApiBaseUrl = String.fromEnvironment(
  'SHIFTS_API_URL',
  defaultValue: 'http://localhost:8080',
);

/// Equivalente di `apiDioProvider` (`api_client.dart`) ma per `shifts`:
/// stesso comportamento (header `Authorization` automatico, retry su 401
/// dopo un refresh), riusato da `createAuthenticatedDio` invece di
/// duplicarlo — l'unica differenza è il `baseUrl`.
final shiftsDioProvider = Provider<Dio>((ref) {
  return createAuthenticatedDio(ref, shiftsApiBaseUrl);
});
