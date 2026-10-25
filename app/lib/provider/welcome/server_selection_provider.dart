import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/server_instance.dart';

part 'server_selection_provider.g.dart';

class ServerState {
  List<ServerInstance> availableServers;
  ServerInstance? selectedServer;

  ServerState(this.availableServers, this.selectedServer);
}

@riverpod
class ServerSelectionNotifier extends _$ServerSelectionNotifier {
  @override
  Future<ServerState> build() async {
    final dio = Dio();
    final response = await dio.get('https://wanderer.to/server/servers.json');

    final List<dynamic> data = response.data;
    return ServerState(
      data.map((json) => ServerInstance.fromJson(json)).toList(),
      null,
    );
  }

  void setSelectedServer(ServerInstance server) {
    // A "Last used" entry (quick-260926-ijp) can be tapped while the
    // servers.json fetch is still loading or has failed (e.g. wanderer.to is
    // down but a self-hosted instance is up), so `requireValue` would throw
    // here. Falling back to an empty list on that window is accepted: it is
    // sub-second, and the user simply reselects if the fetch then overwrites
    // this state.
    final newState = ServerState(
      state.value?.availableServers ?? const [],
      server,
    );
    state = AsyncValue.data(newState);
  }
}
