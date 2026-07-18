// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'planned_gpx_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order, Pitfall 3), appending each segment's polyline via
/// `skip(1)` so the shared boundary point between two consecutive segments
/// is not duplicated.
///
/// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
/// changes. Never sets `ele` (D-10): the elevation tab owns the
/// elevation-merged copy.

@ProviderFor(plannedGpx)
final plannedGpxProvider = PlannedGpxProvider._();

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order, Pitfall 3), appending each segment's polyline via
/// `skip(1)` so the shared boundary point between two consecutive segments
/// is not duplicated.
///
/// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
/// changes. Never sets `ele` (D-10): the elevation tab owns the
/// elevation-merged copy.

final class PlannedGpxProvider extends $FunctionalProvider<Gpx, Gpx, Gpx>
    with $Provider<Gpx> {
  /// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
  /// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
  ///
  /// Walks the anchor-id chain starting at `anchors.first`, following each
  /// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
  /// array order, Pitfall 3), appending each segment's polyline via
  /// `skip(1)` so the shared boundary point between two consecutive segments
  /// is not duplicated.
  ///
  /// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
  /// changes. Never sets `ele` (D-10): the elevation tab owns the
  /// elevation-merged copy.
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
/// tab's chart, built entirely from data already living on
/// `routeAnchorsProvider`'s segments — no network call of its own (that
/// fetch now lives on `route_anchor_provider.dart`'s `_resolveElevation`,
/// fired fire-and-forget per segment on creation/update).
///
/// Same anchor-chain walk as [plannedGpx] (`beforeAnchorId -> afterAnchorId`
/// links, not array order), but each segment contributes its own
/// [RouteSegment.elevationProfile] points (falling back to [RouteSegment.polyline]
/// when a segment's height fetch hasn't resolved yet) paired with
/// [RouteSegment.elevations] — `ele` stays `null` for any point beyond
/// what's been fetched so far, which the elevation chart already treats as
/// `0` (transient, until that segment's fetch resolves).

@ProviderFor(plannedElevationGpx)
final plannedElevationGpxProvider = PlannedElevationGpxProvider._();

/// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
/// tab's chart, built entirely from data already living on
/// `routeAnchorsProvider`'s segments — no network call of its own (that
/// fetch now lives on `route_anchor_provider.dart`'s `_resolveElevation`,
/// fired fire-and-forget per segment on creation/update).
///
/// Same anchor-chain walk as [plannedGpx] (`beforeAnchorId -> afterAnchorId`
/// links, not array order), but each segment contributes its own
/// [RouteSegment.elevationProfile] points (falling back to [RouteSegment.polyline]
/// when a segment's height fetch hasn't resolved yet) paired with
/// [RouteSegment.elevations] — `ele` stays `null` for any point beyond
/// what's been fetched so far, which the elevation chart already treats as
/// `0` (transient, until that segment's fetch resolves).

final class PlannedElevationGpxProvider
    extends $FunctionalProvider<Gpx, Gpx, Gpx>
    with $Provider<Gpx> {
  /// Sibling of [plannedGpx]: an elevation-bearing `Gpx` for the Elevation
  /// tab's chart, built entirely from data already living on
  /// `routeAnchorsProvider`'s segments — no network call of its own (that
  /// fetch now lives on `route_anchor_provider.dart`'s `_resolveElevation`,
  /// fired fire-and-forget per segment on creation/update).
  ///
  /// Same anchor-chain walk as [plannedGpx] (`beforeAnchorId -> afterAnchorId`
  /// links, not array order), but each segment contributes its own
  /// [RouteSegment.elevationProfile] points (falling back to [RouteSegment.polyline]
  /// when a segment's height fetch hasn't resolved yet) paired with
  /// [RouteSegment.elevations] — `ele` stays `null` for any point beyond
  /// what's been fetched so far, which the elevation chart already treats as
  /// `0` (transient, until that segment's fetch resolves).
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
