// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_style_sources_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Fetches `/map/style-sources` and writes the result through to app-private
/// storage. The proxy builds redirect targets from these templates, and an
/// offline cold start must know them with no network call available.
///
/// On a failed fetch it falls back to the persisted copy. A first-ever run
/// with no network and nothing persisted genuinely has no sources, so the
/// original error is rethrown rather than fabricated away.

@ProviderFor(MapStyleSourcesNotifier)
final mapStyleSourcesProvider = MapStyleSourcesNotifierProvider._();

/// Fetches `/map/style-sources` and writes the result through to app-private
/// storage. The proxy builds redirect targets from these templates, and an
/// offline cold start must know them with no network call available.
///
/// On a failed fetch it falls back to the persisted copy. A first-ever run
/// with no network and nothing persisted genuinely has no sources, so the
/// original error is rethrown rather than fabricated away.
final class MapStyleSourcesNotifierProvider
    extends $AsyncNotifierProvider<MapStyleSourcesNotifier, MapStyleSources> {
  /// Fetches `/map/style-sources` and writes the result through to app-private
  /// storage. The proxy builds redirect targets from these templates, and an
  /// offline cold start must know them with no network call available.
  ///
  /// On a failed fetch it falls back to the persisted copy. A first-ever run
  /// with no network and nothing persisted genuinely has no sources, so the
  /// original error is rethrown rather than fabricated away.
  MapStyleSourcesNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapStyleSourcesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapStyleSourcesNotifierHash();

  @$internal
  @override
  MapStyleSourcesNotifier create() => MapStyleSourcesNotifier();
}

String _$mapStyleSourcesNotifierHash() =>
    r'a81959bf89f1cc3daf318d83bdf0170295209489';

/// Fetches `/map/style-sources` and writes the result through to app-private
/// storage. The proxy builds redirect targets from these templates, and an
/// offline cold start must know them with no network call available.
///
/// On a failed fetch it falls back to the persisted copy. A first-ever run
/// with no network and nothing persisted genuinely has no sources, so the
/// original error is rethrown rather than fabricated away.

abstract class _$MapStyleSourcesNotifier
    extends $AsyncNotifier<MapStyleSources> {
  FutureOr<MapStyleSources> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<MapStyleSources>, MapStyleSources>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<MapStyleSources>, MapStyleSources>,
              AsyncValue<MapStyleSources>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
