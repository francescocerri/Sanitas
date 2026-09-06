import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';
import 'package:sanitas_app/core/api_exception.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/auth/secure_token_store.dart';
import 'package:sanitas_app/core/raw_api_client.dart';

/// `dio` non è pensato per essere sottoclassato direttamente nei test (i
/// suoi metodi `post`/`get` sono generici, `mocktail` li gestisce meglio
/// tramite `Mock` + `when`/`thenAnswer` che tramite un finto sottotipo).
class MockDio extends Mock implements Dio {}

/// `SecureTokenStore` normalmente appoggia su `flutter_secure_storage`, che
/// userebbe canali nativi non disponibili in un test Dart puro — qui la
/// sostituiamo con una versione in memoria, sufficiente per verificare la
/// logica di `AuthController` senza toccare nulla di piattaforma.
class InMemoryTokenStore extends SecureTokenStore {
  InMemoryTokenStore() : super(const FlutterSecureStorage());

  String? _refreshToken;

  @override
  Future<String?> readRefreshToken() async => _refreshToken;

  @override
  Future<void> saveRefreshToken(String token) async => _refreshToken = token;

  @override
  Future<void> clear() async => _refreshToken = null;
}

/// Stessa idea del helper in `jwt_test.dart`: un JWT sintatticamente valido
/// (3 parti Base64URL) ma con firma finta, sufficiente perché
/// `decodeJwtClaims` (chiamato da `AuthController` dopo login/refresh) lo
/// legga correttamente.
String _fakeJwt({List<String> permissions = const []}) {
  String encodeSegment(Map<String, dynamic> data) {
    return base64Url.encode(utf8.encode(jsonEncode(data))).replaceAll('=', '');
  }

  final header = encodeSegment({'alg': 'RS256', 'typ': 'JWT'});
  final body = encodeSegment({
    'sub': 'user-1',
    'username': 'mario.rossi',
    'roles': const ['shift_manager'],
    'permissions': permissions,
    'exp':
        DateTime.now()
            .toUtc()
            .add(const Duration(hours: 24))
            .millisecondsSinceEpoch ~/
        1000,
  });
  return '$header.$body.fake-signature';
}

Response<Map<String, dynamic>> _authTokensResponse(
  RequestOptions options, {
  String? accessToken,
}) {
  return Response(
    requestOptions: options,
    statusCode: 200,
    data: {
      'token': accessToken ?? _fakeJwt(),
      'refresh_token': 'refresh-token-value',
    },
  );
}

void main() {
  late MockDio mockDio;
  late InMemoryTokenStore tokenStore;
  late ProviderContainer container;

  setUp(() {
    mockDio = MockDio();
    tokenStore = InMemoryTokenStore();
    container = ProviderContainer(
      overrides: [
        rawDioProvider.overrideWithValue(mockDio),
        secureTokenStoreProvider.overrideWithValue(tokenStore),
      ],
    );
    addTearDown(container.dispose);
  });

  /// `AuthController.build()` avvia il bootstrap in un `Future.microtask`
  /// (vedi commento nel file sorgente): non possiamo aspettarlo con `await`
  /// direttamente, quindi lasciamo girare un giro di event loop prima di
  /// controllare lo stato — è l'equivalente per i test di "aspetta che il
  /// prossimo microtask sia stato eseguito". Un singolo
  /// `Future.delayed(Duration.zero)` basta anche per catene di più `await`
  /// consecutivi: Dart esegue SEMPRE tutti i microtask in coda prima di
  /// passare al prossimo timer, per quanto lunga sia la catena.
  Future<void> pumpEventLoop() => Future<void>.delayed(Duration.zero);

  /// Ogni test parte da uno stato "sistemato": il provider è stato
  /// costruito (altrimenti il bootstrap partirebbe più tardi, in mezzo alle
  /// azioni del test, e potrebbe correre in parallelo con esse) e il
  /// bootstrap ha già finito di girare. Nel `setUp` non c'è ancora nessun
  /// refresh token salvato, quindi il bootstrap approda sempre a
  /// "non autenticato" — da lì ogni test parte con le sue azioni.
  Future<void> settleBootstrap() async {
    container.read(authControllerProvider);
    await pumpEventLoop();
  }

  group('bootstrap', () {
    test('nessun refresh token salvato -> stato non autenticato', () async {
      await settleBootstrap();

      expect(
        container.read(authControllerProvider).status,
        AuthStatus.unauthenticated,
      );
    });
  });

  group('login', () {
    test('successo -> stato autenticato con i claim decodificati', () async {
      await settleBootstrap();

      when(
        () => mockDio.post<Map<String, dynamic>>(
          '/v1/login',
          data: any(named: 'data'),
        ),
      ).thenAnswer((invocation) async {
        final options = RequestOptions(path: '/v1/login');
        return _authTokensResponse(
          options,
          accessToken: _fakeJwt(permissions: ['shifts:read']),
        );
      });

      await container
          .read(authControllerProvider.notifier)
          .login(identifier: 'mario.rossi', password: 'correct-password');

      final session = container.read(authControllerProvider);
      expect(session.status, AuthStatus.authenticated);
      expect(session.claims?.username, 'mario.rossi');
      expect(session.claims?.hasPermission('shifts:read'), isTrue);
      expect(await tokenStore.readRefreshToken(), 'refresh-token-value');
    });

    test(
      '401 dal backend -> ApiException con la chiave "credenziali sbagliate"',
      () async {
        await settleBootstrap();

        when(
          () => mockDio.post<Map<String, dynamic>>(
            '/v1/login',
            data: any(named: 'data'),
          ),
        ).thenThrow(
          DioException(
            requestOptions: RequestOptions(path: '/v1/login'),
            response: Response(
              requestOptions: RequestOptions(path: '/v1/login'),
              statusCode: 401,
              data: {'error': 'invalid credentials'},
            ),
          ),
        );

        expect(
          () => container
              .read(authControllerProvider.notifier)
              .login(identifier: 'mario.rossi', password: 'wrong-password'),
          throwsA(
            isA<ApiException>().having(
              (e) => e.translationKey,
              'translationKey',
              'errors.invalid_credentials',
            ),
          ),
        );
      },
    );
  });

  group('logout', () {
    test(
      'pulisce sessione e refresh token anche se la chiamata di rete fallisce',
      () async {
        await settleBootstrap();
        await tokenStore.saveRefreshToken('some-refresh-token');

        when(() => mockDio.post<void>('/v1/logout', data: any(named: 'data')))
            .thenThrow(
              DioException(requestOptions: RequestOptions(path: '/v1/logout')),
            );

        await container.read(authControllerProvider.notifier).logout();

        expect(
          container.read(authControllerProvider).status,
          AuthStatus.unauthenticated,
        );
        expect(await tokenStore.readRefreshToken(), isNull);
      },
    );
  });
}
