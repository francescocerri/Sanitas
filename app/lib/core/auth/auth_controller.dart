import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../api_exception.dart';
import '../jwt.dart';
import '../raw_api_client.dart';
import 'auth_state.dart';
import 'secure_token_store.dart';

/// Il "cervello" della sessione utente: espone `login`/`logout`/
/// `refreshSession`/`activateAccount` e pubblica lo stato corrente
/// ([AuthSession]) a tutta l'app tramite Riverpod.
///
/// In Riverpod, un `Notifier<T>` è la classe base per "uno stato che cambia
/// nel tempo e che la UI vuole osservare": si scrive `build()` per lo stato
/// iniziale, poi si aggiorna `state = nuovoValore` da qualunque metodo — ogni
/// widget che "ascolta" questo provider (con `ref.watch(authControllerProvider)`)
/// si ricostruisce automaticamente quando `state` cambia. Non serve
/// `setState`/`StatefulWidget`: è lo stesso concetto, ma condiviso
/// globalmente invece che dentro un solo widget.
class AuthController extends Notifier<AuthSession> {
  @override
  AuthSession build() {
    // `build()` deve restituire subito un valore (non può essere `async`),
    // quindi partiamo con "stato sconosciuto" e lanciamo il controllo vero
    // e proprio (c'è un refresh token salvato da una sessione precedente?)
    // in background: quando finisce, `_bootstrap` aggiorna `state` da solo,
    // e la UI (vedi `router.dart`) osserva quel cambiamento per decidere se
    // mandare l'utente al login o alla schermata protetta.
    Future.microtask(_bootstrap);
    return const AuthSession.unknown();
  }

  Dio get _rawDio => ref.read(rawDioProvider);
  SecureTokenStore get _tokenStore => ref.read(secureTokenStoreProvider);

  Future<void> _bootstrap() async {
    final savedRefreshToken = await _tokenStore.readRefreshToken();
    // `ref.mounted` è `false` se questo provider è già stato distrutto nel
    // frattempo (es. il container/l'app è stato chiuso mentre l'`await`
    // sopra era in corso). Scrivere su `state` dopo la distruzione
    // lancerebbe un errore — è lo stesso controllo che Riverpod raccomanda
    // ogni volta che si fa qualcosa di asincrono dentro un Notifier.
    if (!ref.mounted) return;

    if (savedRefreshToken == null) {
      state = const AuthSession.unauthenticated();
      return;
    }
    // C'è un refresh token da una sessione precedente: proviamo subito a
    // scambiarlo con un nuovo access token, senza chiedere di nuovo email e
    // password. Se il refresh token è scaduto/non più valido, l'utente
    // finisce comunque al login: nessun problema, è lo scenario normale.
    try {
      await _exchangeRefreshToken(savedRefreshToken);
    } on ApiException {
      if (!ref.mounted) return;
      await _tokenStore.clear();
      if (!ref.mounted) return;
      state = const AuthSession.unauthenticated();
    }
  }

  /// Chiamato dalla schermata di login col form compilato dall'utente.
  Future<void> login({
    required String identifier,
    required String password,
  }) async {
    try {
      final response = await _rawDio.post<Map<String, dynamic>>(
        '/v1/login',
        data: {'identifier': identifier, 'password': password},
      );
      await _applyAuthTokens(response.data!);
    } on DioException catch (e) {
      throw ApiException.fromDioException(
        e,
        statusToKey: const {
          400: 'errors.invalid_payload',
          401: 'errors.invalid_credentials',
        },
      );
    }
  }

  /// Usato sia dal bootstrap all'avvio sia dall'interceptor di
  /// `api_client.dart` quando una chiamata autenticata riceve un 401
  /// (access token scaduto): tenta UN refresh, e se va a buon fine
  /// aggiorna la sessione con la nuova coppia di token (il backend li
  /// ruota entrambi ad ogni refresh, vedi `services/registry`).
  Future<void> refreshSession() async {
    final savedRefreshToken = await _tokenStore.readRefreshToken();
    if (savedRefreshToken == null) {
      state = const AuthSession.unauthenticated();
      throw const ApiException('errors.invalid_or_expired_token');
    }
    try {
      await _exchangeRefreshToken(savedRefreshToken);
    } on ApiException {
      await _tokenStore.clear();
      state = const AuthSession.unauthenticated();
      rethrow;
    }
  }

