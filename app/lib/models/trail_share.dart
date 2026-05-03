import 'package:freezed_annotation/freezed_annotation.dart';
import 'actor.dart';

part 'trail_share.freezed.dart';
part 'trail_share.g.dart';

enum TrailPermission {
  @JsonValue('view')
  view,
  @JsonValue('edit')
  edit,
}

@freezed
abstract class TrailShareExpand with _$TrailShareExpand {
  const factory TrailShareExpand({required Actor actor}) = _TrailShareExpand;

  factory TrailShareExpand.fromJson(Map<String, dynamic> json) =>
      _$TrailShareExpandFromJson(json);
}

@freezed
abstract class TrailShare with _$TrailShare {
  const factory TrailShare({
    String? id,
    required String actor,
    required String trail,
    required TrailPermission permission,
    TrailShareExpand? expand,
  }) = _TrailShare;

  factory TrailShare.fromJson(Map<String, dynamic> json) =>
      _$TrailShareFromJson(json);
}
