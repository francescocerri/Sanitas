import '../jwt.dart';

/// Le 3 fasi in cui può trovarsi la sessione dell'utente:
/// - [unknown]: l'app è appena partita, non sappiamo ancora se c'è un
///   refresh token salvato da una sessione precedente (stiamo controllando).
/// - [authenticated]: abbiamo un access token valido in memoria.
/// - [unauthenticated]: nessuna sessione valida, l'utente deve fare login.
///
/// Il router (`lib/router.dart`) usa questo stato per decidere se mostrare
/// le schermate protette o rimandare al login.
enum AuthStatus { unknown, authenticated, unauthenticated }

/// Stato immutabile della sessione. "Immutabile" vuol dire che non lo
/// modifichiamo mai sul posto: quando qualcosa cambia (login, refresh,
/// logout) creiamo una nuova istanza con [copyWith] e la pubblichiamo tramite
/// Riverpod — è il pattern standard in Flutter per lo stato gestito da un
/// Notifier, rende molto più facile capire "quando" e "perché" la UI si
/// aggiorna (si aggiorna sempre e solo quando arriva una nuova istanza).
class AuthSession {
  const AuthSession({
    required this.status,
    this.accessToken,
    this.claims,
  });

  const AuthSession.unknown() : this(status: AuthStatus.unknown);
  const AuthSession.unauthenticated() : this(status: AuthStatus.unauthenticated);

  final AuthStatus status;

  /// L'access token JWT corrente. Vive SOLO in memoria (mai su disco): si
  /// perde a ogni riavvio dell'app o reload della pagina web, e viene
  /// rigenerato al bootstrap con un refresh silenzioso (vedi
  /// `auth_controller.dart`). Tenerlo solo in RAM riduce la superficie di
  /// esposizione rispetto a salvarlo in uno storage persistente.
  final String? accessToken;

  /// I claim decodificati dell'access token corrente (ruoli, permessi,
  /// scadenza) — evita di dover ri-decodificare il token ogni volta che la
  /// UI deve sapere, ad esempio, se l'utente ha un certo permesso.
  final JwtClaims? claims;

  bool get isAuthenticated => status == AuthStatus.authenticated;

  AuthSession copyWith({
    AuthStatus? status,
    String? accessToken,
    JwtClaims? claims,
  }) {
    return AuthSession(
      status: status ?? this.status,
      accessToken: accessToken ?? this.accessToken,
      claims: claims ?? this.claims,
    );
  }
}
