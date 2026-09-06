// Decodifica (senza verificare la firma) il payload di un access token JWT
// emesso da `registry`. Non è un controllo di sicurezza: la firma RS256 viene
// già verificata dai servizi backend ad ogni richiesta autenticata — qui ci
// serve solo leggere i claim (ruoli, permessi, scadenza) per decidere cosa
// mostrare nell'interfaccia, senza dover fare una chiamata di rete in più.
//
// Non usiamo nessuna libreria esterna per questo: un JWT è solo tre parti
// separate da "." (header.payload.signature), e il payload è JSON codificato
// in Base64URL — bastano le funzioni di `dart:convert` già incluse in Flutter.
import 'dart:convert';

/// I campi del payload JWT che ci interessano lato client. Il backend
/// (`services/registry/internal/user/jwt.go`) ne include altri (iat, ecc.)
/// che qui ignoriamo perché non servono all'interfaccia.
class JwtClaims {
  const JwtClaims({
    required this.subject,
    required this.username,
    required this.roles,
    required this.permissions,
    required this.expiresAt,
  });

  /// `sub`: id dell'utente (UUID).
  final String subject;
  final String username;
  final List<String> roles;
  final List<String> permissions;

  /// `exp`: scadenza del token, in UTC. Il backend la esprime come "secondi
  /// da epoch" (claim JWT standard) — la convertiamo subito in un
  /// `DateTime` così il resto del codice non deve più pensarci.
  final DateTime expiresAt;

  /// true se il token è già scaduto (o scade nei prossimi [skew], per
  /// lasciare un margine e rinnovarlo un po' in anticipo invece di aspettare
  /// che una richiesta fallisca con 401).
  bool isExpired({Duration skew = const Duration(seconds: 30)}) {
    return DateTime.now().toUtc().add(skew).isAfter(expiresAt);
  }

  bool hasPermission(String permission) => permissions.contains(permission);
}

/// Lanciata se il token non è un JWT valido (formato inatteso). In pratica
/// non dovrebbe mai succedere con token emessi da `registry`, ma un client
/// non deve mai fidarsi ciecamente di un input esterno.
class MalformedJwtException implements Exception {
  const MalformedJwtException(this.message);
  final String message;

  @override
  String toString() => 'MalformedJwtException: $message';
}

JwtClaims decodeJwtClaims(String token) {
  final parts = token.split('.');
  if (parts.length != 3) {
    throw const MalformedJwtException('il token non ha 3 parti separate da "."');
  }

  final payloadJson = _decodeBase64UrlSegment(parts[1]);
  final Map<String, dynamic> payload;
  try {
    payload = jsonDecode(payloadJson) as Map<String, dynamic>;
  } on FormatException catch (e) {
    throw MalformedJwtException('payload non è JSON valido: $e');
  }

  final expSeconds = payload['exp'];
  if (expSeconds is! int) {
    throw const MalformedJwtException('claim "exp" mancante o non numerico');
  }

  return JwtClaims(
    subject: payload['sub'] as String? ?? '',
    username: payload['username'] as String? ?? '',
    roles: (payload['roles'] as List<dynamic>? ?? const []).cast<String>(),
    permissions: (payload['permissions'] as List<dynamic>? ?? const []).cast<String>(),
    expiresAt: DateTime.fromMillisecondsSinceEpoch(expSeconds * 1000, isUtc: true),
  );
}

/// Il Base64 di un JWT è "Base64URL" (usa "-"/"_" al posto di "+"/"/") e
/// spesso arriva senza il padding "=" finale che Base64 standard richiede.
/// `dart:convert` si aspetta il padding corretto, quindi lo ricostruiamo a
/// mano prima di decodificare.
String _decodeBase64UrlSegment(String segment) {
  final normalized = base64Url.normalize(segment);
  return utf8.decode(base64Url.decode(normalized));
}
