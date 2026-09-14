import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/features/shifts/shift_models.dart';
import 'package:sanitas_app/features/shifts/shift_status_style.dart';

RoleCoverage _role(
  ShiftRole role,
  ShiftOccurrenceStatus status, {
  MyBookingStatus? myBookingStatus,
  String? volunteerId,
}) => RoleCoverage(
  role: role,
  status: status,
  myBookingStatus: myBookingStatus,
  volunteerId: volunteerId,
);

ShiftOccurrence _occurrence(List<RoleCoverage> roles) => ShiftOccurrence(
  templateId: 'tpl',
  date: DateTime(2026, 1, 1),
  weekday: 4,
  startTime: '20:00',
  endTime: '08:00',
  label: 'Turno',
  roles: roles,
);

void main() {
  group('aggregateStatus', () {
    test('complete when driver/leader/rescuer are all confirmed, regardless '
        'of the observer', () {
      final occurrence = _occurrence([
        _role(ShiftRole.driver, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.leader, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.rescuer, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.observer, ShiftOccurrenceStatus.free),
      ]);

      expect(aggregateStatus(occurrence), OccurrenceAggregateStatus.complete);
    });

    test('pending wins over free when an operative role is pending, even if '
        'another role is still free elsewhere on the same occurrence — the '
        'bug reported by the user (2 confirmed + 1 pending looked "free")', () {
      final occurrence = _occurrence([
        _role(ShiftRole.driver, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.leader, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.rescuer, ShiftOccurrenceStatus.pending),
        _role(ShiftRole.observer, ShiftOccurrenceStatus.free),
      ]);

      expect(aggregateStatus(occurrence), OccurrenceAggregateStatus.pending);
    });

    test('a pending observer alone (no operative role pending) does not turn '
        'the aggregate into pending', () {
      final occurrence = _occurrence([
        _role(ShiftRole.driver, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.leader, ShiftOccurrenceStatus.confirmed),
        _role(ShiftRole.rescuer, ShiftOccurrenceStatus.free),
        _role(ShiftRole.observer, ShiftOccurrenceStatus.pending),
      ]);

      expect(aggregateStatus(occurrence), OccurrenceAggregateStatus.free);
    });

    test('free when nothing is confirmed or pending on operative roles', () {
      final occurrence = _occurrence([
        _role(ShiftRole.driver, ShiftOccurrenceStatus.free),
        _role(ShiftRole.leader, ShiftOccurrenceStatus.free),
        _role(ShiftRole.rescuer, ShiftOccurrenceStatus.free),
        _role(ShiftRole.observer, ShiftOccurrenceStatus.free),
      ]);

      expect(aggregateStatus(occurrence), OccurrenceAggregateStatus.free);
    });
  });
}
