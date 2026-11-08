import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail_sync_state.dart';

void main() {
  group('resolveSyncDisplayState', () {
    test('synced stays synced, even when its id is in flight', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.synced,
          localId: 'l',
          inFlight: {'l'},
        ),
        TrailSyncState.synced,
      );
    });

    test('a draft stays a draft, even when its id is in flight', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.draft,
          localId: 'l',
          inFlight: {'l'},
        ),
        TrailSyncState.draft,
        reason:
            'A draft is never a drain candidate; showing it as Uploading '
            'would hide the Upload/Discard actions.',
      );
    });

    test('pending with its id in flight reads as uploading', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.pending,
          localId: 'l',
          inFlight: {'l'},
        ),
        TrailSyncState.uploading,
      );
    });

    test('pending with a null localId stays pending', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.pending,
          localId: null,
          inFlight: {'other'},
        ),
        TrailSyncState.pending,
      );
    });

    test(
      'persisted uploading reads as uploading without an in-flight entry',
      () {
        expect(
          resolveSyncDisplayState(
            TrailSyncState.uploading,
            localId: 'l',
            inFlight: {},
          ),
          TrailSyncState.uploading,
        );
      },
    );

    test('failed, not in flight, stays failed', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.failed,
          localId: 'l',
          inFlight: {},
        ),
        TrailSyncState.failed,
      );
    });

    test('failed WITH its id in flight reads as uploading', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.failed,
          localId: 'l',
          inFlight: {'l'},
        ),
        TrailSyncState.uploading,
      );
    });

    test('pending, not in flight, stays pending', () {
      expect(
        resolveSyncDisplayState(
          TrailSyncState.pending,
          localId: 'l',
          inFlight: {},
        ),
        TrailSyncState.pending,
      );
    });
  });
}
