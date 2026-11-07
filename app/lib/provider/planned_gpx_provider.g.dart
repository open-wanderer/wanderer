// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'planned_gpx_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider].
///
/// Walks [RouteAnchorsState.orderedSegments], appending each segment's
/// polyline via `skip(1)` so the shared boundary point between two
/// consecutive segments isn't duplicated. The leading single-point `Trkseg`
/// carries the start anchor.
///
/// This layout is for in-app consumers that read the route as one continuous
/// point stream (`gpx.allPoints`) — its `trkseg` boundaries sit one point
/// *after* each anchor, so it is NOT round-trippable through
/// `anchorsFromTrack`. Route export uses [buildFinalPlannedGpx] instead,
/// which emits whole legs including both endpoints.
///
/// Never sets `ele`: the elevation tab owns the elevation-merged copy.

@ProviderFor(plannedGpx)
final plannedGpxProvider = PlannedGpxProvider._();

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider].
///
/// Walks [RouteAnchorsState.orderedSegments], appending each segment's
/// polyline via `skip(1)` so the shared boundary point between two
/// consecutive segments isn't duplicated. The leading single-point `Trkseg`
/// carries the start anchor.
///
/// This layout is for in-app consumers that read the route as one continuous
/// point stream (`gpx.allPoints`) — its `trkseg` boundaries sit one point
/// *after* each anchor, so it is NOT round-trippable through
/// `anchorsFromTrack`. Route export uses [buildFinalPlannedGpx] instead,
/// which emits whole legs including both endpoints.
///
/// Never sets `ele`: the elevation tab owns the elevation-merged copy.

final class PlannedGpxProvider extends $FunctionalProvider<Gpx, Gpx, Gpx>
    with $Provider<Gpx> {
  /// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
  /// from the in-progress route held by [routeAnchorsProvider].
  ///
  /// Walks [RouteAnchorsState.orderedSegments], appending each segment's
  /// polyline via `skip(1)` so the shared boundary point between two
  /// consecutive segments isn't duplicated. The leading single-point `Trkseg`
  /// carries the start anchor.
  ///
  /// This layout is for in-app consumers that read the route as one continuous
  /// point stream (`gpx.allPoints`) — its `trkseg` boundaries sit one point
  /// *after* each anchor, so it is NOT round-trippable through
  /// `anchorsFromTrack`. Route export uses [buildFinalPlannedGpx] instead,
  /// which emits whole legs including both endpoints.
  ///
  /// Never sets `ele`: the elevation tab owns the elevation-merged copy.
  PlannedGpxProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'plannedGpxProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$plannedGpxHash();

  @$internal
  @override
  $ProviderElement<Gpx> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Gpx create(Ref ref) {
    return plannedGpx(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Gpx value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Gpx>(value),
    );
  }
}

String _$plannedGpxHash() => r'320a52783e92694b1083a6f969764792d988c813';

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built from data already on `routeAnchorsProvider`'s
/// segments (elevation fetches happen there, fire-and-forget per segment).
///
/// Same layout and [RouteAnchorsState.orderedSegments] walk as [plannedGpx]
/// (and the same non-round-trippable caveat), but each segment contributes its
/// [RouteSegment.elevationProfile] points (falling back to
/// [RouteSegment.polyline] while a segment's height fetch is pending) paired
/// with [RouteSegment.elevations]; `ele` stays `null` for points not yet
/// fetched, which the chart treats as `0` until that segment resolves.

@ProviderFor(plannedElevationGpx)
final plannedElevationGpxProvider = PlannedElevationGpxProvider._();

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built from data already on `routeAnchorsProvider`'s
/// segments (elevation fetches happen there, fire-and-forget per segment).
///
/// Same layout and [RouteAnchorsState.orderedSegments] walk as [plannedGpx]
/// (and the same non-round-trippable caveat), but each segment contributes its
/// [RouteSegment.elevationProfile] points (falling back to
/// [RouteSegment.polyline] while a segment's height fetch is pending) paired
/// with [RouteSegment.elevations]; `ele` stays `null` for points not yet
/// fetched, which the chart treats as `0` until that segment resolves.

final class PlannedElevationGpxProvider
    extends $FunctionalProvider<Gpx, Gpx, Gpx>
    with $Provider<Gpx> {
  /// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
  /// tab's chart, built from data already on `routeAnchorsProvider`'s
  /// segments (elevation fetches happen there, fire-and-forget per segment).
  ///
  /// Same layout and [RouteAnchorsState.orderedSegments] walk as [plannedGpx]
  /// (and the same non-round-trippable caveat), but each segment contributes its
  /// [RouteSegment.elevationProfile] points (falling back to
  /// [RouteSegment.polyline] while a segment's height fetch is pending) paired
  /// with [RouteSegment.elevations]; `ele` stays `null` for points not yet
  /// fetched, which the chart treats as `0` until that segment resolves.
  PlannedElevationGpxProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'plannedElevationGpxProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$plannedElevationGpxHash();

  @$internal
  @override
  $ProviderElement<Gpx> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Gpx create(Ref ref) {
    return plannedElevationGpx(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Gpx value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Gpx>(value),
    );
  }
}

String _$plannedElevationGpxHash() =>
    r'8635ffebb4dc7148db335134b5e6ced66557dac1';
