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
    final available = state.value?.availableServers ?? <ServerInstance>[];
    final newState = ServerState(available, server);
    state = AsyncValue.data(newState);
  }
}
