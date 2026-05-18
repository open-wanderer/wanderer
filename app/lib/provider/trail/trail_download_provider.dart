import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/services/trail_download_service.dart';

part 'trail_download_provider.g.dart';

@riverpod
class TrailDownloadServiceNotifier extends _$TrailDownloadServiceNotifier {
  @override
  TrailDownloadService build() {
    final store = ref.watch(objectBoxProvider);
    final api = ref.watch(apiProvider);

    return TrailDownloadService(store, api);
  }
}
