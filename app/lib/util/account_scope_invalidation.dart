/// One declarative list of every keepAlive Riverpod provider holding
/// account-scoped state, plus its invalidator — invalidated on any auth
/// user-id change so no in-memory cache can serve the previous account's
/// data after a switch (T-h2p-02).
///
/// Inclusion rule: a cache holding server data or draft content belonging to
/// the signed-in account, that outlives a screen because it is `keepAlive`.
///
/// Exclusion rules:
/// - Device/server-level state stays untouched: `apiProvider`,
///   `objectBoxProvider`, `cookieJarProvider`, `localSettings*`,
///   `mapCameraProvider`, `onlineStatusProvider`, `glyphSpriteCacheProvider`,
///   the map-style providers — none of these hold account-scoped data.
/// - `downloadingTrailIdsProvider` / `tileRepositoryManagerProvider` stay:
///   invalidating them mid-download would desync in-flight download
///   bookkeeping from its `CancelToken`s.
/// - Every region provider stays: downloaded regions are PUBLIC, device-level
///   data shared by all accounts on the device (they are expensive basemap
///   archives with no user-specific content and must never be re-downloaded
///   per account), so nothing about a region snapshot is account-scoped.
///   `regionListNotifierProvider`/`tileRepositoryStatusProvider` were briefly
///   listed here on the mistaken theory that an account switch detached
///   region download state; the real cause was the backend re-minting region
///   record ids, fixed by keying region identity on `path`.
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';
// `ProviderOrFamily` is a real exported type, but flutter_riverpod's main
// barrel does not re-export it — it lives in the package's `misc.dart`
// sub-library (confirmed via package source: flutter_riverpod-3.3.2/lib/misc.dart).
import 'package:flutter_riverpod/misc.dart' show ProviderOrFamily;
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';

/// Every keepAlive provider holding account-scoped state. Every entry is a
/// non-family provider, so no family handling is needed here.
final accountScopedProviders = <ProviderOrFamily>[
  ownProfileProvider,
  settingsProvider,
  trailLibraryProvider,
  trailSearchProvider,
  mapTrailSearchProvider,
  mapClusterSearchProvider,
  trailFilterProvider,
  trailSaveProvider,
  waypointSaveProvider,
  routeAnchorsProvider,
  categoryProvider,
  subcategoryProvider,
];

/// Invalidates every provider in [accountScopedProviders] from widget-tree
/// code.
void invalidateAccountScopedProviders(WidgetRef ref) {
  for (final provider in accountScopedProviders) {
    ref.invalidate(provider);
  }
}

/// Same invalidation loop, reachable from a plain [ProviderContainer]
/// without a widget tree (so it is unit-testable).
void invalidateAccountScopedProvidersOn(ProviderContainer container) {
  for (final provider in accountScopedProviders) {
    container.invalidate(provider);
  }
}
