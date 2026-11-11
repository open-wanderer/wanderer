import 'package:freezed_annotation/freezed_annotation.dart';

part 'list_result.freezed.dart';
part 'list_result.g.dart';

@Freezed(genericArgumentFactories: true)
abstract class ListResult<T> with _$ListResult<T> {
  const factory ListResult({
    required List<T> items,
    required int page,
    required int perPage,
    required int totalPages,
    required int totalItems,
  }) = _ListResult;

  factory ListResult.fromJson(
    Map<String, dynamic> json,
    T Function(Object?) fromJsonT,
  ) => _$ListResultFromJson(json, fromJsonT);
}
