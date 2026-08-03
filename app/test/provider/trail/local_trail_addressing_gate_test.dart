import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for T-36-11-01/T-36-11-02: a real ObjectBox [Store]
/// cannot be opened in `flutter test` (`libobjectbox.dylib` fails to load),
/// so the query itself is not behaviourally testable here -- same rationale
/// as `test/routes/trail_create_screen_local_save_gate_test.dart` and
/// `test/components/trail/trail_dropdown_delete_gate_test.dart`.
void main() {
  final libDir = Directory('lib');

  test(
    'local_trail_store.dart declares readOwnLocalTrail, owner-scoped',
    () {
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to "app/" (e.g. "cd app && flutter test").',
      );

      final source = File(
        'lib/store/local_trail_store.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      expect(
        codeOnly.contains('Trail? readOwnLocalTrail('),
        isTrue,
        reason:
            'readOwnLocalTrail was renamed or removed. Re-point this gate '
            'rather than deleting it -- the invariant still matters.',
      );

      final start = codeOnly.indexOf('Trail? readOwnLocalTrail(');
      final end = codeOnly.indexOf('\n}\n', start);
      expect(end, isNot(-1));
      final body = codeOnly.substring(start, end);

      expect(
        body.contains('TrailEntity_.owner.equals(accountId)'),
        isTrue,
        reason:
            'This is the security invariant: without the owner clause, '
            'account B could render account A\'s not-yet-uploaded trail '
            '(D-13). The gate fails the moment someone "simplifies" the '
            'scoping away.',
      );
    },
  );

  test(
    'local_trail_provider.dart validates the local id and scopes by '
    'account, never by savedByUserIds',
    () {
      final source = File(
        'lib/provider/trail/local_trail_provider.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      expect(
        codeOnly.contains('isLocalId(localId)'),
        isTrue,
        reason:
            'localTrailProvider must fail closed on a non-local id -- a '
            'server-issued id reaching this provider is a caller bug '
            '(T-36-11-02).',
      );
      expect(
        codeOnly.contains('currentAccountId('),
        isTrue,
        reason:
            'localTrailProvider must resolve the signed-in account fresh '
            'and delegate account scoping to readOwnLocalTrail (T-36-11-01).',
      );
      expect(
        codeOnly.contains('savedByUserIds'),
        isFalse,
        reason:
            'D-10: ownership ("I made this") and library membership ("I '
            'downloaded this") must stay strictly separate. A local '
            'capture never sets savedByUserIds.',
      );
    },
  );
}
