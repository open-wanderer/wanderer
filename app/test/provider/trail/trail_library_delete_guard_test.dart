import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for: there is no ObjectBox test harness
/// for plain `flutter test` (see `test/store/local_trail_store_test.dart`'s
/// header), so `TrailLibraryNotifier.deleteTrail` -- which touches a real
/// `Store` -- is covered here at source level rather than behaviourally.
/// The behavioural half (hiding the action where it is meaningless)
/// is covered by the menu tests plan 05 adds.
///
/// The invariant this file protects: `deleteTrail` must never `box.remove`
/// a row that is still some account's live capture. On the cross-account
/// overlap row (`owner` set, `localId` set, `syncState` unsynced), an
/// unguarded removal permanently destroys the hiker's queued upload --
/// `selectDrainCandidates` has no other handle on it once the row is gone.
/// `deleteTrail` must still drop this account's `savedByUserIds` membership
/// on that same row, so "Remove download" is never a silent no-op.
void main() {
  test(
    'trail_library_provider.deleteTrail guards box.remove with '
    'isLiveCaptureRow, ordered before the removal',
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
        'lib/provider/trail/trail_library_provider.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      final methodStart = codeOnly.indexOf(
        'Future<void> deleteTrail(String id) async {',
      );
      expect(
        methodStart,
        isNot(-1),
        reason:
            'deleteTrail was renamed -- re-point this gate rather than '
            'deleting it, the invariant still matters.',
      );

      final methodEnd = codeOnly.indexOf('\n  }', methodStart);
      expect(
        methodEnd,
        isNot(-1),
        reason: 'Could not find the end of deleteTrail\'s body.',
      );
      final body = codeOnly.substring(methodStart, methodEnd);

      // Exactly one removal call must remain in the method, and it must be
      // inside the guarded branch.
      final removeCount = RegExp(
        r'box\.remove\(',
      ).allMatches(body).length;
      expect(
        removeCount,
        1,
        reason:
            'deleteTrail must contain exactly one box.remove call, inside '
            'the isLiveCaptureRow-guarded branch. A second call site would '
            'reintroduce an unscoped removal path.',
      );

      final liveCaptureIdx = body.indexOf('isLiveCaptureRow(');
      final removeIdx = body.indexOf('box.remove(');
      expect(
        liveCaptureIdx,
        isNot(-1),
        reason:
            'isLiveCaptureRow must be consulted inside deleteTrail -- '
            'On the overlap row, box.remove destroys the hiker\'s '
            'pending recording permanently, because selectDrainCandidates '
            'has no other handle on it.',
      );
      expect(
        liveCaptureIdx < removeIdx,
        isTrue,
        reason:
            'isLiveCaptureRow must be evaluated BEFORE box.remove -- '
            'On the overlap row, box.remove destroys the hiker\'s '
            'pending recording permanently, because selectDrainCandidates '
            'has no other handle on it. A guard checked after the removal '
            'is no guard at all.',
      );

      expect(
        body.contains('!isLiveCaptureRow(entity)'),
        isTrue,
        reason:
            'The guard must be a refusal (negated) to remove, not merely '
            'a log or an unrelated read of isLiveCaptureRow.',
      );

      expect(
        body.contains('entity.savedByUserIds = remaining;'),
        isTrue,
        reason:
            'Membership must still be dropped on the keep-the-row branch '
            '-- otherwise "Remove download" silently does nothing for a '
            'live-capture row.',
      );

      expect(
        body.contains('deleteUnsyncedPhotoDir'),
        isFalse,
        reason:
            'deleteTrail must never reach into unsynced capture storage. '
            'Owner-scoped deletion of that storage stays '
            'TrailSync.deleteUnsynced\'s job.',
      );

      final rowRemovedGuardIdx = body.indexOf('if (rowRemoved)');
      expect(
        rowRemovedGuardIdx,
        isNot(-1),
        reason:
            'The library/<id>/ directory delete must be gated on '
            'rowRemoved -- deleting the downloaded files while keeping the '
            'row would strand a live capture pointing at missing images.',
      );

      // The guard above pins that the directory delete is gated; these two
      // pin WHICH BRANCH sets the gate. Without them, swapping the two
      // `return` statements -- exactly the rename-and-invert performed
      // when `stillHeldByAnother` became `rowRemoved` -- leaves every other
      // assertion in this file green while `library/<id>/` is deleted on
      // the KEEP-the-row path: the downloaded photo files of a live capture
      // the guard just refused to remove, leaving the hiker's pending
      // recording pointing at missing images.
      expect(
        RegExp(
          r'box\.remove\(entity\.obxId\);\s*return true;',
        ).hasMatch(body),
        isTrue,
        reason:
            'The REMOVAL branch must return true. rowRemoved gates the '
            'library/<id>/ delete, so inverting it deletes the downloaded '
            'files on the keep-the-row path -- the precise outcome '
            'deleteTrail\'s doc comment says this guard exists to prevent.',
      );
      expect(
        RegExp(
          r'entity\.savedByUserIds = remaining;\s*box\.put\(entity\);\s*'
          r'return false;',
        ).hasMatch(body),
        isTrue,
        reason:
            'The KEEP-the-row branch must return false: its library/<id>/ '
            'files are still referenced by the row it just kept.',
      );
    },
  );
}
