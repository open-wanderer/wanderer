import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for a data-loss regression.
///
/// `_deleteTrail` handles two different actions behind one menu item. For a
/// LOCAL trail it means "remove the download" — and nothing else. For a
/// server trail it means a real `DELETE /trail/{id}`.
///
/// The local branch originally had no `return`, so it fell through into the
/// server delete: the only un-download gesture in the app (the download menu
/// item is inert once downloaded) also destroyed the trail on the server. It
/// was reachable while online, because `TrailNotifier.build()` falls back to
/// the ObjectBox cache on ANY fetch exception and a cached model is stamped
/// `isLocal: true` — one timeout on your own downloaded trail was enough.
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
}
