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

String _$plannedGpxHash() => r'66fc4f46edb79bdca1eec997569448d491c8ea66';
