import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/geo/polyline.dart';

part 'trail_polyline_provider.g.dart';

@riverpod
Future<List<Geographic>?> trailPolyline(Ref ref, String trailId) async {
  final api = ref.read(apiProvider);

  final response = await api.get('/trail/$trailId');
  if (response.data == null) return null;

  final trail = Trail.fromJson(response.data);
  final polylineStr = trail.polyline;
  if (polylineStr == null || polylineStr.isEmpty) return null;

  return PolylineUtil.decode(polylineStr);
}