  Future<void> _exchangeRefreshToken(String refreshToken) async {
    try {
      final response = await _rawDio.post<Map<String, dynamic>>(
        '/v1/refresh',
        data: {'refresh_token': refreshToken},
      );
      await _applyAuthTokens(response.data!);
    } on DioException catch (e) {
      throw ApiException.fromDioException(
        e,
        statusToKey: const {
          400: 'errors.invalid_payload',
          401: 'errors.invalid_or_expired_token',
        },
      );
    }
  }

  /// Comune a login e refresh: entrambi rispondono con la stessa forma
  /// `{token, refresh_token}` (vedi contratto API in `docs/adr/0022-...`).
  Future<void> _applyAuthTokens(Map<String, dynamic> body) async {
    final accessToken = body['token'] as String;
    final newRefreshToken = body['refresh_token'] as String;

    await _tokenStore.saveRefreshToken(newRefreshToken);

    final claims = decodeJwtClaims(accessToken);
    state = AuthSession(
      status: AuthStatus.authenticated,
      accessToken: accessToken,
      claims: claims,
    );
  }

  /// Il backend non invalida l'access token lato server (limite noto,
  /// ADR-0013): possiamo solo invalidare il refresh token (così non potrà
  /// più essere usato per ottenere nuovi access token) e scartare tutto
  /// localmente. Se la chiamata di rete fallisce (es. offline) usciamo
  /// comunque dalla sessione localmente: dal punto di vista dell'utente
  /// "esci" deve funzionare sempre, anche senza connessione.
  Future<void> logout() async {
    final savedRefreshToken = await _tokenStore.readRefreshToken();
    if (savedRefreshToken != null) {
      try {
        await _rawDio.post<void>(
          '/v1/logout',
          data: {'refresh_token': savedRefreshToken},
        );
      } on DioException {
        // Ignorato volutamente: vedi commento sopra.
      }
    }
    await _tokenStore.clear();
    state = const AuthSession.unauthenticated();
  }

  /// Imposta la password per un account creato da un admin (oggi il link
  /// di attivazione viene inoltrato a mano, non c'è ancora invio email
  /// automatico — vedi `docs/funzionale/registry.md`). Non modifica lo
  /// stato di sessione: dopo l'attivazione l'utente fa comunque login
  /// normalmente con le credenziali appena impostate.
  Future<void> activateAccount({
    required String token,
    required String password,
  }) async {
    try {
      await _rawDio.post<void>(
        '/v1/users/activate',
        data: {'token': token, 'password': password},
      );
    } on DioException catch (e) {
      throw ApiException.fromDioException(
        e,
        statusToKey: const {
          400: 'errors.invalid_payload',
          401: 'errors.invalid_or_expired_token',
        },
      );
    }
  }

  /// Chiede l'invio dell'email di reimpostazione password. Il backend
  /// risponde sempre allo stesso modo (204), che l'identifier corrisponda o
  /// no a un account reale — mai rivelare l'esistenza di un account, vedi
  /// docs/adr/0024-recupero-password.md. Per questo la UI (vedi
  /// forgot_password_screen.dart) mostra sempre lo stesso messaggio di
  /// conferma, indipendentemente dalla risposta.
  Future<void> requestPasswordReset({required String identifier}) async {
    try {
      await _rawDio.post<void>(
        '/v1/password/reset/request',
        data: {'identifier': identifier},
      );
    } on DioException catch (e) {
      throw ApiException.fromDioException(
        e,
        statusToKey: const {400: 'errors.invalid_payload'},
      );
    }
  }

  /// Imposta una nuova password a partire dal token ricevuto via email.
  /// Non modifica lo stato di sessione, come `activateAccount`: dopo il
  /// reset l'utente fa comunque login normalmente con la nuova password.
  Future<void> confirmPasswordReset({
    required String token,
    required String password,
  }) async {
    try {
      await _rawDio.post<void>(
        '/v1/password/reset/confirm',
        data: {'token': token, 'password': password},
      );
    } on DioException catch (e) {
      throw ApiException.fromDioException(
        e,
        statusToKey: const {
          400: 'errors.invalid_payload',
          401: 'errors.invalid_or_expired_token',
        },
      );
    }
  }
}

final authControllerProvider = NotifierProvider<AuthController, AuthSession>(
  AuthController.new,
);
