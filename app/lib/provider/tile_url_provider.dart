import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'tile_url_provider.g.dart';

@Riverpod(keepAlive: true)
Future<String> tileUrl(Ref ref) async {
  final api = ref.watch(apiProvider);
  final response = await api.get('/map/tileurl');
  return response.data['url'] as String;
}
