// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tile_proxy_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Exposes the resolved loopback base URL (e.g.
/// `http://127.0.0.1:54321/<32-hex-secret>`) of the app-wide
/// [TileProxyServer] — same `keepAlive` "overridden in main.dart" shape as
/// `objectbox_store_provider.dart`. Port and secret are persisted and stable
/// across launches, not merely for the process lifetime, so there is no
/// `update*` method. The URL carries the per-install secret: never log it or
/// surface it in UI.

@ProviderFor(TileProxyBaseUrl)
final tileProxyBaseUrlProvider = TileProxyBaseUrlProvider._();

/// Exposes the resolved loopback base URL (e.g.
/// `http://127.0.0.1:54321/<32-hex-secret>`) of the app-wide
/// [TileProxyServer] — same `keepAlive` "overridden in main.dart" shape as
/// `objectbox_store_provider.dart`. Port and secret are persisted and stable
/// across launches, not merely for the process lifetime, so there is no
/// `update*` method. The URL carries the per-install secret: never log it or
/// surface it in UI.
final class TileProxyBaseUrlProvider
    extends $NotifierProvider<TileProxyBaseUrl, String> {
  /// Exposes the resolved loopback base URL (e.g.
  /// `http://127.0.0.1:54321/<32-hex-secret>`) of the app-wide
  /// [TileProxyServer] — same `keepAlive` "overridden in main.dart" shape as
  /// `objectbox_store_provider.dart`. Port and secret are persisted and stable
  /// across launches, not merely for the process lifetime, so there is no
  /// `update*` method. The URL carries the per-install secret: never log it or
  /// surface it in UI.
  TileProxyBaseUrlProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tileProxyBaseUrlProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tileProxyBaseUrlHash();

  @$internal
  @override
  TileProxyBaseUrl create() => TileProxyBaseUrl();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(String value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<String>(value),
    );
  }
}

String _$tileProxyBaseUrlHash() => r'175844101b9b032f7116539bff9b0e32788b77ed';

/// Exposes the resolved loopback base URL (e.g.
/// `http://127.0.0.1:54321/<32-hex-secret>`) of the app-wide
/// [TileProxyServer] — same `keepAlive` "overridden in main.dart" shape as
/// `objectbox_store_provider.dart`. Port and secret are persisted and stable
/// across launches, not merely for the process lifetime, so there is no
/// `update*` method. The URL carries the per-install secret: never log it or
/// surface it in UI.

abstract class _$TileProxyBaseUrl extends $Notifier<String> {
  String build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<String, String>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<String, String>,
              String,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
