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
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order), appending each segment's polyline via `skip(1)` so the
/// shared boundary point between two consecutive segments isn't duplicated.
///
/// Never sets `ele`: the elevation tab owns the elevation-merged copy.

@ProviderFor(plannedGpx)
final plannedGpxProvider = PlannedGpxProvider._();

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider].
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order), appending each segment's polyline via `skip(1)` so the
/// shared boundary point between two consecutive segments isn't duplicated.
///
/// Never sets `ele`: the elevation tab owns the elevation-merged copy.

final class PlannedGpxProvider extends $FunctionalProvider<Gpx, Gpx, Gpx>
    with $Provider<Gpx> {
  /// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
  /// from the in-progress route held by [routeAnchorsProvider].
  ///
  /// Walks the anchor-id chain starting at `anchors.first`, following each
  /// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
  /// array order), appending each segment's polyline via `skip(1)` so the
  /// shared boundary point between two consecutive segments isn't duplicated.
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

String _$plannedGpxHash() => r'7016eabc1c55ba40c78dfff70be678bf4fca7a90';

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built from data already on `routeAnchorsProvider`'s
/// segments (elevation fetches happen there, fire-and-forget per segment).
///
/// Same anchor-chain walk as [plannedGpx], but each segment contributes its
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
/// Same anchor-chain walk as [plannedGpx], but each segment contributes its
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
  /// Same anchor-chain walk as [plannedGpx], but each segment contributes its
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
    r'e871230092d3a77c6f7126108e9561cfa8a98dd7';
