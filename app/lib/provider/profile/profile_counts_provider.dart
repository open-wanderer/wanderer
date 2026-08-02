import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/pb_filter_util.dart';

part 'profile_counts_provider.g.dart';

class ProfileCounts {
  final int trailCount;
  final int listCount;
  const ProfileCounts({required this.trailCount, required this.listCount});
}

@riverpod
Future<ProfileCounts> profileCounts(Ref ref, String actorId) async {
  final api = ref.watch(apiProvider);
  // `actorId` is a server-issued id rather than free text, so this is defence
  // in depth rather than a live hole — but the raw-interpolation pattern is
  // what spreads, and the next collection it lands on may not be as harmless.
  final author = escapePbFilterValue(actorId);
  final results = await Future.wait([
    api.get(
      '/trail',
      queryParameters: {'filter': "author='$author'", 'perPage': 1},
    ),
    api.get(
      '/list',
      queryParameters: {'filter': "author='$author'", 'perPage': 1},
    ),
  ]);
  final trailCount = (results[0].data['totalItems'] as num?)?.toInt() ?? 0;
  final listCount = (results[1].data['totalItems'] as num?)?.toInt() ?? 0;
  return ProfileCounts(trailCount: trailCount, listCount: listCount);
}
