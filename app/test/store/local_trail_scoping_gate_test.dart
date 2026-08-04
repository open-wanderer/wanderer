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
    test('is declared bool, not void -- a void signature makes the '
        'no-match case indistinguishable from success', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      expect(
        codeOnly.contains('bool deleteLocalTrailRow('),
        isTrue,
        reason:
            'deleteLocalTrailRow was renamed, removed, or reverted to '
            'void. Re-point this gate rather than deleting it -- callers '
            'must be able to tell whether a row was actually matched and '
            'removed (CR-01).',
      );
      expect(
        codeOnly.contains('void deleteLocalTrailRow('),
        isFalse,
        reason:
            'A void deleteLocalTrailRow cannot distinguish "matched and '
            'removed" from "no row owned by this account carried this '
            'localId" -- exactly the CR-01 failure: the caller deleted '
            'photo files and reported success for an operation that '
            'deleted nothing it was allowed to delete.',
      );
    });

    test('body contains both return false; and return true;', () {
      final codeOnly = readCodeOnly('lib/store/local_trail_store.dart');

      final start = codeOnly.indexOf('bool deleteLocalTrailRow(');
      expect(start, isNot(-1));
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      expect(
        body.contains('return false;'),
        isTrue,
        reason:
            'The entity == null (no-match) path must report false so the '
            'caller can refuse to delete files or report deleted (D-07).',
      );
      expect(
        body.contains('return true;'),
        isTrue,
        reason:
            'The matched-and-removed path must report true so the caller '
            'can proceed with the rest of the delete flow.',
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
}
