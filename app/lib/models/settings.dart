import 'package:freezed_annotation/freezed_annotation.dart';

part 'settings.freezed.dart';
part 'settings.g.dart';

enum Language {
  @JsonValue('cs')
  cs,
  @JsonValue('en')
  en,
  @JsonValue('de')
  de,
  @JsonValue('es')
  es,
  @JsonValue('eu')
  eu,
  @JsonValue('fr')
  fr,
  @JsonValue('hu')
  hu,
  @JsonValue('it')
  it,
  @JsonValue('nl')
  nl,
  @JsonValue('no')
  no,
  @JsonValue('pl')
  pl,
  @JsonValue('pt')
  pt,
  @JsonValue('ru')
  ru,
  @JsonValue('zh')
  zh,
}

enum NotificationType {
  @JsonValue('trail_share')
  trailShare,
  @JsonValue('trail_like')
  trailLike,
  @JsonValue('list_share')
  listShare,
  @JsonValue('new_follower')
  newFollower,
  @JsonValue('trail_comment')
  trailComment,
  @JsonValue('summit_log_create')
  summitLogCreate,
  @JsonValue('comment_mention')
  commentMention,
  @JsonValue('trail_mention')
  trailMention,
  @JsonValue('summit_log_mention')
  summitLogMention,
}

@freezed
abstract class SettingsLocation with _$SettingsLocation {
  const factory SettingsLocation({
    required String name,
    required double lat,
    required double lon,
  }) = _SettingsLocation;

  factory SettingsLocation.fromJson(Map<String, dynamic> json) =>
      _$SettingsLocationFromJson(json);
}

@freezed
abstract class SettingsPrivacy with _$SettingsPrivacy {
  const factory SettingsPrivacy({
    required String account,
    required String trails,
    required String lists,
  }) = _SettingsPrivacy;

  factory SettingsPrivacy.fromJson(Map<String, dynamic> json) =>
      _$SettingsPrivacyFromJson(json);
}

@freezed
abstract class NotificationPreference with _$NotificationPreference {
  const factory NotificationPreference({
    required bool web,
    required bool email,
  }) = _NotificationPreference;

  factory NotificationPreference.fromJson(Map<String, dynamic> json) =>
      _$NotificationPreferenceFromJson(json);
}

@freezed
abstract class Settings with _$Settings {
  const factory Settings({
    String? id,
    String? unit,
    Language? language,
    String? bio,
    SettingsLocation? location,
    String? category,
    String? user,
    SettingsPrivacy? privacy,
    Map<String, NotificationPreference>? notifications,
  }) = _Settings;

  factory Settings.fromJson(Map<String, dynamic> json) =>
      _$SettingsFromJson(json);
}
