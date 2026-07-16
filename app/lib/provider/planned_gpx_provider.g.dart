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
/// Keyed by [travelProfile] — the same family key as [routeAnchorsProvider]
/// — so Phase 21's handoff (HANDOFF-01) can consume the same provider.
/// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
/// changes. Never sets `ele` (D-10): the elevation tab owns the
/// elevation-merged copy.

@ProviderFor(plannedGpx)
final plannedGpxProvider = PlannedGpxFamily._();

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order, Pitfall 3), appending each segment's polyline via
/// `skip(1)` so the shared boundary point between two consecutive segments
/// is not duplicated.
///
/// Keyed by [travelProfile] — the same family key as [routeAnchorsProvider]
/// — so Phase 21's handoff (HANDOFF-01) can consume the same provider.
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
  /// Keyed by [travelProfile] — the same family key as [routeAnchorsProvider]
  /// — so Phase 21's handoff (HANDOFF-01) can consume the same provider.
  /// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
  /// changes. Never sets `ele` (D-10): the elevation tab owns the
  /// elevation-merged copy.
  PlannedGpxProvider._({
    required PlannedGpxFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'plannedGpxProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$plannedGpxHash();

  @override
  String toString() {
    return r'plannedGpxProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<Gpx> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Gpx create(Ref ref) {
    final argument = this.argument as String;
    return plannedGpx(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Gpx value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Gpx>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is PlannedGpxProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$plannedGpxHash() => r'311b908cc8a3743d937ff2ed276cf37ca707ef1c';

/// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
/// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
///
/// Walks the anchor-id chain starting at `anchors.first`, following each
/// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
/// array order, Pitfall 3), appending each segment's polyline via
/// `skip(1)` so the shared boundary point between two consecutive segments
/// is not duplicated.
///
/// Keyed by [travelProfile] — the same family key as [routeAnchorsProvider]
/// — so Phase 21's handoff (HANDOFF-01) can consume the same provider.
/// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
/// changes. Never sets `ele` (D-10): the elevation tab owns the
/// elevation-merged copy.

final class PlannedGpxFamily extends $Family
    with $FunctionalFamilyOverride<Gpx, String> {
  PlannedGpxFamily._()
    : super(
        retry: null,
        name: r'plannedGpxProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Derives an ordered, pre-elevation [Gpx] skeleton (points only, no `ele`)
  /// from the in-progress route held by [routeAnchorsProvider] (PLANUI-02).
  ///
  /// Walks the anchor-id chain starting at `anchors.first`, following each
  /// segment's `beforeAnchorId -> afterAnchorId` link (not `state.segments`
  /// array order, Pitfall 3), appending each segment's polyline via
  /// `skip(1)` so the shared boundary point between two consecutive segments
  /// is not duplicated.
  ///
  /// Keyed by [travelProfile] — the same family key as [routeAnchorsProvider]
  /// — so Phase 21's handoff (HANDOFF-01) can consume the same provider.
  /// Recomputes whenever `routeAnchorsProvider`'s anchors/segments identity
  /// changes. Never sets `ele` (D-10): the elevation tab owns the
  /// elevation-merged copy.

  PlannedGpxProvider call(String travelProfile) =>
      PlannedGpxProvider._(argument: travelProfile, from: this);

  @override
  String toString() => r'plannedGpxProvider';
}
