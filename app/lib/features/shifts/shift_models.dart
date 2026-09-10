/// Stato di copertura aggregato di un'occorrenza (vedi
/// `shift.OccurrenceStatus` in `services/shifts/internal/shift/occurrence.go`):
/// libero, in attesa (qualcuno ha richiesto ma nessuno è stato confermato) o
/// confermato. `pending` NON blocca la prenotazione da parte di altri
/// volontari — solo `confirmed` lo fa (vedi ADR-0025).
enum ShiftOccurrenceStatus { free, pending, confirmed }

ShiftOccurrenceStatus _parseOccurrenceStatus(String raw) {
  switch (raw) {
    case 'pending':
      return ShiftOccurrenceStatus.pending;
    case 'confirmed':
      return ShiftOccurrenceStatus.confirmed;
    default:
      return ShiftOccurrenceStatus.free;
  }
}

/// Stato della PROPRIA prenotazione su un'occorrenza (se esiste) — diverso
/// dallo stato aggregato: un'occorrenza può essere "in attesa" perché
/// qualcun altro l'ha richiesta, mentre la mia eventuale richiesta ha un suo
/// stato indipendente (vedi `my_booking_status` nel backend).
enum MyBookingStatus { pending, confirmed }

MyBookingStatus? _parseMyBookingStatus(String? raw) {
  switch (raw) {
    case 'pending':
      return MyBookingStatus.pending;
    case 'confirmed':
      return MyBookingStatus.confirmed;
    default:
      return null;
  }
}

/// Un'occorrenza calendario così come la restituisce
/// `GET /v1/shift-occurrences` — un turno-template su un giorno concreto,
/// con lo stato di copertura già calcolato dal backend. Mai persistita:
/// generata al volo, vedi `shift.Occurrence` sul backend.
class ShiftOccurrence {
  const ShiftOccurrence({
    required this.templateId,
    required this.date,
    required this.weekday,
    required this.startTime,
    required this.endTime,
    required this.label,
    required this.status,
    required this.myBookingStatus,
  });

  final String templateId;

  /// Solo la data (senza orario) — il backend la restituisce come
  /// `YYYY-MM-DDT00:00:00Z`.
  final DateTime date;

  /// 0 = domenica .. 6 = sabato, stessa convenzione di `time.Weekday` in Go.
  final int weekday;
  final String startTime;
  final String endTime;
  final String label;
  final ShiftOccurrenceStatus status;
  final MyBookingStatus? myBookingStatus;

  /// Prenotabile da un nuovo volontario: libero, oppure già in attesa ma
  /// non ancora confermato (più richieste pending possono coesistere sullo
  /// stesso slot) — mai se il chiamante ha già una propria richiesta lì.
  bool get isBookable =>
      myBookingStatus == null && status != ShiftOccurrenceStatus.confirmed;

  /// Chiave univoca template+data, usata per la selezione multipla e per
  /// distinguere occorrenze dello stesso template su giorni diversi.
  String get key => '$templateId|${date.toIso8601String().split('T').first}';

  factory ShiftOccurrence.fromJson(Map<String, dynamic> json) {
    return ShiftOccurrence(
      templateId: json['template_id'] as String,
      date: DateTime.parse(json['date'] as String),
      weekday: json['weekday'] as int,
      startTime: json['start_time'] as String,
      endTime: json['end_time'] as String,
      label: json['label'] as String,
      status: _parseOccurrenceStatus(json['status'] as String),
      myBookingStatus: _parseMyBookingStatus(
        json['my_booking_status'] as String?,
      ),
    );
  }
}
