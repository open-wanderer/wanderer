import 'package:freezed_annotation/freezed_annotation.dart';
import 'actor.dart';
import 'trail.dart';

part 'trail_like.freezed.dart';
part 'trail_like.g.dart';

@freezed
abstract class TrailLikeExpand with _$TrailLikeExpand {
  const factory TrailLikeExpand({required Actor actor, required Trail trail}) =
      _TrailLikeExpand;

  factory TrailLikeExpand.fromJson(Map<String, dynamic> json) =>
      _$TrailLikeExpandFromJson(json);
}

@freezed
abstract class TrailLike with _$TrailLike {
  const factory TrailLike({
    String? id,
    required String actor,
    required String trail,
    TrailLikeExpand? expand,
  }) = _TrailLike;

  factory TrailLike.fromJson(Map<String, dynamic> json) =>
      _$TrailLikeFromJson(json);
}
