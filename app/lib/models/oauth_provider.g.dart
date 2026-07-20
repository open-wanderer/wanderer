// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'oauth_provider.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_OAuthProvider _$OAuthProviderFromJson(Map<String, dynamic> json) =>
    _OAuthProvider(
      name: json['name'] as String,
      displayName: json['displayName'] as String,
      state: json['state'] as String,
      codeVerifier: json['codeVerifier'] as String,
      url: json['url'] as String,
      img: json['img'] as String?,
    );

Map<String, dynamic> _$OAuthProviderToJson(_OAuthProvider instance) =>
    <String, dynamic>{
      'name': instance.name,
      'displayName': instance.displayName,
      'state': instance.state,
      'codeVerifier': instance.codeVerifier,
      'url': instance.url,
      'img': instance.img,
    };
