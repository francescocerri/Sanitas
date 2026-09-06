import 'package:dio/dio.dart';

/// Un errore di chiamata API già tradotto in una CHIAVE di traduzione
/// i18n (non nel testo finale — quello lo decide la UI con `.tr()` al
/// momento di mostrarlo, in modo che cambi lingua automaticamente se
/// l'utente cambia lingua). Il backend risponde sempre con
/// `{"error": "messaggio in inglese"}` (vedi ADR-0015): non mostriamo MAI
/// quel messaggio grezzo all'utente finale, solo nei log per debug.
class ApiException implements Exception {
  const ApiException(this.translationKey, {this.debugMessage});

  /// Chiave dentro `errors.*` nei file di traduzione
  /// (`assets/translations/it.json`/`en.json`), es. "errors.invalid_credentials".
  final String translationKey;

  /// Messaggio originale del backend (inglese), solo per log/debug — mai
  /// mostrato nella UI.
  final String? debugMessage;

  @override
  String toString() => 'ApiException($translationKey): $debugMessage';

  /// Traduce un [DioException] (l'eccezione che `dio` lancia per ogni
  /// richiesta fallita) in un [ApiException] con la chiave i18n giusta.
  ///
  /// [statusToKey] mappa lo status HTTP alla chiave di errore SPECIFICA di
  /// quella chiamata (es. per il login, 401 vuol dire "credenziali
  /// sbagliate"; per il cambio password, 401 vuol dire "vecchia password
  /// sbagliata" — stesso status code, significato diverso a seconda
  /// dell'endpoint, quindi la mappa la passa chi chiama, non è fissa qui).
  factory ApiException.fromDioException(
    DioException e, {
    required Map<int, String> statusToKey,
  }) {
    // Nessuna risposta arrivata (rete assente, timeout, server irraggiungibile):
    // non è un errore "applicativo", è un problema di connessione.
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return ApiException('errors.network', debugMessage: e.message);
    }

    final status = e.response?.statusCode;
    final backendMessage = _extractBackendErrorMessage(e);

    if (status != null && statusToKey.containsKey(status)) {
      return ApiException(statusToKey[status]!, debugMessage: backendMessage);
    }

    return ApiException(
      'errors.unknown',
      debugMessage: backendMessage ?? e.message,
    );
  }

  static String? _extractBackendErrorMessage(DioException e) {
    final data = e.response?.data;
    if (data is Map && data['error'] is String) {
      return data['error'] as String;
    }
    return null;
  }
}
