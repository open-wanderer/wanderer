import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/local_settings_entity.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/util/recent_servers.dart';

part 'recent_servers_provider.g.dart';

/// The persisted "Last used" instance list shown in the server picker.
///
/// `keepAlive` because [ref.read]-only writers reach this from the keepAlive
/// `Auth` notifier on login success (see `auth_provider.dart`), mirroring
/// `LocalSettingsNotifier`, which is keepAlive for the same reason.
@Riverpod(keepAlive: true)
class RecentServersNotifier extends _$RecentServersNotifier {
  @override
  List<ServerInstance> build() {
    final box = ref.watch(objectBoxProvider).box<LocalSettingsEntity>();
    final raw = box.getAll().firstOrNull?.recentServersJson ?? '';
    return decodeRecentServers(raw);
  }

  /// Records [server] as the most recently used instance.
  void record(ServerInstance server) {
    _persist(pushRecentServer(state, server));
  }

  /// Drops [server] from the list, e.g. from the picker's remove button.
  void remove(ServerInstance server) {
    final next = removeRecentServer(state, server);
    if (identical(next, state)) return;
    _persist(next);
  }

  /// Re-reads the row fresh before writing — the same read-modify-write
  /// every other [LocalSettingsEntity] writer follows (see
  /// `local_settings_provider.dart`, `map_source_persistence.dart`) — so a
  /// concurrent write to another field on the singleton row is never
  /// clobbered.
  void _persist(List<ServerInstance> next) {
    final box = ref.read(objectBoxProvider).box<LocalSettingsEntity>();
    final entity = box.getAll().firstOrNull ?? LocalSettingsEntity();
    entity.recentServersJson = encodeRecentServers(next);
    box.put(entity);

    state = next;
  }
}
