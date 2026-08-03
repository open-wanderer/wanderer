import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for `retireUploadedLocalTrail`'s invariants.
///
/// `flutter test` cannot open an ObjectBox `Store` (Phase 31 established
/// there is no ObjectBox test harness for plain `flutter test`, restated in
/// `local_trail_store_test.dart`'s own header), and there is no PocketBase in
/// the test environment. So the transaction this function runs in has NO
/// behavioural surface here at all -- not a hard-to-reach one, an ABSENT one.
/// `local_trail_store_test.dart` covers the pure half of this plan
/// (`shouldDeleteUploadedRow`, `resolveLocalSaveModeForRow`) with real
/// assertions on real inputs; this file pins only the structural invariants a
/// reader cannot see from the call site -- the cascade, the delete-or-demote
/// choice, and (once Task 2 lands) the drain's ordering. The end-to-end
/// behaviour is covered by the device re-test in `36-14-PLAN.md`'s
/// `<verification>` (UAT Test 5).
///
/// The technique mirrors `test/components/trail/trail_dropdown_delete_gate_test.dart`:
/// slice the function body out of the source file and assert on substrings
/// and their relative order, rather than asserting "the source contains
/// retireUploadedLocalTrail" and leaving it at that -- a call-site presence
/// check would pass for a call placed in the `catch` block just as easily as
/// the right place.
void main() {
  final libDir = Directory('lib');

  /// [retireUploadedLocalTrail]'s body, isolated from the rest of the file.
  ///
  /// Comment-stripped first so a `reason:` string quoting one of the
  /// asserted substrings (e.g. a doc comment mentioning
  /// `shouldDeleteUploadedRow(`) can never make an assertion pass on prose
  /// instead of code.
  String retirementBody() {
    expect(
      libDir.existsSync(),
      isTrue,
      reason:
          'This test must be run with `flutter test`\'s working directory '
          'set to `app/` (e.g. "cd app && flutter test").',
    );

    final source = File(
      'lib/util/local_trail_store.dart',
    ).readAsStringSync();
    final codeOnly = source
        .split('\n')
        .where((line) => !RegExp(r'^\s*//').hasMatch(line))
        .join('\n');

    final sigStart = codeOnly.indexOf(
      'void retireUploadedLocalTrail(Store store, String localId) {',
    );
    expect(
      sigStart,
      isNot(-1),
      reason:
          'retireUploadedLocalTrail was renamed or its signature changed. '
          'Re-point this gate rather than deleting it -- the invariant '
          'still matters.',
    );

    final bodyEnd = codeOnly.indexOf('\n}', sigStart);
    expect(
      bodyEnd,
      isNot(-1),
      reason: 'Could not find the end of retireUploadedLocalTrail\'s body.',
    );

    return codeOnly.substring(sigStart, bodyEnd);
  }

  group('retireUploadedLocalTrail -- the write transaction', () {
    test('runs inside one write transaction', () {
      final body = retirementBody();

      expect(
        body.contains('runInTransaction(TxMode.write'),
        isTrue,
        reason:
            'Splitting the lookup and the removes across two transactions '
            'reintroduces a window in which a crash leaves a half-retired '
            'row -- a trail present neither as a resumable local capture '
            'nor as a clean server entry.',
      );
    });

    test('is keyed on TrailEntity_.localId, not the server id', () {
      final body = retirementBody();

      expect(
        body.contains('TrailEntity_.localId.equals(localId)'),
        isTrue,
        reason:
            'The local id is the only handle the drain holds for the '
            'whole resume sequence (D-05); keying on anything else breaks '
            'retirement for a row the drain is mid-sequence with.',
      );
    });

    test('gates the delete on shouldDeleteUploadedRow', () {
      final body = retirementBody();

      expect(
        body.contains('shouldDeleteUploadedRow('),
        isTrue,
        reason:
            'An unconditional delete destroys the offline library entry '
            'of any account that downloaded the trail during the upload '
            'window -- shouldDeleteUploadedRow is what prevents that.',
      );
    });

    test('removes the waypoint children via waypointBox.remove(', () {
      final body = retirementBody();

      expect(
        body.contains('waypointBox.remove('),
        isTrue,
        reason:
            'TrailLibraryNotifier.deleteTrail removes the trail row alone '
            'and leaks its waypoints -- "simplifying" this into a call to '
            'that one leaks a set of waypoints per upload.',
      );
    });

    test('the demote branch actually clears entity.owner', () {
      final body = retirementBody();

      expect(
        body.contains('entity.owner = null'),
        isTrue,
        reason:
            'Without giving up ownership, a demoted row still matches '
            'readOwnLocalTrails\' owner clause and the orphan class comes '
            'back through the side door.',
      );
    });
  });
}
