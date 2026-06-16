import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/profile/profile_follows_provider.dart';

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

  void _onScroll() {
    final pos = _scrollController.position;
    if (!pos.hasContentDimensions) return;
    if (pos.maxScrollExtent <= 0) return;
    if (pos.pixels / pos.maxScrollExtent >= 0.8) {
      final s = ref.read(
        profileFollowsProvider(widget.handle, widget.type),
      );
      if (s.value?.hasMore == true && !s.isLoading) {
        ref
            .read(profileFollowsProvider(widget.handle, widget.type).notifier)
            .loadNextPage();
      }
    }
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

    return Scaffold(
      appBar: AppBar(
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(_title(context)),
      ),
      body: followsAsync.when(
        data: (state) => RefreshIndicator(
          onRefresh: () async => ref.invalidate(
            profileFollowsProvider(widget.handle, widget.type),
          ),
          child: state.items.isEmpty
              ? _EmptyState(type: widget.type)
              : ListView.builder(
                  controller: _scrollController,
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
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, stack) => WandererError(err: err, stack: stack),
      ),
    );
  }
}

class _ActorTile extends StatelessWidget {
  final Actor actor;
  const _ActorTile({required this.actor});

  String get _avatarUrl =>
      actor.icon?.isNotEmpty == true
          ? actor.icon!
          : 'https://api.dicebear.com/7.x/initials/png?seed=${actor.preferredUsername}&backgroundType=gradientLinear';

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
      leading: CircleAvatar(
        radius: 22,
        backgroundColor: Colors.grey.shade300,
        backgroundImage: NetworkImage(_avatarUrl),
        onBackgroundImageError: (e, _) {},
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
  const _EmptyState({required this.type});

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    final label = type == 'following' ? l.following : l.followers;
    return Center(
      child: Text(
        'No $label yet.',
        style: TextStyle(color: Colors.grey[600]),
      ),
    );
  }
}
