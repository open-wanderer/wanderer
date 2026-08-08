import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/follow.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

part 'follow_provider.g.dart';

@riverpod
class FollowNotifier extends _$FollowNotifier {
  @override
  FutureOr<FollowState> build(String profileActorId) async {
    final api = ref.read(apiProvider);
    final user = ref.read(authProvider).requireValue;
    if (user == null) return const FollowState(isFollowing: false);

    final response = await api.get(
      '/follow',
      queryParameters: {
        'filter': "follower='${user.actorId}'&&followee='$profileActorId'",
      },
    );

    final items = (response.data['items'] as List?) ?? [];
    if (items.isEmpty) return const FollowState(isFollowing: false);
    final follow = items.first as Map<String, dynamic>;
    return FollowState(
      isFollowing: true,
      followRecordId: follow['id'] as String,
    );
  }

  Future<void> toggle() async {
    final current = state.value;
    if (current == null || current.isLoading) return;

    final wasFollowing = current.isFollowing;

    // Optimistic flip
    state = AsyncData(
      current.copyWith(isFollowing: !wasFollowing, isLoading: true),
    );

    try {
      final api = ref.read(apiProvider);
      if (!wasFollowing) {
        // Follow — PUT /api/v1/follow  body: { followee: profileActorId }
        final response = await api.put(
          '/follow',
          data: {'followee': profileActorId},
        );
        final newId = (response.data as Map<String, dynamic>)['id'] as String;
        state = AsyncData(
          FollowState(isFollowing: true, followRecordId: newId),
        );
      } else {
        // Unfollow — DELETE /api/v1/follow/:id
        await api.delete('/follow/${current.followRecordId}');
        state = AsyncData(const FollowState(isFollowing: false));
      }
    } catch (_) {
      // Roll back optimistic update and show error toast
      state = AsyncData(current.copyWith(isLoading: false));
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: wasFollowing ? 'Failed to unfollow.' : 'Failed to follow.',
            ),
          );
    }
  }
}
