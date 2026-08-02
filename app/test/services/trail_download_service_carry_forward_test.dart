import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for a data-loss regression.
///
/// `TrailEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)` and
/// `downloadTrail` puts a FRESH row built by `fromModel`. Putting it blind
/// would wipe six local-bookkeeping fields the moment a trail that was
/// captured on this device (and possibly already uploaded, or still
/// mid-drain) is later re-downloaded: `owner` (silently vanishing from the
/// offline own-trails list, REC-06, with no error anywhere), `localId` (the
/// permanent local identity the drain and its photo directory key off),
/// `syncState`/`syncAttempts`/`syncNextAttemptAt` (a failed row's resume
/// state), and `localPhotos` (app-owned unsynced photo copies).
///
/// Tested at source level rather than behaviourally on purpose: driving
/// `downloadTrail` needs a live ObjectBox store, a Dio client and a real
/// filesystem, which is a lot of fragile scaffolding to protect a
/// six-field carry-forward invariant. This mirrors the gate in
/// `test/components/trail/trail_dropdown_delete_gate_test.dart`.
void main() {
  test(
    'downloadTrail carries owner/localId/syncState/syncAttempts/'
    'syncNextAttemptAt/localPhotos forward from the existing row inside its '
    'write transaction',
    () {
      final source = File(
        'lib/services/trail_download_service.dart',
      ).readAsStringSync();

      final txStart = source.indexOf('runInTransaction(TxMode.write');
      expect(
        txStart,
        isNot(-1),
        reason:
            'The write transaction in downloadTrail was renamed or removed. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final txEnd = source.indexOf('\n    });', txStart);
      expect(txEnd, isNot(-1), reason: 'Could not find the transaction end.');

      final body = source.substring(txStart, txEnd);

      const carriedFields = [
        'existing?.owner',
        'existing?.localId',
        'existing?.syncState',
        'existing?.syncAttempts',
        'existing?.syncNextAttemptAt',
        'existing?.localPhotos',
      ];

      const lossDescriptions = {
        'existing?.owner':
            'MISSING owner carry-forward. A re-download of a trail captured '
            'on this device would wipe its owner, and the trail would '
            'silently vanish from the offline own-trails list (REC-06) with '
            'no error anywhere.',
        'existing?.localId':
            'MISSING localId carry-forward. A re-download would strip the '
            'permanent local identity the drain and its photo directory key '
            'off, orphaning any in-flight upload bookkeeping.',
        'existing?.syncState':
            'MISSING syncState carry-forward. A re-download mid-drain would '
            'reset a trail\'s upload lifecycle, losing a parked failure or '
            'in-progress upload marker.',
        'existing?.syncAttempts':
            'MISSING syncAttempts carry-forward. A re-download mid-drain '
            'would reset the retry-attempt counter, undermining the D-07 '
            'backoff budget.',
        'existing?.syncNextAttemptAt':
            'MISSING syncNextAttemptAt carry-forward. A re-download '
            'mid-drain would clear a scheduled backoff retry time, causing '
            'a premature or missed retry.',
        'existing?.localPhotos':
            'MISSING localPhotos carry-forward. A re-download would strip '
            'the app-owned unsynced photo copies a not-yet-uploaded trail '
            'depends on.',
      };

      for (final field in carriedFields) {
        expect(
          body.contains(field),
          isTrue,
          reason: lossDescriptions[field],
        );
      }
    },
  );
}
