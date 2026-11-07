// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_geometry_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Fetches a single region's cached boundary geometry from
/// `GET /regions/{path}/geometry`, keyed by the region's materialized path.
///
/// Auto-dispose (no `keepAlive`): one region's outline is a screen-scoped
/// read and the map screen is the only consumer.
///
/// [path] is validated via [assertValidRegionPath] before being interpolated
/// into the request URL — the same defense-in-depth guard
/// `tile_repository_manager.dart` applies before building a region request
/// URL; the path is never string-concatenated into a URL unvalidated.

@ProviderFor(regionGeometry)
final regionGeometryProvider = RegionGeometryFamily._();

/// Fetches a single region's cached boundary geometry from
/// `GET /regions/{path}/geometry`, keyed by the region's materialized path.
///
/// Auto-dispose (no `keepAlive`): one region's outline is a screen-scoped
/// read and the map screen is the only consumer.
///
/// [path] is validated via [assertValidRegionPath] before being interpolated
/// into the request URL — the same defense-in-depth guard
/// `tile_repository_manager.dart` applies before building a region request
/// URL; the path is never string-concatenated into a URL unvalidated.

final class RegionGeometryProvider
    extends
        $FunctionalProvider<
          AsyncValue<RegionGeometry>,
          RegionGeometry,
          FutureOr<RegionGeometry>
        >
    with $FutureModifier<RegionGeometry>, $FutureProvider<RegionGeometry> {
  /// Fetches a single region's cached boundary geometry from
  /// `GET /regions/{path}/geometry`, keyed by the region's materialized path.
  ///
  /// Auto-dispose (no `keepAlive`): one region's outline is a screen-scoped
  /// read and the map screen is the only consumer.
  ///
  /// [path] is validated via [assertValidRegionPath] before being interpolated
  /// into the request URL — the same defense-in-depth guard
  /// `tile_repository_manager.dart` applies before building a region request
  /// URL; the path is never string-concatenated into a URL unvalidated.
  RegionGeometryProvider._({
    required RegionGeometryFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'regionGeometryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$regionGeometryHash();

  @override
  String toString() {
    return r'regionGeometryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<RegionGeometry> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<RegionGeometry> create(Ref ref) {
    final argument = this.argument as String;
    return regionGeometry(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is RegionGeometryProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$regionGeometryHash() => r'd87f2665491a67b4adfb6dfa5aea96982b2dba1a';

/// Fetches a single region's cached boundary geometry from
/// `GET /regions/{path}/geometry`, keyed by the region's materialized path.
///
/// Auto-dispose (no `keepAlive`): one region's outline is a screen-scoped
/// read and the map screen is the only consumer.
///
/// [path] is validated via [assertValidRegionPath] before being interpolated
/// into the request URL — the same defense-in-depth guard
/// `tile_repository_manager.dart` applies before building a region request
/// URL; the path is never string-concatenated into a URL unvalidated.

final class RegionGeometryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<RegionGeometry>, String> {
  RegionGeometryFamily._()
    : super(
        retry: null,
        name: r'regionGeometryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Fetches a single region's cached boundary geometry from
  /// `GET /regions/{path}/geometry`, keyed by the region's materialized path.
  ///
  /// Auto-dispose (no `keepAlive`): one region's outline is a screen-scoped
  /// read and the map screen is the only consumer.
  ///
  /// [path] is validated via [assertValidRegionPath] before being interpolated
  /// into the request URL — the same defense-in-depth guard
  /// `tile_repository_manager.dart` applies before building a region request
  /// URL; the path is never string-concatenated into a URL unvalidated.

  RegionGeometryProvider call(String path) =>
      RegionGeometryProvider._(argument: path, from: this);

  @override
  String toString() => r'regionGeometryProvider';
}
