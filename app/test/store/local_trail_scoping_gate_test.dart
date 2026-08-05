import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guards for the 38.1 store-level scoping invariants
/// (D-04/D-07/D-08/D-09/D-12) that cannot be exercised behaviourally
/// because there is no ObjectBox test harness for plain `flutter test`
/// (see `test/store/local_trail_store_test.dart`'s file header) -- same
/// rationale as `test/provider/trail/local_trail_addressing_gate_test.dart`
/// and `test/store/local_trail_retirement_gate_test.dart`.
void main() {
  final libDir = Directory('lib');

  String readCodeOnly(String path) {
    expect(
      libDir.existsSync(),
      isTrue,
      reason:
          'This test must be run with `flutter test`\'s working directory '
          'set to "app/" (e.g. "cd app && flutter test").',
    );
    final source = File(path).readAsStringSync();
    return source
        .split('\n')
        .where((line) => !RegExp(r'^\s*//').hasMatch(line))
        .join('\n');
  }

  group('deleteLocalTrailRow (D-07, store half of CR-01)', () {
    test('is declared LocalRowDeleteOutcome, not void or bool -- neither '
        'can tell the caller whether the ROW is gone', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      expect(
        codeOnly.contains('LocalRowDeleteOutcome deleteLocalTrailRow('),
        isTrue,
        reason:
            'deleteLocalTrailRow was renamed, removed, or reverted to '
            'void/bool. Re-point this gate rather than deleting it -- '
            'callers must be able to tell "removed", "demoted because '
            'another account still holds it" and "no row I was allowed to '
            'touch" apart (CR-01, CR-02).',
      );
      expect(
        codeOnly.contains('void deleteLocalTrailRow(') ||
            codeOnly.contains('bool deleteLocalTrailRow('),
        isFalse,
        reason:
            'A void deleteLocalTrailRow cannot distinguish "matched and '
            'removed" from "no row owned by this account carried this '
            'localId" -- the CR-01 failure: the caller deleted photo files '
            'and reported success for an operation that deleted nothing it '
            'was allowed to delete. A bool cannot distinguish "the row is '
            'gone" from "the row was kept for another account" -- the '
            'CR-02 failure: only on the former may the caller reclaim '
            'library/<serverId>/.',
      );
    });

    test('returns all three LocalRowDeleteOutcome values from the branches '
        'that own them', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      final start = codeOnly.indexOf(
        'LocalRowDeleteOutcome deleteLocalTrailRow(',
      );
      expect(start, isNot(-1));
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      expect(
        RegExp(
          r'if \(entity == null\) return LocalRowDeleteOutcome\.noMatch;',
        ).hasMatch(body),
        isTrue,
        reason:
            'The entity == null (no-match) path must report noMatch so the '
            'caller can refuse to delete files or report deleted (D-07).',
      );
      expect(
        RegExp(
          r'trailBox\.remove\(entity\.obxId\);\s*'
          r'return LocalRowDeleteOutcome\.removed;',
        ).hasMatch(body),
        isTrue,
        reason:
            'The branch that actually removes the row must report '
            '`removed` -- that is what tells the caller library/<serverId>/ '
            'has no referent left and may be reclaimed. Inverting this '
            'deletes another account\'s downloaded photos.',
      );
      expect(
        RegExp(
          r'trailBox\.put\(entity\);\s*'
          r'return LocalRowDeleteOutcome\.demotedStillHeld;',
        ).hasMatch(body),
        isTrue,
        reason:
            'The keep-the-row branch must report `demotedStillHeld`, never '
            '`removed`: its library/<serverId>/ files still belong to the '
            'account holding this trail in savedByUserIds.',
      );
    });

    test('gates the row removal on shouldDeleteUploadedRow -- CR-02: an '
        'unconditional remove destroys another account\'s library entry', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      final start = codeOnly.indexOf(
        'LocalRowDeleteOutcome deleteLocalTrailRow(',
      );
      expect(start, isNot(-1));
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      final guardIdx = body.indexOf('!shouldDeleteUploadedRow(');
      final removeIdx = body.indexOf('trailBox.remove(');
      expect(
        guardIdx,
        isNot(-1),
        reason:
            'TrailEntity.id is @Unique(onConflict: replace), so once '
            'writeServerTrailId has stamped a server id another account\'s '
            'download lands in THIS row. Removing it unconditionally '
            'destroys that account\'s savedByUserIds entry and orphans '
            'library/<serverId>/ forever -- there is no library/ sweep '
            'anywhere in the app. This is the same guard '
            'retireUploadedLocalTrail uses for the other way a capture '
            'ends.',
      );
      expect(removeIdx, isNot(-1));
      expect(
        guardIdx < removeIdx,
        isTrue,
        reason:
            'A guard checked after the removal is no guard at all.',
      );
    });
  });

  group('isOwnLiveCapture (D-04/D-12/T-38.1-07)', () {
    test('body scopes by TrailEntity_.owner.equals(accountId) and does not '
        'call toModel()', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      final start = codeOnly.indexOf('bool isOwnLiveCapture(');
      expect(start, isNot(-1));
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      expect(
        body.contains('TrailEntity_.owner.equals(accountId)'),
        isTrue,
        reason:
            'This is the security invariant CR-01/CR-03 both trace back '
            'to: without the owner clause, a destructive action gated on '
            'this predicate could match another account\'s row.',
      );
      expect(
        body.contains('toModel()'),
        isFalse,
        reason:
            'A destructive-action gate must not silently flip because the '
            'row\'s cached GPX stopped parsing (CR-02, phase 36, same '
            'reasoning as readLocalTrailServerId). isOwnLiveCapture reads '
            'the row\'s own owner/syncState directly, never through '
            'toModel().',
      );
    });
  });

  group('applyServerTrailToLibraryRow waypointsAreAuthoritative (D-08/D-09, '
      'CR-02)', () {
    test('signature declares bool waypointsAreAuthoritative = true', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      expect(
        codeOnly.contains('bool waypointsAreAuthoritative = true'),
        isTrue,
        reason:
            'The default must stay true (D-08): the D-14 fetch trigger in '
            'trail_provider.dart is a full server response and must keep '
            'pruning, or WR-05 (server-side waypoint deletes never '
            'propagating) regresses.',
      );
    });

    test('the prune call only runs inside the guarded branch', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      final start = codeOnly.indexOf('void applyServerTrailToLibraryRow(');
      expect(start, isNot(-1));
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      final guardIdx = body.indexOf('if (waypointsAreAuthoritative)');
      final pruneIdx = body.indexOf('waypointBox.remove(');
      expect(guardIdx, isNot(-1));
      expect(pruneIdx, isNot(-1));
      expect(
        guardIdx < pruneIdx,
        isTrue,
        reason:
            'CR-02: one failed waypoint PATCH used to unconditionally '
            'prune a still-live waypoint from the offline copy. The prune '
            'must only run when the caller has told us the incoming '
            'waypoint list is complete.',
      );
      expect(
        body.contains('entity.waypoints.add('),
        isTrue,
        reason:
            'D-09: when the incoming list is not authoritative, an '
            'existing waypoint absent from it must be re-added as-is, not '
            'silently dropped.',
      );
    });

    test('trail_create_screen.dart wires waypointsAreAuthoritative to '
        '!result.hadWaypointFailures', () {
      final codeOnly = readCodeOnly('lib/routes/trail_create_screen.dart');

      expect(
        codeOnly.contains(
          'waypointsAreAuthoritative: !result.hadWaypointFailures',
        ),
        isTrue,
        reason:
            'Without this, the D-13 save trigger silently prunes from a '
            'known-incomplete waypoint list on any failed waypoint '
            'create/update (CR-02).',
      );
    });

    test('trail_provider.dart (the D-14 fetch trigger) does not pass '
        'waypointsAreAuthoritative -- it must stay on the authoritative '
        'default or WR-05 regresses', () {
      final source = File(
        'lib/provider/trail/trail_provider.dart',
      ).readAsStringSync();

      expect(
        source.contains('waypointsAreAuthoritative'),
        isFalse,
        reason:
            'The D-14 fetch trigger is a full server response and must '
            'stay authoritative, or WR-05 (server-side waypoint deletes '
            'never propagating) regresses.',
      );
    });
  });
}
