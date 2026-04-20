import 'dart:convert';

import 'package:freezed_annotation/freezed_annotation.dart';

part 'user.freezed.dart';
part 'user.g.dart';

@Freezed()
abstract class User with _$User {
  const factory User({
    required String id,
    required String username,
    required String email,
    required bool emailVisibility,
    required bool verified,
    required String created,
    required String updated,
    String? avatar,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);

  factory User.fromCookie(String cookieValue) {
    final decoded = Uri.decodeComponent(cookieValue);
    final map = jsonDecode(decoded) as Map<String, dynamic>;
    return User.fromJson(map['record'] as Map<String, dynamic>);
  }
}
