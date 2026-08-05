import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// UAT note (36-13): the reachability half of what this file appears to
/// assert -- that the delete/download gating below is actually live in the
/// running app -- is now covered BEHAVIOURALLY by
/// `trail_dropdown_menu_test.dart`, which opens the real `PopupMenuButton`
/// and reads the real rendered items. This file's remaining scope is branch
/// ORDER only: the source-text assertions below pin an invariant no widget
/// test can observe (that `_deleteTrail`'s unsynced branch runs, and
/// returns, before anything else), which stayed green for the whole phase
/// even while `TrailDropdown` itself was unreachable for an unsynced trail.
///
/// Source-level guard for a data-loss regression.
///
/// 38-05 (D-01): `_deleteTrail` used to handle THREE different actions
/// behind one menu item, ordered unsynced -> downloaded -> server, with the
/// "downloaded" (un-download) branch gated on `trail.isLocal` -- a
/// provenance flag `TrailEntity.toModel()` hardcodes `true` for every cached
/// row, and that `TrailNotifier.build()`'s any-exception cache fallback
/// could attach to a server-authored trail after nothing more than a
/// timeout, arming a server DELETE with no ownership check at all. That
/// branch is now gone: un-downloading is its own labelled menu item
/// (`TrailAction.removeDownload`) with its own confirm
/// (`_confirmRemoveDownload`), which calls only the library-membership
/// provider and never touches `_deleteTrail`.
///
/// `_deleteTrail` now handles TWO actions:
/// - UNSYNCED (never reached the server): `TrailSync.deleteUnsynced` removes
///   the only copy that exists, on the device.
/// - Otherwise: falls straight through to `_deleteOnServer`, a real
///   `DELETE /trail/{id}`, gated on authorship by `_allowDelete`.
///
/// The unsynced branch must still be checked, and must still return, before
/// the fall-through -- an unsynced trail's copy is device-only, and reaching
/// `_deleteOnServer` for it would be meaningless.
///
/// Tested at source level rather than behaviourally on purpose: driving
/// `TrailDropdown` needs auth, router, download-state, search, profile-trails
/// AND a live ObjectBox store behind `trailLibraryProvider`, which is a lot of
/// fragile scaffolding to protect a one-token invariant. This mirrors the
/// PORT-03 gate in `test/util/trail_import_util_test.dart`.
void main() {
  test('trail_dropdown._deleteTrail falls straight through to '
      '_deleteOnServer after the unsynced branch -- no un-download branch '
      'remains in between', () {
    final source = File(
      'lib/components/trail/trail_dropdown.dart',
    ).readAsStringSync();

    final methodStart = source.indexOf(
      'Future<void> _deleteTrail(BuildContext context, Trail trail) async {',
    );
    expect(
      methodStart,
      isNot(-1),
      reason:
          '_deleteTrail was renamed or its signature changed. Re-point this '
          'gate rather than deleting it — the invariant still matters.',
    );

    final deleteTrailBodyEnd = source.indexOf('\n  }', methodStart);
    expect(
      deleteTrailBodyEnd,
      isNot(-1),
      reason: 'Could not find the end of _deleteTrail\'s body.',
    );
    final deleteTrailBody = source.substring(methodStart, deleteTrailBodyEnd);

    // D-01: nothing in _deleteTrail may branch on a cached-provenance flag
    // any more, and nothing in it may reach the library-membership provider
    // -- un-downloading is a separate menu item with a separate handler now.
    expect(
      deleteTrailBody.contains('isLocal'),
      isFalse,
      reason:
          'A cached-provenance branch reappeared in _deleteTrail. '
          'Destructive-action availability must derive from library '
          'membership and authorship only (D-01).',
    );
    expect(
      deleteTrailBody.contains('trailLibraryProvider'),
      isFalse,
      reason:
          '_deleteTrail must not touch trailLibraryProvider -- '
          'un-downloading is now TrailAction.removeDownload / '
          '_confirmRemoveDownload, a separate handler entirely.',
    );

    // The server DELETE (`trailSaveProvider.notifier).deleteTrail(trail)`)
    // was extracted into `_deleteOnServer` (36-15), so it no longer lives
    // inside `_deleteTrail` at all -- a plain forward-index-and-compare
    // check would either fail outright or silently degenerate into a
    // whole-file-order check once the call moved to a different method.
    // Assert BOTH halves explicitly: `_deleteTrail`'s own body contains no
    // such call, and `_deleteOnServer`'s body does -- so the extraction is
    // pinned, not merely tolerated, and the server delete cannot be
    // silently re-inlined into the fall-through path.
    expect(
      deleteTrailBody.contains(
        'trailSaveProvider.notifier).deleteTrail(trail)',
      ),
      isFalse,
      reason:
          'The server DELETE must not live inside _deleteTrail any more '
          '-- it was extracted into _deleteOnServer so the unsynced '
          "branch's null-localId fall-through (WR-08) can route to it "
          'directly. Its reappearance here means the extraction was '
          'reverted.',
    );
    expect(
      deleteTrailBody.contains('await _deleteOnServer(context, trail);'),
      isTrue,
      reason:
          '_deleteTrail must fall through directly to _deleteOnServer once '
          'the unsynced branch is exhausted -- no branch should sit '
          'between them any more.',
    );

    final onServerStart = source.indexOf(
      'Future<void> _deleteOnServer(BuildContext context, Trail trail) async {',
    );
    expect(
      onServerStart,
      isNot(-1),
      reason:
          '_deleteOnServer is gone or renamed. Re-point this gate rather '
          'than deleting it -- the invariant still matters.',
    );
    final onServerBodyEnd = source.indexOf('\n  }', onServerStart);
    expect(
      onServerBodyEnd,
      isNot(-1),
      reason: 'Could not find the end of _deleteOnServer\'s body.',
    );
    final onServerBody = source.substring(onServerStart, onServerBodyEnd);
    expect(
      onServerBody.contains('trailSaveProvider.notifier).deleteTrail(trail)'),
      isTrue,
      reason:
          '_deleteOnServer must contain the real server DELETE call -- '
          'otherwise nothing in the file issues it any more.',
    );
  });

  test('trail_dropdown._deleteTrail checks the unsynced branch first and '
      'returns from it', () {
    final source = File(
      'lib/components/trail/trail_dropdown.dart',
    ).readAsStringSync();

    final methodStart = source.indexOf(
      'Future<void> _deleteTrail(BuildContext context, Trail trail) async {',
    );
    expect(methodStart, isNot(-1));

    final unsyncedIdx = source.indexOf(
      'isUnsyncedState(trail.syncState)',
      methodStart,
    );

    expect(
      unsyncedIdx,
      isNot(-1),
      reason:
          'The unsynced branch is gone from _deleteTrail. An unsynced '
          "trail's copy lives only on the device, so without its own "
          'branch checked first it would fall through into a meaningless '
          'server DELETE with no feedback.',
    );

    final unsyncedBranchEnd = source.indexOf('\n    }', unsyncedIdx);
    expect(unsyncedBranchEnd, isNot(-1));
    final unsyncedBranch = source.substring(unsyncedIdx, unsyncedBranchEnd);
    expect(
      unsyncedBranch.contains('return;'),
      isTrue,
      reason:
          'MISSING `return` in the unsynced branch. Without it, an '
          'unsynced delete falls through into the server delete branch '
          'below.',
    );
  });

  test('trail_dropdown hides the download family behind showDownloadFamily, '
      'never a bare syncState read (if (showDownloadFamily) collection-if)', () {
    final source = File(
      'lib/components/trail/trail_dropdown.dart',
    ).readAsStringSync();

    final guardIdx = source.indexOf('if (showDownloadFamily) ...[');
    expect(
      guardIdx,
      isNot(-1),
      reason:
          'The download family is no longer gated behind '
          'if (showDownloadFamily) -- a destructive/safe-action gate in '
          'this file must never be decided by syncState read off the '
          'shared cache row (D-02/D-12). Offering Download for an unsynced '
          'trail also issues a server fetch with an empty trail id (D-17).',
    );

    final downloadIdx = source.indexOf('value: TrailAction.download', guardIdx);
    expect(
      downloadIdx,
      isNot(-1),
      reason: 'Could not find the download PopupMenuItem after the guard.',
    );

    expect(
      source.contains('ref.watch(ownLiveCaptureProvider('),
      isTrue,
      reason:
          'showDownloadFamily and the delete gate must both derive from '
          'ownLiveCaptureProvider (D-12: exactly one predicate serving '
          'both surfaces), not a re-derived local.',
    );

    final allowDeleteStart = source.indexOf(
      'bool _allowDelete(WidgetRef ref, {required bool isOwnLiveCapture})',
    );
    expect(
      allowDeleteStart,
      isNot(-1),
      reason:
          '_allowDelete must take isOwnLiveCapture -- the owner-scoped '
          'escape hatch is now resolved by the caller via '
          'ownLiveCaptureProvider, not by a bare syncState read inside '
          '_allowDelete itself.',
    );
    final allowDeleteBodyEnd = source.indexOf('\n  }', allowDeleteStart);
    expect(allowDeleteBodyEnd, isNot(-1));
    final allowDeleteBody = source.substring(
      allowDeleteStart,
      allowDeleteBodyEnd,
    );
    expect(
      allowDeleteBody.contains('isUnsyncedState('),
      isFalse,
      reason:
          '_allowDelete must never re-derive its escape hatch from '
          'isUnsyncedState(trail.syncState) -- that is exactly the CR-01 '
          'bug (a destructive gate decided by a field read off the shared '
          'cache row, with no ownership check at all).',
    );
  });

  test('_confirmDelete references delete_unsynced_trail_confirm for the '
      'unrecoverable unsynced-delete copy', () {
    final source = File(
      'lib/components/trail/trail_dropdown.dart',
    ).readAsStringSync();

    expect(
      source.contains('delete_unsynced_trail_confirm'),
      isTrue,
      reason:
          'An unsynced delete must use its own l10n key stating it cannot '
          'be undone -- the shared delete_trail_confirm string also '
          'doubles as the (genuinely reversible) un-download confirm in '
          'library_screen.dart.',
    );
  });

  test(
    '38.1 WR-05: the unsynced branch consults serverIdForRetired before '
    'toasting error_deleting_trail',
    () {
      final source = File(
        'lib/components/trail/trail_dropdown.dart',
      ).readAsStringSync();

      final failedIdx = source.indexOf('case UnsyncedDeleteResult.failed:');
      expect(
        failedIdx,
        isNot(-1),
        reason:
            'Could not find the failed case. Re-point this gate rather '
            'than deleting it -- the invariant still matters.',
      );

      final retiredIdx = source.indexOf('serverIdForRetired(', failedIdx);
      final toastIdx = source.indexOf('error_deleting_trail', failedIdx);
      expect(
        retiredIdx,
        isNot(-1),
        reason:
            'The failed branch must consult serverIdForRetired. Without it, '
            'a drain that retires the row between this screen\'s last read '
            'and the confirm tap leaves the hiker unable to delete the '
            'trail at all, under a toast blaming a failure that never '
            'happened (WR-05).',
      );
      expect(
        retiredIdx,
        lessThan(toastIdx),
        reason:
            'serverIdForRetired must be consulted BEFORE the error toast -- '
            'toasting first means the recovery route is unreachable.',
      );
    },
  );
}
