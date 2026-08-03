import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// UAT note (36-13): the reachability half of what this file appears to
/// assert -- that the delete/download gating below is actually live in the
/// running app -- is now covered BEHAVIOURALLY by
/// `trail_dropdown_menu_test.dart`, which opens the real `PopupMenuButton`
/// and reads the real rendered items. This file's remaining scope is branch
/// ORDER only: the source-text assertions below pin an invariant no widget
/// test can observe (that `_deleteTrail`'s unsynced branch runs, and
/// returns, BEFORE its `isLocal` un-download branch), which stayed green for
/// the whole phase even while `TrailDropdown` itself was unreachable for an
/// unsynced trail.
///
/// Source-level guard for a data-loss regression.
///
/// `_deleteTrail` handles THREE different actions behind one menu item,
/// ordered unsynced -> downloaded -> server:
/// - UNSYNCED (never reached the server): `TrailSync.deleteUnsynced` removes
///   the only copy that exists, on the device.
/// - DOWNLOADED (a LOCAL trail with a real server id) means "remove the
///   download" — and nothing else.
/// - Otherwise it means a real `DELETE /trail/{id}`.
///
/// The downloaded branch originally had no `return`, so it fell through into
/// the server delete: the only un-download gesture in the app (the download
/// menu item is inert once downloaded) also destroyed the trail on the
/// server. It was reachable while online, because `TrailNotifier.build()`
/// falls back to the ObjectBox cache on ANY fetch exception and a cached
/// model is stamped `isLocal: true` — one timeout on your own downloaded
/// trail was enough.
///
/// After 36-08, `trail.isLocal` is ALSO true for an unsynced trail
/// (`TrailEntity.toModel()` hardcodes it), so the unsynced branch must be
/// checked, and must return, BEFORE the downloaded branch — otherwise an
/// unsynced trail routes into `trailLibraryProvider.deleteTrail('')` with an
/// empty id and silently no-ops, leaving the hiker's only copy on the device
/// with no feedback.
///
/// Tested at source level rather than behaviourally on purpose: driving
/// `TrailDropdown` needs auth, router, download-state, search, profile-trails
/// AND a live ObjectBox store behind `trailLibraryProvider`, which is a lot of
/// fragile scaffolding to protect a one-token invariant. This mirrors the
/// PORT-03 gate in `test/util/trail_import_util_test.dart`.
void main() {
  test(
    'trail_dropdown._deleteTrail returns after un-downloading a local trail, '
    'so it never falls through to the server DELETE',
    () {
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

      final branchStart = source.indexOf('if (trail.isLocal) {', methodStart);
      expect(
        branchStart,
        isNot(-1),
        reason:
            'The local-trail branch is gone. If un-download moved elsewhere, '
            'move this gate with it.',
      );

      final branchEnd = source.indexOf('\n    }', branchStart);
      expect(branchEnd, isNot(-1), reason: 'Could not find the branch end.');

      final branch = source.substring(branchStart, branchEnd);

      expect(
        branch.contains('deleteTrail(trail.id)'),
        isTrue,
        reason: 'The branch no longer un-downloads; this gate is stale.',
      );
      expect(
        branch.contains('return;'),
        isTrue,
        reason:
            'MISSING `return` in _deleteTrail\'s local-trail branch. Removing '
            'a download now also issues DELETE /trail/{id} and destroys the '
            'user\'s trail on the server.',
      );

      // And the server delete really is downstream of that branch — if it
      // moved above, the return would no longer protect anything.
      final serverDelete = source.indexOf(
        'trailSaveProvider.notifier).deleteTrail(trail)',
        methodStart,
      );
      expect(serverDelete, isNot(-1));
      expect(
        serverDelete > branchEnd,
        isTrue,
        reason: 'The server DELETE is no longer guarded by the early return.',
      );
    },
  );

  test('trail_dropdown._deleteTrail checks the unsynced branch BEFORE the '
      'isLocal (un-download) branch, and returns from it', () {
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
    final localIdx = source.indexOf('if (trail.isLocal) {', methodStart);

    expect(
      unsyncedIdx,
      isNot(-1),
      reason:
          'The unsynced branch is gone from _deleteTrail. An unsynced '
          'trail also reports isLocal == true, so without its own branch '
          'checked first it silently routes into '
          "trailLibraryProvider.deleteTrail('') and no-ops, destroying "
          "the hiker's only copy with no feedback.",
    );
    expect(
      unsyncedIdx < localIdx,
      isTrue,
      reason:
          'The unsynced branch must be checked BEFORE the isLocal '
          '(un-download) branch -- an unsynced trail is also isLocal, so '
          'the wrong order silently no-ops the delete on an empty id.',
    );

    final unsyncedBranchEnd = source.indexOf('\n    }', unsyncedIdx);
    expect(unsyncedBranchEnd, isNot(-1));
    final unsyncedBranch = source.substring(unsyncedIdx, unsyncedBranchEnd);
    expect(
      unsyncedBranch.contains('return;'),
      isTrue,
      reason:
          'MISSING `return` in the unsynced branch. Without it, an '
          'unsynced delete falls through into the un-download / server '
          'delete branches below.',
    );
  });

  test('trail_dropdown hides the download menu item for an unsynced trail '
      '(if (!isUnsynced) collection-if)', () {
    final source = File(
      'lib/components/trail/trail_dropdown.dart',
    ).readAsStringSync();

    final guardIdx = source.indexOf('if (!isUnsynced) ...[');
    expect(
      guardIdx,
      isNot(-1),
      reason:
          'The download menu item is no longer gated behind '
          "if (!isUnsynced) -- offering download for an unsynced trail "
          "issues a server fetch with an empty trail id (D-17).",
    );

    final downloadIdx = source.indexOf('value: TrailAction.download', guardIdx);
    expect(
      downloadIdx,
      isNot(-1),
      reason: 'Could not find the download PopupMenuItem after the guard.',
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
}
