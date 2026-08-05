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
/// - `trailSyncProvider` stays, deliberately, for the same reason as
///   `downloadingTrailIdsProvider` above: it is a `keepAlive` provider
///   tracking in-flight operations (the deferred-upload drain's local-id
///   set), and invalidating it mid-drain would desync that set from the
///   upload sequence's actual position (`writeServerTrailId` may already
///   have committed while the notifier's own state still thinks the trail
///   is pre-create). Account scoping for the drain is already enforced at
///   the query, not by cache invalidation: `drainIfOnline` re-reads
///   `currentAccountId(store)` fresh on every run and `selectDrainCandidates`
///   filters on that id via an owner predicate (D-13). This is a deliberate
///   decision, not an omission — see 36-RESEARCH.md, which flags this
///   provider as the one case the `downloadingTrailIdsProvider` precedent
///   does not automatically cover.
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
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';

/// Every keepAlive provider holding account-scoped state.
///
/// `trailFilterProvider` is a FAMILY (`TrailFilterNotifierFamily`, keyed by
/// filter id); invalidating the family object invalidates all of its
/// instances, which is exactly what an account switch needs.
/// `mapTrailSearchProvider` and `mapClusterSearchProvider` are also FAMILIES
/// (keyed by `(authorId, filterId)`, see `profile_trail_map_screen.dart`) for
/// the same reason: a family-level invalidate hits every instance regardless
/// of which profile map created it. Every other entry here is a plain
/// provider — note `ownProfileProvider`, not the `profileProvider` family
/// that lives beside it.
///
/// Riverpod keeps one Notifier instance alive across rebuilds and only re-runs
/// `build()`, so anything listed here must tolerate `build()` running more than
/// once on the same instance. A `late final` field assigned in `build()` will
/// throw `LateInitializationError` on the second pass (this is what happened to
/// `TrailFilterNotifier.defaultFilter`).
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
  // 38.1 CR-01. Both are FAMILIES keyed only on `localId` and both resolve
  // `currentAccountId(store)` INSIDE `build()`, so the account is baked into
  // a cache entry whose key does not mention it. Auto-dispose does not save
  // them: it only fires when the LAST listener goes away, and Flutter keeps
  // pushed-under routes mounted (`maintainState: true`). A screen retained
  // on the back stack across an account switch therefore keeps serving
  // account A's answer to account B -- for `localTrailProvider` that renders
  // A's not-yet-uploaded trail to B (the exact leak `readOwnLocalTrail`'s
  // owner clause exists to prevent, arriving through the cache instead of
  // the query), and for `ownLiveCaptureProvider` it re-arms
  // `trail_dropdown.dart`'s `_allowDelete` escape hatch with no authorship
  // check. A cached value also never recomputes when the drain retires the
  // row, so Delete stays offered on a trail that is now fully synced.
  localTrailProvider,
  ownLiveCaptureProvider,
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
