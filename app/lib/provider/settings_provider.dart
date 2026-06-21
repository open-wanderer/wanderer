import 'package:objectbox/objectbox.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/settings_entity.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'settings_provider.g.dart';

@Riverpod(keepAlive: true)
class SettingsNotifier extends _$SettingsNotifier {
  Box<SettingsEntity> get _box =>
      ref.read(objectBoxProvider).box<SettingsEntity>();

  @override
  Settings? build() {
    final box = ref.watch(objectBoxProvider).box<SettingsEntity>();
    return box.getAll().firstOrNull?.toModel();
  }

  /// Persists settings received from the server into ObjectBox and updates state.
  Future<void> updateFromServer(Settings settings) async {
    final existing = _box.getAll().firstOrNull;
    final entity = SettingsEntity.fromModel(settings);
    if (existing != null) {
      entity.obxId = existing.obxId;
    }
    _box.put(entity);
    ref.invalidateSelf();
  }

  /// POSTs updated settings to the server, then syncs the response locally.
  Future<void> saveToServer(Settings settings) async {
    final id = settings.id;
    if (id == null || id.isEmpty) {
      throw StateError('Cannot save settings: id is null or empty');
    }
    final response = await ref
        .read(apiProvider)
        .post('/settings/$id', data: settings.toJson());
    final updated = Settings.fromJson(response.data as Map<String, dynamic>);
    await updateFromServer(updated);
  }
}
