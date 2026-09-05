// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_trail_bounding_box_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Per-profile-handle bounding box, backing the initial camera fit for
/// `ProfileTrailMapScreen`.
///
/// This is autoDispose, deliberately not keepAlive: a stale bbox for a
/// profile whose trails changed would frame the map wrongly, and one extra
/// request per screen open is cheap relative to the search requests that
/// follow.
///
/// Never surfaces an error to the UI. The world-view fallback path is
/// first-class here, not an edge case: on a mixed-version federation
/// the remote will commonly not support the `handle` param, and the correct
/// behaviour is a silent degrade to the default camera with the bounds
/// search still running. Any `DioException`, timeout, or parse failure
/// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
/// than propagating an exception.

@ProviderFor(profileTrailBoundingBox)
final profileTrailBoundingBoxProvider = ProfileTrailBoundingBoxFamily._();

/// Per-profile-handle bounding box, backing the initial camera fit for
/// `ProfileTrailMapScreen`.
///
/// This is autoDispose, deliberately not keepAlive: a stale bbox for a
/// profile whose trails changed would frame the map wrongly, and one extra
/// request per screen open is cheap relative to the search requests that
/// follow.
///
/// Never surfaces an error to the UI. The world-view fallback path is
/// first-class here, not an edge case: on a mixed-version federation
/// the remote will commonly not support the `handle` param, and the correct
/// behaviour is a silent degrade to the default camera with the bounds
/// search still running. Any `DioException`, timeout, or parse failure
/// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
/// than propagating an exception.

final class ProfileTrailBoundingBoxProvider
    extends
        $FunctionalProvider<
          AsyncValue<TrailBoundingBox>,
          TrailBoundingBox,
          FutureOr<TrailBoundingBox>
        >
    with $FutureModifier<TrailBoundingBox>, $FutureProvider<TrailBoundingBox> {
  /// Per-profile-handle bounding box, backing the initial camera fit for
  /// `ProfileTrailMapScreen`.
  ///
  /// This is autoDispose, deliberately not keepAlive: a stale bbox for a
  /// profile whose trails changed would frame the map wrongly, and one extra
  /// request per screen open is cheap relative to the search requests that
  /// follow.
  ///
  /// Never surfaces an error to the UI. The world-view fallback path is
  /// first-class here, not an edge case: on a mixed-version federation
  /// the remote will commonly not support the `handle` param, and the correct
  /// behaviour is a silent degrade to the default camera with the bounds
  /// search still running. Any `DioException`, timeout, or parse failure
  /// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
  /// than propagating an exception.
  ProfileTrailBoundingBoxProvider._({
    required ProfileTrailBoundingBoxFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileTrailBoundingBoxProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileTrailBoundingBoxHash();

  @override
  String toString() {
    return r'profileTrailBoundingBoxProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<TrailBoundingBox> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<TrailBoundingBox> create(Ref ref) {
    final argument = this.argument as String;
    return profileTrailBoundingBox(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ProfileTrailBoundingBoxProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileTrailBoundingBoxHash() =>
    r'c4477fd40efba68ef3118aafdf0b5bc96c94db15';

/// Per-profile-handle bounding box, backing the initial camera fit for
/// `ProfileTrailMapScreen`.
///
/// This is autoDispose, deliberately not keepAlive: a stale bbox for a
/// profile whose trails changed would frame the map wrongly, and one extra
/// request per screen open is cheap relative to the search requests that
/// follow.
///
/// Never surfaces an error to the UI. The world-view fallback path is
/// first-class here, not an edge case: on a mixed-version federation
/// the remote will commonly not support the `handle` param, and the correct
/// behaviour is a silent degrade to the default camera with the bounds
/// search still running. Any `DioException`, timeout, or parse failure
/// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
/// than propagating an exception.

final class ProfileTrailBoundingBoxFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<TrailBoundingBox>, String> {
  ProfileTrailBoundingBoxFamily._()
    : super(
        retry: null,
        name: r'profileTrailBoundingBoxProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Per-profile-handle bounding box, backing the initial camera fit for
  /// `ProfileTrailMapScreen`.
  ///
  /// This is autoDispose, deliberately not keepAlive: a stale bbox for a
  /// profile whose trails changed would frame the map wrongly, and one extra
  /// request per screen open is cheap relative to the search requests that
  /// follow.
  ///
  /// Never surfaces an error to the UI. The world-view fallback path is
  /// first-class here, not an edge case: on a mixed-version federation
  /// the remote will commonly not support the `handle` param, and the correct
  /// behaviour is a silent degrade to the default camera with the bounds
  /// search still running. Any `DioException`, timeout, or parse failure
  /// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
  /// than propagating an exception.

  ProfileTrailBoundingBoxProvider call(String handle) =>
      ProfileTrailBoundingBoxProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileTrailBoundingBoxProvider';
}
