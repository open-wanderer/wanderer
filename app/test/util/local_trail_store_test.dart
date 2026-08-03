import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/local_trail_store.dart';

// ---------------------------------------------------------------------------
// Pure-decision tests for local_trail_store.dart. No Store construction --
// Phase 31 established there is no ObjectBox test harness for plain
// `flutter test`, so every Store-touching function here is covered only at
// source level (see the plan's grep-based acceptance criteria) or verified
// on-device.
// ---------------------------------------------------------------------------

void main() {
  group('resolveLocalSaveMode', () {
    test('returns networkUpdate for a synced trail with a server id', () {
      final trail = Trail.empty().copyWith(
        id: 'server-id-123',
        syncState: TrailSyncState.synced,
      );

      expect(resolveLocalSaveMode(trail), LocalSaveMode.networkUpdate);
    });

    test('returns createLocal for a trail with id: "" and localId: null', () {
      final trail = Trail.empty().copyWith(id: '', localId: null);

      expect(resolveLocalSaveMode(trail), LocalSaveMode.createLocal);
    });

    test('returns updateLocal for a trail with id: "", a non-null localId and '
        'syncState: pending', () {
      final trail = Trail.empty().copyWith(
        id: '',
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );

      expect(resolveLocalSaveMode(trail), LocalSaveMode.updateLocal);
    });

    test('returns updateLocal -- NOT networkUpdate -- for a trail with a '
        'non-empty server id whose syncState is still uploading (the '
        'mid-drain resume case)', () {
      final trail = Trail.empty().copyWith(
        id: 'server-id-456',
        localId: 'local-1-0',
        syncState: TrailSyncState.uploading,
      );

      expect(resolveLocalSaveMode(trail), LocalSaveMode.updateLocal);
    });
  });

  group(
    'resolveLocalSaveMode routing input: snapshot vs persisted row (CR-04)',
    () {
      // TrailCreateScreen used to route on its own `trail` field, captured at
      // the end of _finishLocalSave while syncState was still `pending`. The
      // screen never watches the drain, so once the upload finished moments
      // later the snapshot was stale. These two cases are the SAME trail at the
      // same instant, seen through the two different inputs -- they disagree,
      // and the disagreement is the whole bug: routing on the stale one sent a
      // post-upload edit into updateLocal, which carried `synced` forward, and
      // selectDrainCandidates skips synced rows. The edit never left the device,
      // under a green "trail saved successfully" toast.
      final staleScreenSnapshot = Trail.empty().copyWith(
        id: '',
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );

      final persistedRowAfterUpload = Trail.empty().copyWith(
        id: 'server-id-789',
        localId: 'local-1-0',
        syncState: TrailSyncState.synced,
      );

      test(
        'the stale screen snapshot routes to updateLocal -- a local-only write',
        () {
          expect(
            resolveLocalSaveMode(staleScreenSnapshot),
            LocalSaveMode.updateLocal,
          );
        },
      );

      test(
        'the persisted row routes to networkUpdate, the only target that can '
        'carry a post-upload edit anywhere',
        () {
          expect(
            resolveLocalSaveMode(persistedRowAfterUpload),
            LocalSaveMode.networkUpdate,
          );
        },
      );

      test(
        'the two inputs disagree, so which one is passed is load-bearing',
        () {
          expect(
            resolveLocalSaveMode(staleScreenSnapshot),
            isNot(resolveLocalSaveMode(persistedRowAfterUpload)),
          );
        },
      );
    },
  );

  group('shouldDeleteUploadedRow', () {
    test('returns true for an empty savedByUserIds list -- the '
        'overwhelmingly common case', () {
      expect(shouldDeleteUploadedRow(const []), isTrue);
    });

    test('returns false when one account holds the trail in its offline '
        'library', () {
      expect(shouldDeleteUploadedRow(const ['user-a']), isFalse);
    });

    test('returns false when multiple accounts hold the trail in their '
        'offline library', () {
      expect(shouldDeleteUploadedRow(const ['user-a', 'user-b']), isFalse);
    });
  });

  group('resolveLocalSaveModeForRow', () {
    test(
      'returns networkUpdate when persistedLocalId is set but persisted is '
      'null -- the row was retired because the upload finished',
      () {
        final screenTrail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );

        expect(
          resolveLocalSaveModeForRow(
            screenTrail: screenTrail,
            persistedLocalId: 'local-1-0',
            persisted: null,
          ),
          LocalSaveMode.networkUpdate,
        );
      },
    );

    test(
      'delegates to resolveLocalSaveMode on screenTrail when '
      'persistedLocalId is null -- the never-saved-anywhere case',
      () {
        final screenTrail = Trail.empty().copyWith(id: '', localId: null);

        expect(
          resolveLocalSaveModeForRow(
            screenTrail: screenTrail,
            persistedLocalId: null,
            persisted: null,
          ),
          LocalSaveMode.createLocal,
        );
      },
    );

    test(
      'returns updateLocal for an ordinary offline re-save with a persisted '
      'pending row',
      () {
        final screenTrail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );
        final persisted = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );

        expect(
          resolveLocalSaveModeForRow(
            screenTrail: screenTrail,
            persistedLocalId: 'local-1-0',
            persisted: persisted,
          ),
          LocalSaveMode.updateLocal,
        );
      },
    );

    test(
      'returns networkUpdate when the persisted row already carries a '
      'server id and is synced -- delegation still honours '
      'resolveLocalSaveMode\'s own rule',
      () {
        final screenTrail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );
        final persisted = Trail.empty().copyWith(
          id: 'server-1',
          localId: 'local-1-0',
          syncState: TrailSyncState.synced,
        );

        expect(
          resolveLocalSaveModeForRow(
            screenTrail: screenTrail,
            persistedLocalId: 'local-1-0',
            persisted: persisted,
          ),
          LocalSaveMode.networkUpdate,
        );
      },
    );

    test(
      'disagrees with resolveLocalSaveMode alone when persisted is null -- '
      'proving the substitution in the create screen is load-bearing, not '
      'cosmetic',
      () {
        final screenTrail = Trail.empty().copyWith(
          id: '',
          localId: 'local-1-0',
          syncState: TrailSyncState.pending,
        );

        expect(
          resolveLocalSaveMode(screenTrail),
          LocalSaveMode.updateLocal,
        );
        expect(
          resolveLocalSaveModeForRow(
            screenTrail: screenTrail,
            persistedLocalId: 'local-1-0',
            persisted: null,
          ),
          LocalSaveMode.networkUpdate,
        );
      },
    );
  });

  group('isDrainDue', () {
    TrailEntity buildEntity({
      required TrailSyncState syncState,
      DateTime? syncNextAttemptAt,
    }) {
      final entity = TrailEntity(
        id: 'local-1-0',
        name: 'Test Trail',
        created: DateTime.now(),
        updated: DateTime.now(),
      );
      entity.syncState = syncState;
      entity.syncNextAttemptAt = syncNextAttemptAt;
      return entity;
    }

    test('is true for a pending row with a null syncNextAttemptAt', () {
      final entity = buildEntity(syncState: TrailSyncState.pending);

      expect(isDrainDue(entity, DateTime.now()), isTrue);
    });

    test('is false for a pending row whose syncNextAttemptAt is in the '
        'future, and true once now passes it', () {
      final now = DateTime.now();
      final entity = buildEntity(
        syncState: TrailSyncState.pending,
        syncNextAttemptAt: now.add(const Duration(minutes: 5)),
      );

      expect(isDrainDue(entity, now), isFalse);
      expect(isDrainDue(entity, now.add(const Duration(minutes: 10))), isTrue);
    });

    test('is false for a failed row even with a null syncNextAttemptAt', () {
      final entity = buildEntity(syncState: TrailSyncState.failed);

      expect(isDrainDue(entity, DateTime.now()), isFalse);
    });

    test('is false for a synced row', () {
      final entity = buildEntity(syncState: TrailSyncState.synced);

      expect(isDrainDue(entity, DateTime.now()), isFalse);
    });
  });

  group('resolveDrainFailureOutcome', () {
    test('parks the row as failed with no scheduled retry once the '
        'incremented attempt count reaches maxAttempts', () {
      final now = DateTime.now();

      final outcome = resolveDrainFailureOutcome(
        currentAttempts: 2,
        now: now,
        maxAttempts: 3,
        backoff: (attempts) => Duration(minutes: attempts),
      );

      expect(outcome.syncState, TrailSyncState.failed);
      expect(outcome.syncAttempts, 3);
      expect(outcome.syncNextAttemptAt, isNull);
    });

    test('stays pending with a backoff-scheduled retry below maxAttempts', () {
      final now = DateTime.now();

      final outcome = resolveDrainFailureOutcome(
        currentAttempts: 0,
        now: now,
        maxAttempts: 3,
        backoff: (attempts) => Duration(minutes: attempts * 5),
      );

      expect(outcome.syncState, TrailSyncState.pending);
      expect(outcome.syncAttempts, 1);
      expect(outcome.syncNextAttemptAt, now.add(const Duration(minutes: 5)));
    });
  });
}
