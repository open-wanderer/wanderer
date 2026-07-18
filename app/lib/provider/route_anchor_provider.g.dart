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
/// Rec B: a single `@Riverpod(keepAlive: true)` provider with NO family
/// argument — `travelProfile` + `costingOptions` live fully inside state and
/// are switched via [switchProfile] (mid-session bucket change) or
/// [resetForSession] (called once per planner entry, since `keepAlive` means
/// this single instance survives across sheet/screen mounts and must be
/// reset so a re-entry never leaks the previous session's route).

@ProviderFor(RouteAnchors)
final routeAnchorsProvider = RouteAnchorsProvider._();

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// Rec B: a single `@Riverpod(keepAlive: true)` provider with NO family
/// argument — `travelProfile` + `costingOptions` live fully inside state and
/// are switched via [switchProfile] (mid-session bucket change) or
/// [resetForSession] (called once per planner entry, since `keepAlive` means
/// this single instance survives across sheet/screen mounts and must be
/// reset so a re-entry never leaks the previous session's route).
final class RouteAnchorsProvider
    extends $NotifierProvider<RouteAnchors, RouteAnchorsState> {
  /// Route-planner state provider: owns the ordered anchor list, the
  /// per-segment Valhalla routing engine (with a race-guard against
  /// out-of-order responses), the geometric segment-split used by a plain
  /// insert tap, and an immutable-snapshot undo/redo stack.
  ///
  /// Rec B: a single `@Riverpod(keepAlive: true)` provider with NO family
  /// argument — `travelProfile` + `costingOptions` live fully inside state and
  /// are switched via [switchProfile] (mid-session bucket change) or
  /// [resetForSession] (called once per planner entry, since `keepAlive` means
  /// this single instance survives across sheet/screen mounts and must be
  /// reset so a re-entry never leaks the previous session's route).
  RouteAnchorsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'routeAnchorsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$routeAnchorsHash();

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
}

String _$routeAnchorsHash() => r'8795cf6e241b994121ba9d5c903ce015537486d9';

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// Rec B: a single `@Riverpod(keepAlive: true)` provider with NO family
/// argument — `travelProfile` + `costingOptions` live fully inside state and
/// are switched via [switchProfile] (mid-session bucket change) or
/// [resetForSession] (called once per planner entry, since `keepAlive` means
/// this single instance survives across sheet/screen mounts and must be
/// reset so a re-entry never leaks the previous session's route).

abstract class _$RouteAnchors extends $Notifier<RouteAnchorsState> {
  RouteAnchorsState build();
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
    element.handleCreate(ref, build);
  }
}
