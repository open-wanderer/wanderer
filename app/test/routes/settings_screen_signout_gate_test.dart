import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level gate for: a hiker who signs out while trails are still
/// waiting to upload must be warned first, naming the count.
///
/// Tested at source level rather than behaviourally, on purpose: driving the
/// real Settings sign-out button needs auth, a router, and a live ObjectBox
/// store behind `countUnsyncedTrails` -- the same rationale
/// `trail_dropdown_delete_gate_test.dart` records for its own gate.
///
/// Two call sites exist. `settings_screen.dart`'s is a sign-out the hiker
/// chose and MUST go through the guard -- otherwise a hiker signs out and
/// their pending trails silently stop uploading with no warning.
/// `settings_account_screen.dart`'s follows account deletion, is not a
/// choice the hiker made, and is a deliberate, documented exemption.
void main() {
  test(
    'settings_screen.dart routes its logout button through '
    'confirmSignOutWithUnsyncedTrails before calling logout()',
    () {
      final source = File(
        'lib/routes/settings_screen.dart',
      ).readAsStringSync();

      final onPressedStart = source.indexOf('onPressed: () async {');
      expect(
        onPressedStart,
        isNot(-1),
        reason:
            'The logout button\'s onPressed is no longer an async callback. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final onPressedEnd = source.indexOf('},', onPressedStart);
      expect(
        onPressedEnd,
        isNot(-1),
        reason: 'Could not find the end of the onPressed body.',
      );

      final body = source.substring(onPressedStart, onPressedEnd);

      final guardIndex = body.indexOf('confirmSignOutWithUnsyncedTrails');
      expect(
        guardIndex,
        isNot(-1),
        reason:
            'MISSING confirmSignOutWithUnsyncedTrails in the Settings '
            'sign-out button\'s onPressed. A hiker who taps Logout with '
            'trails still pending would sign out and their trails would '
            'silently stop uploading with no warning.',
      );

      final logoutIndex = body.indexOf('authProvider.notifier).logout()');
      expect(logoutIndex, isNot(-1));

      expect(
        guardIndex < logoutIndex,
        isTrue,
        reason:
            'confirmSignOutWithUnsyncedTrails must appear BEFORE logout() '
            'is called, so a cancelled/dismissed dialog actually prevents '
            'the sign-out.',
      );
    },
  );

  test(
    'settings_account_screen.dart documents its logout() call as a '
    'deliberate exemption from the sign-out guard',
    () {
      final source = File(
        'lib/routes/settings_account_screen.dart',
      ).readAsStringSync();

      final logoutIndex = source.indexOf('authProvider.notifier).logout()');
      expect(
        logoutIndex,
        isNot(-1),
        reason:
            'The post-account-deletion logout() call moved or was removed. '
            'Re-point this gate rather than deleting it.',
      );

      // The exemption comment must sit immediately above the call, not just
      // exist somewhere in the file, so a future reader cannot confuse this
      // site with the one that IS guarded.
      final precedingSource = source.substring(0, logoutIndex);
      final commentStart = precedingSource.lastIndexOf(
        '// Deliberately NOT routed through the shared unsynced-trails '
        'sign-out',
      );
      expect(
        commentStart,
        isNot(-1),
        reason:
            'settings_account_screen.dart\'s logout() call is missing its '
            'deliberate-exemption comment. Without it, a future reader may '
            '"fix" the missing guard here, but this sign-out follows '
            'account deletion (server-side, unrecoverable) -- any pending '
            'trails can never upload regardless, so the warning would be '
            'pointless.',
      );

      // And it really is the closest preceding comment -- no actual CALL to
      // the guard (as opposed to a mention of its name in this very
      // exemption comment) sits between it and the logout().
      expect(
        source.substring(commentStart, logoutIndex).contains(
          'confirmSignOutWithUnsyncedTrails(',
        ),
        isFalse,
        reason:
            'settings_account_screen.dart must NOT call '
            'confirmSignOutWithUnsyncedTrails -- this sign-out is not a '
            'choice the hiker made.',
      );
    },
  );
}
