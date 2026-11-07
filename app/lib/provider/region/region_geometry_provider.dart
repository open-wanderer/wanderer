import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/region_geometry.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/region/file_path.dart';

part 'region_geometry_provider.g.dart';

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
@riverpod
Future<RegionGeometry> regionGeometry(Ref ref, String path) async {
  final validated = assertValidRegionPath(path);
  final api = ref.watch(apiProvider);
  final response = await api.get('/regions/$validated/geometry');
  return RegionGeometry.fromJson(response.data as Map<String, dynamic>);
}
