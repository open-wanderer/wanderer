import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for CR-03's UI half (38.1 plan 05): a widget test is
/// deliberately not used here. Mounting `LibraryScreen` requires
/// `trailLibraryProvider`, `routerProvider` and `trailFilterProvider`
/// overrides plus a live `Store` behind the library provider, and there is
/// no ObjectBox harness under plain `flutter test` (see
/// `test/store/local_trail_store_test.dart`'s own header for the same
/// constraint). The behavioural coverage for this predicate -- that the
/// action is actually hidden/shown correctly -- lives in
/// `trail_dropdown_menu_test.dart`, which mounts a real widget tree.
///
/// The invariant this file protects: `_showContextMenu`'s Remove `ListTile`
/// must be guarded by `ownLiveCaptureProvider` -- the SAME predicate
/// `trail_dropdown.dart` reads (D-12: one predicate, two surfaces) -- and
/// that guard must precede the call that reaches
/// `trailLibraryProvider.notifier.deleteTrail`. Every row this screen
/// renders comes from `trailLibraryProvider`, which filters only on
/// `savedByUserIds`, so without this guard the sheet fires unconditionally
/// for a row that could still be the hiker's own pending recording, while
/// `_confirmRemoveDownload`'s copy (`remove_download_confirm_body`) promises
/// "the trail itself is not deleted".
void main() {
  test(
    'library_screen._showContextMenu guards the Remove ListTile with '
    'ownLiveCaptureProvider, ordered before _confirmRemoveDownload',
    () {
      final libDir = Directory('lib');
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to "app/" (e.g. "cd app && flutter test").',
      );

      final source = File(
        'lib/routes/library_screen.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      final methodStart = codeOnly.indexOf(
        'void _showContextMenu(BuildContext context, Trail trail, router) {',
      );
      expect(
        methodStart,
        isNot(-1),
        reason:
            '_showContextMenu was renamed or its signature changed -- '
            're-point this gate rather than deleting it, the invariant '
            'still matters.',
      );

      final methodEnd = codeOnly.indexOf('\n  }', methodStart);
      expect(
        methodEnd,
        isNot(-1),
        reason: 'Could not find the end of _showContextMenu\'s body.',
      );
      final body = codeOnly.substring(methodStart, methodEnd);

      expect(
        body.contains('ownLiveCaptureProvider('),
        isTrue,
        reason:
            '_showContextMenu must consult ownLiveCaptureProvider -- the '
            'same predicate trail_dropdown.dart reads (D-12: exactly one '
            'predicate serving both surfaces), not a second implementation.',
      );

      final guardIdx = body.indexOf('if (!isOwnLiveCapture)');
      final confirmIdx = body.indexOf('_confirmRemoveDownload(');
      expect(
        guardIdx,
        isNot(-1),
        reason:
            'The Remove ListTile must be wrapped in if (!isOwnLiveCapture) '
            '-- D-13: hidden entirely (not disabled) when this account owns '
            'the row as a live, not-yet-uploaded capture.',
      );
      expect(
        confirmIdx,
        isNot(-1),
        reason:
            'Could not find the _confirmRemoveDownload call the guard '
            'exists to gate.',
      );
      expect(
        guardIdx < confirmIdx,
        isTrue,
        reason:
            'if (!isOwnLiveCapture) must precede the action it gates -- a '
            'guard checked after _confirmRemoveDownload is reachable is no '
            'guard at all.',
      );

      expect(
        body.contains('isUnsyncedState('),
        isFalse,
        reason:
            'The guard must not be re-derived from isUnsyncedState read '
            'off the shared model\'s syncState -- D-02/D-12 require it '
            'come from the owner-scoped ownLiveCaptureProvider only.',
      );

      expect(
        source.contains('remove_download_confirm_body'),
        isTrue,
        reason:
            '_confirmRemoveDownload must keep promising the trail itself '
            'is not deleted -- this guard is what makes that promise true.',
      );
      expect(
        source.contains('delete_trail_confirm'),
        isFalse,
        reason:
            'library_screen.dart must never share confirm copy with the '
            'server-delete flow -- un-downloading is genuinely undoable, '
            'unlike a server delete.',
      );
    },
  );
}
