import 'package:freezed_annotation/freezed_annotation.dart';

part 'follow.freezed.dart';

@freezed
abstract class FollowState with _$FollowState {
  const factory FollowState({
    required bool isFollowing,
    String? followRecordId,
    @Default(false) bool isLoading,
  }) = _FollowState;
}
