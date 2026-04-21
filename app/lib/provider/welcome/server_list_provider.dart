import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/server_instance.dart';

part 'server_list_provider.g.dart';

@riverpod
class ServerDirectory extends _$ServerDirectory {
  @override
  Future<List<ServerInstance>> build() async {
    final dio = Dio();
    final response = await dio.get('https://wanderer.to/server/servers.json');

    final List<dynamic> data = response.data;
    return data.map((json) => ServerInstance.fromJson(json)).toList();
  }
}
