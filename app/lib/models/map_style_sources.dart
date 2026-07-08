import 'package:freezed_annotation/freezed_annotation.dart';

part 'map_style_sources.freezed.dart';
part 'map_style_sources.g.dart';

@freezed
abstract class MapStyleSources with _$MapStyleSources {
  const factory MapStyleSources({
    @JsonKey(name: 'tileUrl') required String tileUrl,
    @JsonKey(name: 'glyphUrl') required String glyphUrl,
    @JsonKey(name: 'spriteUrl') required String spriteUrl,
  }) = _MapStyleSources;

  factory MapStyleSources.fromJson(Map<String, dynamic> json) =>
      _$MapStyleSourcesFromJson(json);
}
