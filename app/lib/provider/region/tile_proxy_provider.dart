import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'tile_proxy_provider.g.dart';

/// Exposes the resolved loopback base URL (e.g. `http://127.0.0.1:54321`) of
/// the app-wide [TileProxyServer] — mirrors `objectbox_store_provider.dart`'s
/// exact `keepAlive` "overridden in main.dart" shape. The server's port is
/// fixed for the app process lifetime (it never needs to stop), so there is
/// no `update*` method here, unlike `api_provider.dart`'s mutable base URL.
@Riverpod(keepAlive: true)
class TileProxyBaseUrl extends _$TileProxyBaseUrl {
  @override
  String build() {
    // This will be overridden in main.dart
    throw UnimplementedError();
  }
}
