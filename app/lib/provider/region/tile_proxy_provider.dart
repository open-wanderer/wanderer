import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'tile_proxy_provider.g.dart';

/// Exposes the resolved loopback base URL (e.g.
/// `http://127.0.0.1:54321/<32-hex-secret>`) of the app-wide
/// [TileProxyServer] — same `keepAlive` "overridden in main.dart" shape as
/// `objectbox_store_provider.dart`. Port and secret are persisted and stable
/// across launches, not merely for the process lifetime, so there is no
/// `update*` method. The URL carries the per-install secret: never log it or
/// surface it in UI.
@Riverpod(keepAlive: true)
class TileProxyBaseUrl extends _$TileProxyBaseUrl {
  @override
  String build() {
    // This will be overridden in main.dart
    throw UnimplementedError();
  }
}
