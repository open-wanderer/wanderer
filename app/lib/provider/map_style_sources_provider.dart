import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/map_style_sources.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'map_style_sources_provider.g.dart';

@Riverpod(keepAlive: true)
class MapStyleSourcesNotifier extends _$MapStyleSourcesNotifier {
  @override
  Future<MapStyleSources> build() async {
    final api = ref.watch(apiProvider);
    final response = await api.get('/map/style-sources');
    return MapStyleSources.fromJson(response.data);
  }
}
