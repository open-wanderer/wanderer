// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'route_anchor_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
/// argument fixed for the notifier's lifetime (D-07).

@ProviderFor(RouteAnchors)
final routeAnchorsProvider = RouteAnchorsFamily._();

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
/// argument fixed for the notifier's lifetime (D-07).
final class RouteAnchorsProvider
    extends $NotifierProvider<RouteAnchors, RouteAnchorsState> {
  /// Route-planner state provider: owns the ordered anchor list, the
  /// per-segment Valhalla routing engine (with a race-guard against
  /// out-of-order responses), the geometric segment-split used by a plain
  /// insert tap, and an immutable-snapshot undo/redo stack.
  ///
  /// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
  /// argument fixed for the notifier's lifetime (D-07).
  RouteAnchorsProvider._({
    required RouteAnchorsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'routeAnchorsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$routeAnchorsHash();

  @override
  String toString() {
    return r'routeAnchorsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  RouteAnchors create() => RouteAnchors();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RouteAnchorsState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RouteAnchorsState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is RouteAnchorsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$routeAnchorsHash() => r'1974ca7aff66711e439e9a982c8e039b5a6fc099';

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
/// argument fixed for the notifier's lifetime (D-07).

final class RouteAnchorsFamily extends $Family
    with
        $ClassFamilyOverride<
          RouteAnchors,
          RouteAnchorsState,
          RouteAnchorsState,
          RouteAnchorsState,
          String
        > {
  RouteAnchorsFamily._()
    : super(
        retry: null,
        name: r'routeAnchorsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Route-planner state provider: owns the ordered anchor list, the
  /// per-segment Valhalla routing engine (with a race-guard against
  /// out-of-order responses), the geometric segment-split used by a plain
  /// insert tap, and an immutable-snapshot undo/redo stack.
  ///
  /// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
  /// argument fixed for the notifier's lifetime (D-07).

  RouteAnchorsProvider call(String travelProfile) =>
      RouteAnchorsProvider._(argument: travelProfile, from: this);

  @override
  String toString() => r'routeAnchorsProvider';
}

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// `travelProfile` (`'pedestrian'` | `'bicycle'`) is a required family
/// argument fixed for the notifier's lifetime (D-07).

abstract class _$RouteAnchors extends $Notifier<RouteAnchorsState> {
  late final _$args = ref.$arg as String;
  String get travelProfile => _$args;

  RouteAnchorsState build(String travelProfile);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<RouteAnchorsState, RouteAnchorsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<RouteAnchorsState, RouteAnchorsState>,
              RouteAnchorsState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
