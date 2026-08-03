import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/trail/trail_download_state_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/provider/account_scope_invalidation.dart';

// ---------------------------------------------------------------------------
// Tests for account_scope_invalidation's declarative provider list and its
// invalidator (T-h2p-02) — no ObjectBox store or widget tree is needed
// because invalidate() on a never-built provider is a no-op.
// ---------------------------------------------------------------------------

void main() {
  group('accountScopedProviders', () {
    test(
      'contains ownProfileProvider, settingsProvider and trailLibraryProvider',
      () {
        expect(accountScopedProviders, contains(ownProfileProvider));
        expect(accountScopedProviders, contains(settingsProvider));
        expect(accountScopedProviders, contains(trailLibraryProvider));
      },
    );

    test('contains no duplicate entries', () {
      expect(
        accountScopedProviders.toSet().length,
        accountScopedProviders.length,
      );
    });

    test(
      'does not contain device/server-level or in-flight-download providers',
      () {
        expect(accountScopedProviders, isNot(contains(localSettingsProvider)));
        expect(accountScopedProviders, isNot(contains(onlineStatusProvider)));
        expect(
          accountScopedProviders,
          isNot(contains(downloadingTrailIdsProvider)),
        );
      },
    );

    test('does not contain trailSyncProvider — a deliberate, test-pinned '
        'exclusion (36-04)', () {
      expect(
        accountScopedProviders,
        isNot(contains(trailSyncProvider)),
        reason:
            "trailSyncProvider tracks the deferred-upload drain's "
            "in-flight local-id set. Invalidating it mid-drain would "
            "desync that set from the upload sequence's actual position "
            "(the row may already have a server id from writeServerTrailId "
            "while the notifier still thinks it is pre-create). Account "
            "scoping already lives in the drain's own owner-filtered "
            "query (selectDrainCandidates), not in cache invalidation.",
      );
    });

    test('contains no region provider — regions are shared device data', () {
      // Downloaded regions are public basemap archives shared by every
      // account on the device and must never be re-downloaded per account,
      // so no region provider may be invalidated on an auth change. These
      // were briefly listed on the mistaken theory that an account switch
      // detached region download state; the real cause was the backend
      // re-minting region record ids.
      expect(
        accountScopedProviders,
        isNot(contains(regionListNotifierProvider)),
      );
      expect(
        accountScopedProviders,
        isNot(contains(tileRepositoryStatusProvider)),
      );
    });
  });

  group('invalidateAccountScopedProvidersOn', () {
    late ProviderContainer container;

    setUp(() {
      container = ProviderContainer();
    });

    tearDown(() {
      container.dispose();
    });

    test('completes without throwing and without building any provider', () {
      expect(
        () => invalidateAccountScopedProvidersOn(container),
        returnsNormally,
      );
    });
  });
}
