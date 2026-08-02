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
  test('downloadTrail carries owner/localId/syncState/syncAttempts/'
      'syncNextAttemptAt/localPhotos forward from the existing row inside its '
      'write transaction', () {
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
      expect(body.contains(field), isTrue, reason: lossDescriptions[field]);
    }
  });

  // Same invariant, same failure mode, second site. `updateLocalTrail` also
  // rebuilds the row with `TrailEntity.fromModel` and puts it back over the
  // existing one, so every local-bookkeeping column it does not name is
  // erased on each re-save. `photos` was the one it missed.
  test(
    'updateLocalTrail carries obxId/id/owner/localId/syncState/syncAttempts/'
    'syncNextAttemptAt/savedByUserIds/photos forward from the existing row',
    () {
      final source = File('lib/util/local_trail_store.dart').readAsStringSync();

      final fnStart = source.indexOf('LocalUpdateOutcome updateLocalTrail(');
      expect(
        fnStart,
        isNot(-1),
        reason:
            'updateLocalTrail was renamed or its signature changed. Re-point '
            'this gate rather than deleting it -- the invariant still matters.',
      );

      // '\n}\n', not '\n}': the parameter list itself ends with '\n}) {', so
      // the shorter anchor matches before the body has even started and every
      // assertion below passes vacuously on an empty slice.
      final fnEnd = source.indexOf('\n}\n', fnStart);
      expect(
        fnEnd,
        isNot(-1),
        reason: 'Could not find updateLocalTrail\'s end.',
      );
      expect(
        fnEnd,
        greaterThan(fnStart + 400),
        reason:
            'The sliced body is implausibly short -- the end anchor is '
            'matching inside the signature, which would make every '
            'carry-forward assertion below pass vacuously.',
      );

      final body = source.substring(fnStart, fnEnd);

      const lossDescriptions = {
        'existing.obxId':
            'MISSING obxId carry-forward. The put would insert a SECOND row '
            'instead of updating the existing one.',
        'existing.id':
            'MISSING id carry-forward. The row would lose its server id (or '
            'its local sentinel), orphaning it from the drain.',
        'existing.owner':
            'MISSING owner carry-forward. The trail would silently vanish '
            'from the offline own-trails list (REC-06).',
        'existing.localId':
            'MISSING localId carry-forward. The permanent local identity the '
            'drain and its photo directory key off would be stripped.',
        'existing.syncState':
            'MISSING syncState carry-forward. A re-save would reset the '
            'upload lifecycle, losing a parked failure or in-progress marker.',
        'existing.syncAttempts':
            'MISSING syncAttempts carry-forward. A re-save would reset the '
            'retry-attempt counter, undermining the D-07 backoff budget.',
        'existing.syncNextAttemptAt':
            'MISSING syncNextAttemptAt carry-forward. A re-save would clear a '
            'scheduled backoff retry time.',
        'existing.savedByUserIds':
            'MISSING savedByUserIds carry-forward. Every account holding this '
            'trail in its offline library would lose it.',
        'existing.photos':
            'MISSING photos carry-forward. TrailEntity.fromModel always '
            'leaves photos at [], so a re-save erases the server-side photo '
            'filenames from the row -- leaving a card with no thumbnail and '
            'no way to read the photos back onto this device.',
      };

      for (final entry in lossDescriptions.entries) {
        expect(body.contains(entry.key), isTrue, reason: entry.value);
      }
    },
  );
}
