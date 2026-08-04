import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/store/local_trail_store.dart';

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
    test('returns networkUpdate when persistedLocalId is set but persisted is '
        'null -- the row was retired because the upload finished', () {
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
    });

    test('delegates to resolveLocalSaveMode on screenTrail when '
        'persistedLocalId is null -- the never-saved-anywhere case', () {
      final screenTrail = Trail.empty().copyWith(id: '', localId: null);

      expect(
        resolveLocalSaveModeForRow(
          screenTrail: screenTrail,
          persistedLocalId: null,
          persisted: null,
        ),
        LocalSaveMode.createLocal,
      );
    });

    test('returns updateLocal for an ordinary offline re-save with a persisted '
        'pending row', () {
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
    });

    test('returns networkUpdate when the persisted row already carries a '
        'server id and is synced -- delegation still honours '
        'resolveLocalSaveMode\'s own rule', () {
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
    });

    test('disagrees with resolveLocalSaveMode alone when persisted is null -- '
        'proving the substitution in the create screen is load-bearing, not '
        'cosmetic', () {
      final screenTrail = Trail.empty().copyWith(
        id: '',
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );

      expect(resolveLocalSaveMode(screenTrail), LocalSaveMode.updateLocal);
      expect(
        resolveLocalSaveModeForRow(
          screenTrail: screenTrail,
          persistedLocalId: 'local-1-0',
          persisted: null,
        ),
        LocalSaveMode.networkUpdate,
      );
    });

    // CR-03: a persisted row that already carries a real server id, even
    // though its syncState has not (yet) reached `synced`, must route to
    // the network -- this is the exact `alreadyUploaded` window
    // updateLocalTrail refuses. Getting this wrong (routing it to
    // updateLocal) is what let a network edit reach the server while the
    // local row silently kept its pre-edit values.
    test('returns networkUpdate for a persisted row with a non-empty server '
        'id whose syncState is still pending (CR-03 window)', () {
      final screenTrail = Trail.empty().copyWith(
        id: '',
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );
      final persisted = Trail.empty().copyWith(
        id: 'server-1',
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );

      expect(
        resolveLocalSaveModeForRow(
          screenTrail: screenTrail,
          persistedLocalId: 'local-1-0',
          persisted: persisted,
        ),
        LocalSaveMode.networkUpdate,
      );
    });

    test('returns networkUpdate for a persisted row with a non-empty server '
        'id whose syncState is failed (CR-03 window, deterministic waypoint '
        'failure)', () {
      final screenTrail = Trail.empty().copyWith(
        id: '',
        localId: 'local-1-0',
        syncState: TrailSyncState.failed,
      );
      final persisted = Trail.empty().copyWith(
        id: 'server-1',
        localId: 'local-1-0',
        syncState: TrailSyncState.failed,
      );

      expect(
        resolveLocalSaveModeForRow(
          screenTrail: screenTrail,
          persistedLocalId: 'local-1-0',
          persisted: persisted,
        ),
        LocalSaveMode.networkUpdate,
      );
    });

    // Control case: an ordinary unsynced re-save (empty id, still pending)
    // must stay local. This fails if the CR-03 fix accidentally widens the
    // routing to also catch a trail that has never reached the server at
    // all (REC-01's offline-edit path).
    test('still returns updateLocal for a persisted row with an empty id '
        'and syncState pending -- an ordinary offline re-save', () {
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
    });

    test('still returns createLocal when persistedLocalId is null -- the '
        'first-save case', () {
      final screenTrail = Trail.empty().copyWith(id: '', localId: null);

      expect(
        resolveLocalSaveModeForRow(
          screenTrail: screenTrail,
          persistedLocalId: null,
          persisted: null,
        ),
        LocalSaveMode.createLocal,
      );
    });
  });

  group('trailHasServerId', () {
    test('is false for the blank id a still-local sentinel row reads as '
        '(TrailEntity.toModel() blanks it, D-06)', () {
      expect(trailHasServerId(''), isFalse);
    });

    test('is true for a real server id, regardless of syncState -- a '
        '`failed` row can carry one from a create that succeeded before a '
        'later waypoint upload failed (CR-03, CR-04)', () {
      expect(trailHasServerId('server-1'), isTrue);
    });
  });

  group('resolveNetworkSaveTarget', () {
    test('a real screen id wins even when a retired id is also present', () {
      expect(
        resolveNetworkSaveTarget(
          screenTrailId: 'server-1',
          retiredServerId: 'server-2',
        ),
        'server-1',
      );
    });

    test('a blank screen id falls back to the retired id', () {
      expect(
        resolveNetworkSaveTarget(
          screenTrailId: '',
          retiredServerId: 'server-2',
        ),
        'server-2',
      );
    });

    test('a blank screen id with a null retired id returns null', () {
      expect(
        resolveNetworkSaveTarget(screenTrailId: '', retiredServerId: null),
        isNull,
      );
    });

    test('a blank screen id with an empty-string retired id returns null', () {
      expect(
        resolveNetworkSaveTarget(screenTrailId: '', retiredServerId: ''),
        isNull,
      );
    });
  });

  group('resolveServerDeleteOutcome', () {
    test('404 -- the server copy is already gone -- proceeds with the '
        'local delete', () {
      expect(
        resolveServerDeleteOutcome(statusCode: 404, connectionFailed: false),
        ServerDeleteOutcome.proceedWithLocalDelete,
      );
    });

    test('null status + connectionFailed: true -- the request never '
        'reached a server -- aborts needing a connection', () {
      expect(
        resolveServerDeleteOutcome(statusCode: null, connectionFailed: true),
        ServerDeleteOutcome.abortNeedsConnection,
      );
    });

    test('401 aborts and reports', () {
      expect(
        resolveServerDeleteOutcome(statusCode: 401, connectionFailed: false),
        ServerDeleteOutcome.abortAndReport,
      );
    });

    test('403 aborts and reports', () {
      expect(
        resolveServerDeleteOutcome(statusCode: 403, connectionFailed: false),
        ServerDeleteOutcome.abortAndReport,
      );
    });

    test('500 aborts and reports', () {
      expect(
        resolveServerDeleteOutcome(statusCode: 500, connectionFailed: false),
        ServerDeleteOutcome.abortAndReport,
      );
    });

    test('null status + connectionFailed: false aborts and reports -- '
        'there is no "delete on this device only" escape hatch', () {
      expect(
        resolveServerDeleteOutcome(statusCode: null, connectionFailed: false),
        ServerDeleteOutcome.abortAndReport,
      );
    });
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

  group('hasKeylessPendingWaypoint', () {
    test('is false for an empty list', () {
      expect(hasKeylessPendingWaypoint(const []), isFalse);
    });

    test(
      'is false for a local-id waypoint with a non-null localKey',
      () {
        expect(
          hasKeylessPendingWaypoint(const [
            (id: 'local-1-0', localKey: 'local-2-0'),
          ]),
          isFalse,
        );
      },
    );

    test(
      'is true for a local-id waypoint with a null localKey -- the '
      'invariant break WR-04 guards against',
      () {
        expect(
          hasKeylessPendingWaypoint(const [
            (id: 'local-1-0', localKey: null),
          ]),
          isTrue,
        );
      },
    );

    test(
      'is false for a REAL server id with a null localKey -- an '
      'already-created waypoint needs no key',
      () {
        expect(
          hasKeylessPendingWaypoint(const [
            (id: 'server-1', localKey: null),
          ]),
          isFalse,
        );
      },
    );

    test(
      'is true for a mixed list containing one local-id/null-key entry',
      () {
        expect(
          hasKeylessPendingWaypoint(const [
            (id: 'server-1', localKey: 'k'),
            (id: 'local-1-0', localKey: null),
          ]),
          isTrue,
        );
      },
    );
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

  group('isLiveCaptureRow', () {
    TrailEntity buildEntity({
      String? owner,
      String? localId,
      required TrailSyncState syncState,
    }) {
      final entity = TrailEntity(
        id: 'server-id',
        name: 'Test Trail',
        created: DateTime(2026, 1, 1),
        updated: DateTime(2026, 1, 1),
        owner: owner,
        localId: localId,
        syncState: syncState,
      );
      return entity;
    }

    test('returns false for a plain downloaded row (owner: null, localId: '
        'null, syncState: synced)', () {
      final entity = buildEntity(
        owner: null,
        localId: null,
        syncState: TrailSyncState.synced,
      );

      expect(isLiveCaptureRow(entity), isFalse);
    });

    test('returns true for a downloaded row that also carries another '
        "account's carry-forward -- the CR-01/CR-03 overlap row "
        '(TrailDownloadService.downloadTrail\'s shape)', () {
      final entity = buildEntity(
        owner: 'account-a',
        localId: 'local-1-0',
        syncState: TrailSyncState.failed,
      );

      expect(isLiveCaptureRow(entity), isTrue);
    });

    test('returns false for an owned, synced row', () {
      final entity = buildEntity(
        owner: 'account-a',
        localId: 'local-1-0',
        syncState: TrailSyncState.synced,
      );

      expect(isLiveCaptureRow(entity), isFalse);
    });

    test('returns false when owner is null even with a non-null localId and '
        'a pending syncState', () {
      final entity = buildEntity(
        owner: null,
        localId: 'local-1-0',
        syncState: TrailSyncState.pending,
      );

      expect(isLiveCaptureRow(entity), isFalse);
    });
  });
}
