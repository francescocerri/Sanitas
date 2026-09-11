/// Stato di copertura aggregato di una figura su un'occorrenza (vedi
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

/// Stato della PROPRIA prenotazione su una figura (se esiste) — diverso
/// dallo stato aggregato: una figura può essere "in attesa" perché
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

/// Le 4 figure che compongono un turno (vedi ADR-0025 "Aggiornamento") — un
/// enum fisso, non configurabile per comitato: composizione di un
/// equipaggio CRI, non una scelta specifica di Pavullo. Nomi delle
/// costanti in inglese, stessi valori esatti usati dal backend
/// (`ShiftRole.driver.name == "driver"`, ecc.), niente mappa di
/// conversione manuale da mantenere in sync.
enum ShiftRole { driver, leader, rescuer, observer }

/// Pubblica (non solo per `RoleCoverage.fromJson` qui sotto): riusata anche
/// da `shifts_providers.dart` per decodificare `PendingBooking.role`.
ShiftRole? parseShiftRole(String raw) {
  for (final role in ShiftRole.values) {
    if (role.name == raw) return role;
  }
  return null;
}

/// Una selezione dell'utente: una figura specifica su un'occorrenza
/// specifica — quello che la barra di selezione accumula e che
/// `requestBulkBookings` spedisce (vedi `shifts_providers.dart`).
typedef BookingSelection = ({ShiftOccurrence occurrence, ShiftRole role});

/// Le figure la cui conferma basta perché un turno sia "al completo" —
/// autista, leader e soccorritore. L'osservatore è facoltativo: un turno
/// con queste 3 confermate è pienamente operativo anche senza osservatore
/// (vincolo di dominio esplicitato dall'utente, vedi ADR-0025
/// "Aggiornamento").
const requiredRolesForCompletion = {
  ShiftRole.driver,
  ShiftRole.leader,
  ShiftRole.rescuer,
};

/// Copertura di una singola figura su un'occorrenza — quattro di queste
/// compongono `ShiftOccurrence.roles`, sempre nello stesso ordine di
/// `ShiftRole.values` (stesso ordine canonico del backend).
class RoleCoverage {
  const RoleCoverage({
    required this.role,
    required this.status,
    required this.myBookingStatus,
  });

  final ShiftRole role;
  final ShiftOccurrenceStatus status;
  final MyBookingStatus? myBookingStatus;

  /// Prenotabile da un nuovo volontario: libera, oppure già in attesa ma
  /// non ancora confermata (più richieste pending possono coesistere sulla
  /// stessa figura) — mai se il chiamante ha già una propria richiesta lì.
  bool get isBookable =>
      myBookingStatus == null && status != ShiftOccurrenceStatus.confirmed;

  factory RoleCoverage.fromJson(Map<String, dynamic> json) {
    final role = parseShiftRole(json['role'] as String);
    if (role == null) {
      throw FormatException('unknown role in response: ${json['role']}');
    }
    return RoleCoverage(
      role: role,
      status: _parseOccurrenceStatus(json['status'] as String),
      myBookingStatus: _parseMyBookingStatus(
        json['my_booking_status'] as String?,
      ),
    );
  }
}

/// Un'occorrenza calendario così come la restituisce
/// `GET /v1/shift-occurrences` — un turno-template su un giorno concreto,
/// con la copertura delle sue 4 figure già calcolata dal backend. Mai
/// persistita: generata al volo, vedi `shift.Occurrence` sul backend.
class ShiftOccurrence {
  const ShiftOccurrence({
    required this.templateId,
    required this.date,
    required this.weekday,
    required this.startTime,
    required this.endTime,
    required this.label,
    required this.roles,
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

  /// Sempre esattamente 4 elementi, uno per `ShiftRole`, nello stesso
  /// ordine di `ShiftRole.values`.
  final List<RoleCoverage> roles;

  /// Solo la parte data, formato `YYYY-MM-DD` — usata sia per la chiave di
  /// selezione sia per il payload della richiesta bulk.
  String get dateKey => date.toIso8601String().split('T').first;

  /// Chiave univoca template+data+figura, usata per la selezione multipla.
  String keyFor(ShiftRole role) => '$templateId|$dateKey|${role.name}';

  RoleCoverage coverageFor(ShiftRole role) =>
      roles.firstWhere((rc) => rc.role == role);

  /// Vero quando le 3 figure richieste (vedi [requiredRolesForCompletion])
  /// sono tutte confermate — l'osservatore, libero o meno, non influisce.
  bool get isComplete => roles
      .where((rc) => requiredRolesForCompletion.contains(rc.role))
      .every((rc) => rc.status == ShiftOccurrenceStatus.confirmed);

  factory ShiftOccurrence.fromJson(Map<String, dynamic> json) {
    return ShiftOccurrence(
      templateId: json['template_id'] as String,
      date: DateTime.parse(json['date'] as String),
      weekday: json['weekday'] as int,
      startTime: json['start_time'] as String,
      endTime: json['end_time'] as String,
      label: json['label'] as String,
      roles: (json['roles'] as List<dynamic>)
          .map((r) => RoleCoverage.fromJson(r as Map<String, dynamic>))
          .toList(),
    );
  }
}
