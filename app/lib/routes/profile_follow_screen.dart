import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/base/wanderer_empty_state.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_follows_provider.dart';
import 'package:wanderer/util/format.dart';

class ProfileFollowScreen extends ConsumerStatefulWidget {
  final String handle;
  final String type;
  const ProfileFollowScreen({
    super.key,
    required this.handle,
    required this.type,
  });

  @override
  ConsumerState<ProfileFollowScreen> createState() =>
      _ProfileFollowScreenState();
}

class _ProfileFollowScreenState extends ConsumerState<ProfileFollowScreen> {
  late final ScrollController _scrollController;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _maybeLoadNextPage() {
    final s = ref.read(profileFollowsProvider(widget.handle, widget.type));
    if (s.value?.hasMore == true && !s.isLoading) {
      ref
          .read(profileFollowsProvider(widget.handle, widget.type).notifier)
          .loadNextPage();
    }
  }

  void _onScroll() {
    final pos = _scrollController.position;
    if (!pos.hasContentDimensions) return;
    if (pos.maxScrollExtent <= 0) return;
    if (pos.extentAfter < kPagedPrefetchExtent) {
      _maybeLoadNextPage();
    }
  }

  void _checkFillsViewport() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (!_scrollController.hasClients) return;
      final pos = _scrollController.position;
      if (pos.hasContentDimensions && pos.maxScrollExtent <= 0) {
        _maybeLoadNextPage();
      }
    });
  }

  String _title(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return widget.type == 'following' ? l.following : l.followers;
  }

  @override
  Widget build(BuildContext context) {
    final followsAsync = ref.watch(
      profileFollowsProvider(widget.handle, widget.type),
    );

    final ownUsername = ref.watch(authProvider).value?.preferredUsername;
    final isOwn = ownUsername != null && widget.handle == '@$ownUsername';

    ref.listen(profileFollowsProvider(widget.handle, widget.type), (_, next) {
      if (next.hasValue) _checkFillsViewport();
    });

    return Scaffold(
      appBar: AppBar(
        scrolledUnderElevation: 0,
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(_title(context)),
      ),
      body: AsyncLoader<ProfileFollowsState>(
        asyncValue: followsAsync,
        mockData: ProfileFollowsState.mock(),
        keepValueWhileLoading: true,
        builder: (state) => RefreshIndicator(
          color: Theme.of(context).colorScheme.onSurface,
          onRefresh: () async => ref.invalidate(
            profileFollowsProvider(widget.handle, widget.type),
          ),
          child: state.items.isEmpty
              ? _EmptyState(
                  type: widget.type,
                  isOwn: isOwn,
                  username: usernameFromHandle(widget.handle),
                )
              : ListView.builder(
                  // Not on the skeleton: it is on screen alongside the real
                  // list during the cross-fade, and a controller attached to
                  // two lists throws on `.position`.
                  controller: followsAsync.hasValue ? _scrollController : null,
                  itemCount: state.items.length + (state.hasMore ? 1 : 0),
                  itemBuilder: (context, index) {
                    if (index == state.items.length) {
                      return const Padding(
                        padding: EdgeInsets.symmetric(vertical: 16),
                        child: Center(child: CircularProgressIndicator()),
                      );
                    }
                    return _ActorTile(actor: state.items[index]);
                  },
                ),
        ),
      ),
    );
  }
}

class _ActorTile extends StatelessWidget {
  final Actor actor;
  const _ActorTile({required this.actor});

  String get _handleDisplay {
    if (actor.domain?.isNotEmpty == true) {
      return '@${actor.preferredUsername}@${actor.domain}';
    }
    return '@${actor.preferredUsername}';
  }

  String get _routeHandle {
    if (actor.domain?.isNotEmpty == true) {
      return '@${actor.preferredUsername}@${actor.domain}';
    }
    return '@${actor.preferredUsername}';
  }

  @override
  Widget build(BuildContext context) {
    return ListTile(
      // Replaced so the mock actor never fetches a fallback avatar.
      leading: Skeleton.replace(
        replacement: const Bone.circle(size: 44),
        child: ActorAvatar.fromActor(actor: actor, radius: 22),
      ),
      title: Text(
        actor.username,
        style: const TextStyle(fontWeight: FontWeight.w600),
      ),
      subtitle: Text(
        _handleDisplay,
        style: TextStyle(fontSize: 12, color: Colors.grey[600]),
      ),
      onTap: () => context.push('/profile/$_routeHandle'),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final String type;
  final bool isOwn;
  final String username;

  const _EmptyState({
    required this.type,
    required this.isOwn,
    required this.username,
  });

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    if (type == 'following') {
      return WandererEmptyState(
        icon: FontAwesomeIcons.userCheck,
        title: l.profile_following_empty_title,
        body: isOwn
            ? l.profile_following_empty_own_body
            : l.profile_following_empty_other_body(username),
        actionLabel: isOwn ? l.find_people : null,
        onAction: isOwn ? () => context.push('/search') : null,
      );
    }
    return WandererEmptyState(
      icon: FontAwesomeIcons.userGroup,
      title: l.profile_followers_empty_title,
      body: isOwn
          ? l.profile_followers_empty_own_body
          : l.profile_followers_empty_other_body(username),
      actionLabel: isOwn ? l.share_profile : null,
      onAction: isOwn ? () => context.push('/profile/share') : null,
    );
  }
}
