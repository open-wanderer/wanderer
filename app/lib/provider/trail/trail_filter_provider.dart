import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/trail/offline_filter_bounds.dart';

part 'trail_filter_provider.g.dart';

/// Builds the default [TrailFilter] from server- or device-derived [values].
///
/// Extracted so both the online success path and the offline fallback path
/// (see [TrailFilterNotifier.build]) produce a filter through the exact same
/// construction. `max == limit` on every bounded axis by construction here,
/// regardless of where [values] came from -- that is what makes
/// `toFilterText()` emit no upper-bound clause for a fallback filter, and it
/// is a property of this function, not of the numbers fed into it.
TrailFilter buildDefaultTrailFilter(TrailFilterValues values) {
  return TrailFilter(
    q: "",
    category: [],
    subcategory: [],
    tags: [],
    difficulty: [0, 1, 2],
    author: null,
    public: true,
    shared: true,
    liked: false,
    private: true,
    near: TrailNear(radius: 2000),
    distanceMin: 0,
    distanceMax: values.maxDistance,
    distanceLimit: values.maxDistance,
    elevationGainMin: 0,
    elevationGainMax: values.maxElevationGain,
    elevationGainLimit: values.maxElevationGain,
    elevationLossMin: 0,
    elevationLossMax: values.maxElevationLoss,
    elevationLossLimit: values.maxElevationLoss,
    sort: TrailFilterSort.created,
    sortOrder: SortOrder.desc,
  );
}

/// Retry policy for [TrailFilterNotifier], replacing
/// `ProviderContainer.defaultRetry`'s 10 attempts over ~45s -- which is what
/// turned a single failure into a sustained emission storm feeding
/// `ProfileTrailsNotifier`'s ~20 spinner flashes. Retries at most twice
/// (400ms, then 800ms) before giving up.
Duration? trailFilterRetry(int retryCount, Object error) {
  if (retryCount >= 2) return null;
  return Duration(milliseconds: 400 * (1 << retryCount));
}

@Riverpod(keepAlive: true, retry: trailFilterRetry)
class TrailFilterNotifier extends _$TrailFilterNotifier {
  /// Initialised at declaration, NOT `late` and NOT `late final`.
  ///
  /// It was `late final` once: Riverpod keeps one Notifier instance alive
  /// across rebuilds — only `build()` re-runs — so a `late final` field
  /// assigned inside `build()` threw `LateInitializationError: Field
  /// 'defaultFilter' has already been initialized` on the second build, which
  /// an account switch triggers every time (this provider is invalidated
  /// from `accountScopedProviders`), and which any other refresh would have
  /// hit too.
  ///
  /// Dropping to plain `late` "fixed" that but introduced a second bug:
  /// `build()` only assigns this field on two of its three exit
  /// paths (the success path and the connection-failure fallback). Any other
  /// failure — a 500, a malformed payload, `TrailFilterValues.fromJson`
  /// throwing — rethrows without assigning it, and `resetFilter()` reads it
  /// unconditionally from a button callback, so an un-initialised `late`
  /// field threw `LateInitializationError` there instead. The declaration-
  /// site initialiser removes both failure modes at once: there is no
  /// uninitialized state to reach, and no `late`/`late final` distinction to
  /// get wrong on a rebuild. `build()`'s success and connection-failure paths
  /// still overwrite this value; every other error still surfaces as an
  /// `AsyncError` rather than hiding behind this default.
  TrailFilter defaultFilter = buildDefaultTrailFilter(
    kOfflineTrailFilterValues,
  );

  @override
  Future<TrailFilter> build(String filterId) async {
    final api = ref.watch(apiProvider);

    try {
      final response = await api.get('/trail/filter');

      if (response.data == null) {
        throw Exception('No filter metadata received from server');
      }
      TrailFilterValues filterValues = TrailFilterValues.fromJson(
        response.data,
      );

      defaultFilter = buildDefaultTrailFilter(filterValues);
      return defaultFilter;
    } catch (e) {
      // Distinguish "offline" from "broken": a connection failure means
      // there is genuinely nothing to fetch, so it degrades to a
      // device-derived fallback filter instead of throwing -- with no
      // AsyncError there is no retry loop, so ProfileTrailsNotifier sees one
      // clean emission instead of ~20, and the quick-filter bar renders
      // usable chips offline instead of nothing.
      //
      // The bounds are computed at build time, so a trail recorded while the
      // sheet is already open does not widen the sliders until the provider
      // is next invalidated. That is accepted -- trailFilterProvider is
      // keepAlive and is already invalidated on account switch via
      // accountScopedProviders, and a slider that resizes underneath a
      // hiker mid-drag would be worse than a slightly stale one.
      //
      // Every other error (a 500, a malformed payload, a client-side bug)
      // must still surface rather than hide behind a silent default.
      if (e is DioException && isConnectionFailure(e)) {
        defaultFilter = buildDefaultTrailFilter(_offlineFilterValues());
        return defaultFilter;
      }
      throw Exception('Failed to fetch trail filters: $e');
    }
  }

  /// Computes the offline fallback's [TrailFilterValues] from this device's
  /// own trails.
  ///
  /// Uses `ref.read`, not `ref.watch`, for [objectBoxProvider]:
  /// `objectBoxProvider` is a `$SyncValueProvider` that provably never
  /// re-emits (`main.dart` overrides it once with one `Store` instance at
  /// `runApp`), and `ref.read` cannot create a dependency edge at all -- so
  /// this cannot reintroduce the rebuild storm this plan exists to remove.
  ///
  /// A null account returns [kOfflineTrailFilterValues]: no signed-in
  /// account means no account scope, and per `current_account.dart`'s
  /// contract that means an EMPTY read, never an unfiltered one. An
  /// unfiltered read here would let the slider's maximum disclose the
  /// length of a signed-out account's downloaded trails.
  ///
  /// The whole read is wrapped in one try/catch that degrades to
  /// [kOfflineTrailFilterValues]: `ref.read(objectBoxProvider)` throws
  /// `UnimplementedError` in any `ProviderContainer` that does not override
  /// it, which is every test in this repo. Degrading to the constant keeps
  /// `test/provider/trail_filter_provider_test.dart` green and keeps a
  /// genuine ObjectBox fault from turning "you are offline" into "the filter
  /// provider is broken".
  TrailFilterValues _offlineFilterValues() {
    try {
      final store = ref.read(objectBoxProvider);
      final accountId = currentAccountId(store);
      if (accountId == null) return kOfflineTrailFilterValues;
      return computeOfflineTrailFilterValues(
        readLocalTrailMetrics(store, accountId: accountId),
      );
    } catch (e, st) {
      debugPrint(
        'trail_filter_provider: _offlineFilterValues falling back to '
        'kOfflineTrailFilterValues -- $e\n$st',
      );
      return kOfflineTrailFilterValues;
    }
  }

  void resetFilter() {
    state = AsyncData(defaultFilter);
  }

  void updateFilter(TrailFilter Function(TrailFilter current) updater) {
    final currentState = state.value;
    if (currentState == null) return;

    state = AsyncData(updater(currentState));
  }

  void addTag(Tag tag) {
    updateFilter((f) => f.copyWith(tags: [...f.tags, tag]));
  }

  void removeTag(Tag tag) {
    updateFilter(
      (f) => f.copyWith(
        tags: f.tags
            .where(
              (t) => t.name != tag.name && (t.id == null || t.id != tag.id),
            )
            .toList(),
      ),
    );
  }
}
