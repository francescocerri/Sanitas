import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'authenticated_dio.dart';
import 'raw_api_client.dart';

/// Client `dio` per le chiamate a `registry` che richiedono un utente già
/// autenticato (`GET /v1/me`, `POST /v1/password/change`, `GET /v1/users`,
/// ...). A differenza di `rawDioProvider`, aggiunge da solo l'header
/// `Authorization` e gestisce il refresh automatico su 401 — vedi
/// `createAuthenticatedDio` in `authenticated_dio.dart` per i dettagli,
/// condivisa con `shiftsDioProvider` (`shifts_api_client.dart`).
final apiDioProvider = Provider<Dio>((ref) {
  return createAuthenticatedDio(ref, registryApiBaseUrl);
});
