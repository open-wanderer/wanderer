import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/user.dart';

part 'auth_response.freezed.dart';
part 'auth_response.g.dart';

@Freezed()
abstract class AuthResponse with _$AuthResponse {
  const factory AuthResponse({required User record, required String token}) =
      _AuthResponse;

  factory AuthResponse.fromJson(Map<String, dynamic> json) =>
      _$AuthResponseFromJson(json);
}
