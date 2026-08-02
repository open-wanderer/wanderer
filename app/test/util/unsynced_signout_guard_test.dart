import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/unsynced_signout_guard.dart';

/// [confirmSignOutWithUnsyncedTrails]'s dialog half needs a live ObjectBox
/// `Store` (via `currentAccountId`/`countUnsyncedTrails`) and a `BuildContext`
/// -- it is covered by the source-level gate in
/// `test/routes/settings_screen_signout_gate_test.dart` instead. Only the
/// pure threshold decision is unit-tested here.
void main() {
  group('shouldWarnBeforeSignOut', () {
    test('false for zero pending trails', () {
      expect(shouldWarnBeforeSignOut(0), isFalse);
    });

    test('true for a single pending trail', () {
      expect(shouldWarnBeforeSignOut(1), isTrue);
    });

    test('true for a large pending count', () {
      expect(shouldWarnBeforeSignOut(500), isTrue);
    });
  });
}
