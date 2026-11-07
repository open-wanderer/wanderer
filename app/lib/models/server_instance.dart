import 'package:freezed_annotation/freezed_annotation.dart';

part 'server_instance.freezed.dart';
part 'server_instance.g.dart';

@freezed
abstract class ServerInstance with _$ServerInstance {
  const factory ServerInstance({
    String? name,
    required String url,
    String? description,
    String? image,
    @Default([]) List<String> region,
    @Default([]) List<String> language,
    @Default([]) List<String> category,
  }) = _ServerInstance;

  factory ServerInstance.fromJson(Map<String, dynamic> json) =>
      _$ServerInstanceFromJson(json);
}
