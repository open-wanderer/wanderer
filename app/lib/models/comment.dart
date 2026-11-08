import 'package:freezed_annotation/freezed_annotation.dart';
import 'actor.dart';

part 'comment.freezed.dart';
part 'comment.g.dart';

@freezed
abstract class CommentExpand with _$CommentExpand {
  const factory CommentExpand({required Actor author}) = _CommentExpand;

  factory CommentExpand.fromJson(Map<String, dynamic> json) =>
      _$CommentExpandFromJson(json);
}

@freezed
abstract class Comment with _$Comment {
  const factory Comment({
    required String id,
    required String text,
    required String author,
    required String trail,
    required DateTime created,
    required DateTime updated,
    String? iri,
    CommentExpand? expand,
  }) = _Comment;

  factory Comment.fromJson(Map<String, dynamic> json) =>
      _$CommentFromJson(json);
}
