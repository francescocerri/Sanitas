import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/jwt.dart';

/// Costruisce un JWT "finto" (header/payload arbitrari, firma non
/// verificata) con la stessa forma di quelli emessi da `registry`: ci basta
/// per testare `decodeJwtClaims`, che non verifica MAI la firma (vedi
/// commento in jwt.dart) — non serve una vera chiave RSA per questi test.
String _fakeJwt(Map<String, dynamic> payload) {
  String encodeSegment(Map<String, dynamic> data) {
    return base64Url.encode(utf8.encode(jsonEncode(data))).replaceAll('=', '');
  }

  final header = encodeSegment({'alg': 'RS256', 'typ': 'JWT'});
  final body = encodeSegment(payload);
  return '$header.$body.fake-signature';
}

void main() {
  group('decodeJwtClaims', () {
    test('legge tutti i claim attesi da un token registry', () {
      final expiry = DateTime.utc(2030, 1, 1);
      final token = _fakeJwt({
        'sub': 'user-123',
        'username': 'mario.rossi',
        'roles': ['shift_manager'],
        'permissions': ['shifts:read', 'shifts:write'],
        'exp': expiry.millisecondsSinceEpoch ~/ 1000,
      });

      final claims = decodeJwtClaims(token);

      expect(claims.subject, 'user-123');
      expect(claims.username, 'mario.rossi');
      expect(claims.roles, ['shift_manager']);
      expect(claims.permissions, ['shifts:read', 'shifts:write']);
      expect(claims.expiresAt, expiry);
    });

    test('hasPermission trova un permesso presente e non uno assente', () {
      final token = _fakeJwt({
        'sub': 'u',
        'username': 'u',
        'roles': [],
        'permissions': ['users:manage'],
        'exp': DateTime.now().toUtc().add(const Duration(hours: 1)).millisecondsSinceEpoch ~/ 1000,
      });

      final claims = decodeJwtClaims(token);

      expect(claims.hasPermission('users:manage'), isTrue);
      expect(claims.hasPermission('shifts:write'), isFalse);
    });

    test('isExpired è true per un token già scaduto', () {
      final token = _fakeJwt({
        'sub': 'u',
        'username': 'u',
        'roles': [],
        'permissions': [],
        'exp': DateTime.now().toUtc().subtract(const Duration(minutes: 5)).millisecondsSinceEpoch ~/ 1000,
      });

      expect(decodeJwtClaims(token).isExpired(), isTrue);
    });

    test('isExpired è false per un token valido a lungo', () {
      final token = _fakeJwt({
        'sub': 'u',
        'username': 'u',
        'roles': [],
        'permissions': [],
        'exp': DateTime.now().toUtc().add(const Duration(hours: 24)).millisecondsSinceEpoch ~/ 1000,
      });

      expect(decodeJwtClaims(token).isExpired(), isFalse);
    });

    test('lancia MalformedJwtException se il token non ha 3 parti', () {
      expect(() => decodeJwtClaims('non-e-un-jwt'), throwsA(isA<MalformedJwtException>()));
    });

    test('lancia MalformedJwtException se manca il claim "exp"', () {
      final token = _fakeJwt({'sub': 'u', 'username': 'u', 'roles': [], 'permissions': []});
      expect(() => decodeJwtClaims(token), throwsA(isA<MalformedJwtException>()));
    });
  });
}
