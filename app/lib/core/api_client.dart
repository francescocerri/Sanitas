import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'auth/auth_controller.dart';
import 'raw_api_client.dart';

/// Client `dio` per le chiamate che richiedono un utente già autenticato
/// (`GET /v1/me`, `POST /v1/password/change`). A differenza di
/// `rawDioProvider`, questo:
///
/// 1. aggiunge da solo l'header `Authorization: Bearer <access token>` ad
///    ogni richiesta, leggendo il token corrente da `AuthController`;
/// 2. se una richiesta torna 401 (access token scaduto durante l'uso
///    dell'app — capita, dura solo 24h), prova UNA VOLTA a rinnovarlo
///    (`AuthController.refreshSession`) e ripete automaticamente la
///    richiesta originale con il nuovo token. Se anche il refresh fallisce,
///    l'errore risale al chiamante e `AuthController` avrà già spostato la
///    sessione su "non autenticato" (il router porterà l'utente al login).
///
/// Un "interceptor" in `dio` è semplicemente un punto di aggancio che gira
/// PRIMA di ogni richiesta (`onRequest`) o DOPO ogni errore (`onError`) —
/// utilissimo per logica trasversale come questa, che altrimenti andrebbe
/// ripetuta in ogni singola chiamata API.
final apiDioProvider = Provider<Dio>((ref) {
  final dio = Dio(BaseOptions(baseUrl: registryApiBaseUrl));

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        final accessToken = ref.read(authControllerProvider).accessToken;
        if (accessToken != null) {
          options.headers['Authorization'] = 'Bearer $accessToken';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final isUnauthorized = error.response?.statusCode == 401;
        // `error.requestOptions.extra` è un posto libero per passare dati
        // fra le fasi dell'interceptor: lo usiamo come "guardia" per non
        // ritentare all'infinito se anche la richiesta RIPETUTA torna 401.
        final alreadyRetried = error.requestOptions.extra['retriedAfterRefresh'] == true;

        if (!isUnauthorized || alreadyRetried) {
          handler.next(error);
          return;
        }

        try {
          await ref.read(authControllerProvider.notifier).refreshSession();
        } catch (_) {
          // Refresh fallito: propaga l'errore originale, la sessione è
          // già stata invalidata da `refreshSession()` stesso.
          handler.next(error);
          return;
        }

        final newAccessToken = ref.read(authControllerProvider).accessToken;
        final retryOptions = error.requestOptions
          ..headers['Authorization'] = 'Bearer $newAccessToken'
          ..extra['retriedAfterRefresh'] = true;

        try {
          final retryResponse = await dio.fetch(retryOptions);
          handler.resolve(retryResponse);
        } on DioException catch (retryError) {
          handler.next(retryError);
        }
      },
    ),
  );

  return dio;
});
