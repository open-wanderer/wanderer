// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'settings.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_SettingsLocation _$SettingsLocationFromJson(Map<String, dynamic> json) =>
    _SettingsLocation(
      name: json['name'] as String,
      lat: (json['lat'] as num).toDouble(),
      lon: (json['lon'] as num).toDouble(),
    );

Map<String, dynamic> _$SettingsLocationToJson(_SettingsLocation instance) =>
    <String, dynamic>{
      'name': instance.name,
      'lat': instance.lat,
      'lon': instance.lon,
    };

_SettingsPrivacy _$SettingsPrivacyFromJson(Map<String, dynamic> json) =>
    _SettingsPrivacy(
      account: json['account'] as String,
      trails: json['trails'] as String,
      lists: json['lists'] as String,
    );

Map<String, dynamic> _$SettingsPrivacyToJson(_SettingsPrivacy instance) =>
    <String, dynamic>{
      'account': instance.account,
      'trails': instance.trails,
      'lists': instance.lists,
    };

_NotificationPreference _$NotificationPreferenceFromJson(
  Map<String, dynamic> json,
) => _NotificationPreference(
  web: json['web'] as bool,
  email: json['email'] as bool,
);

Map<String, dynamic> _$NotificationPreferenceToJson(
  _NotificationPreference instance,
) => <String, dynamic>{'web': instance.web, 'email': instance.email};

_Settings _$SettingsFromJson(Map<String, dynamic> json) => _Settings(
  id: json['id'] as String?,
  unit: json['unit'] as String?,
  language: $enumDecodeNullable(_$LanguageEnumMap, json['language']),
  bio: json['bio'] as String?,
  location: json['location'] == null
      ? null
      : SettingsLocation.fromJson(json['location'] as Map<String, dynamic>),
  category: json['category'] as String?,
  user: json['user'] as String?,
  privacy: json['privacy'] == null
      ? null
      : SettingsPrivacy.fromJson(json['privacy'] as Map<String, dynamic>),
  notifications: (json['notifications'] as Map<String, dynamic>?)?.map(
    (k, e) =>
        MapEntry(k, NotificationPreference.fromJson(e as Map<String, dynamic>)),
  ),
);

Map<String, dynamic> _$SettingsToJson(_Settings instance) => <String, dynamic>{
  'id': instance.id,
  'unit': instance.unit,
  'language': _$LanguageEnumMap[instance.language],
  'bio': instance.bio,
  'location': instance.location,
  'category': instance.category,
  'user': instance.user,
  'privacy': instance.privacy,
  'notifications': instance.notifications,
};

const _$LanguageEnumMap = {
  Language.cs: 'cs',
  Language.en: 'en',
  Language.de: 'de',
  Language.es: 'es',
  Language.eu: 'eu',
  Language.fr: 'fr',
  Language.hu: 'hu',
  Language.it: 'it',
  Language.nl: 'nl',
  Language.no: 'no',
  Language.pl: 'pl',
  Language.pt: 'pt',
  Language.ru: 'ru',
  Language.zh: 'zh',
};
