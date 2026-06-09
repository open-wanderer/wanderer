import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'profile_counts_provider.g.dart';

class ProfileCounts {
  final int trailCount;
  final int listCount;
  const ProfileCounts({required this.trailCount, required this.listCount});
}

@riverpod
Future<ProfileCounts> profileCounts(Ref ref, String actorId) async {
  final api = ref.watch(apiProvider);
  final results = await Future.wait([
    api.get("/trail?filter=author='$actorId'&perPage=1"),
    api.get("/list?filter=author='$actorId'&perPage=1"),
  ]);
  final trailCount = (results[0].data['totalItems'] as num?)?.toInt() ?? 0;
  final listCount = (results[1].data['totalItems'] as num?)?.toInt() ?? 0;
  return ProfileCounts(trailCount: trailCount, listCount: listCount);
}
