import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'current_server_provider.g.dart';

@riverpod
class CurrentServer extends _$CurrentServer {
  @override
  String? build() {
    return null;
  }

  void update(String url) {
    state = url;
  }
}
