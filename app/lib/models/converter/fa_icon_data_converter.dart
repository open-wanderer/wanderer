import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/util/icon_util.dart';

class FaIconDataConverter implements JsonConverter<FaIconData, String> {
  const FaIconDataConverter();

  @override
  FaIconData fromJson(String? json) {
    return fontAwesomeIconsMap[json] ?? FontAwesomeIcons.circle;
  }

  @override
  String toJson(FaIconData object) {
    return fontAwesomeIconsMapReversed[object] ?? "circle";
  }
}
