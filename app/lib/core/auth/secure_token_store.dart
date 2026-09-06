import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Incapsula la persistenza del refresh token. È l'UNICA cosa della
/// sessione che salviamo su disco (l'access token resta solo in memoria,
/// vedi `auth_state.dart`): il refresh token deve sopravvivere alla chiusura
/// dell'app per non costringere l'utente a rifare login ogni volta.
///
/// `flutter_secure_storage` usa, sotto al cofano, il Keychain su iOS e il
/// Keystore/EncryptedSharedPreferences su Android — storage cifrato a
/// livello di sistema operativo, non un semplice file di testo. Su Flutter
/// Web (dove non esiste un Keychain) ripiega su storage del browser cifrato
/// via WebCrypto: meno robusto delle controparti native, ma è il massimo
/// che un browser permette a un'app web.
class SecureTokenStore {
  SecureTokenStore(this._storage);

  final FlutterSecureStorage _storage;

  static const _refreshTokenKey = 'sanitas_refresh_token';

  Future<String?> readRefreshToken() => _storage.read(key: _refreshTokenKey);

  Future<void> saveRefreshToken(String token) =>
      _storage.write(key: _refreshTokenKey, value: token);

  Future<void> clear() => _storage.delete(key: _refreshTokenKey);
}

/// Provider Riverpod: espone una singola istanza condivisa di
/// [SecureTokenStore] a tutta l'app (es. ad `AuthController`), invece di
/// istanziare `FlutterSecureStorage` in più punti.
final secureTokenStoreProvider = Provider<SecureTokenStore>((ref) {
  return SecureTokenStore(const FlutterSecureStorage());
});
