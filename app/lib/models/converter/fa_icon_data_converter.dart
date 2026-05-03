import 'package:flutter/widgets.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:freezed_annotation/freezed_annotation.dart';

class FaIconDataConverter
    implements JsonConverter<FaIconData, Map<String, dynamic>> {
  const FaIconDataConverter();

  @override
  FaIconData fromJson(Map<String, dynamic> json) {
    IconData data = IconData(
      json['codePoint'] as int,
      fontFamily: json['fontFamily'] as String?,
      fontPackage: json['fontPackage'] as String?,
    );

    return FaIconData(data);
  }

  @override
  Map<String, dynamic> toJson(FaIconData object) {
    return {
      'codePoint': object.codePoint,
      'fontFamily': object.fontFamily,
      'fontPackage': object.fontPackage,
    };
  }
}
