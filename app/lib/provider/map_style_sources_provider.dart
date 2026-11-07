import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/map_style_sources.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/services/map_source_persistence.dart';

part 'map_style_sources_provider.g.dart';

/// Fetches `/map/style-sources` and writes the result through to app-private
/// storage. The proxy builds redirect targets from these templates, and an
/// offline cold start must know them with no network call available.
///
/// On a failed fetch it falls back to the persisted copy. A first-ever run
/// with no network and nothing persisted genuinely has no sources, so the
/// original error is rethrown rather than fabricated away.
@Riverpod(keepAlive: true)
class MapStyleSourcesNotifier extends _$MapStyleSourcesNotifier {
  @override
  Future<MapStyleSources> build() async {
    final api = ref.watch(apiProvider);
    try {
      final response = await api.get('/map/style-sources');
      final sources = MapStyleSources.fromJson(response.data);
      final store = ref.read(objectBoxProvider);
      writePersistedMapStyleSources(store, sources);
      return sources;
    } catch (error) {
      final store = ref.read(objectBoxProvider);
      final persisted = readPersistedMapStyleSources(store);
      if (persisted != null) return persisted;
      rethrow;
    }
  }
}
