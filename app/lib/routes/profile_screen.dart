import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/list/list_card.dart';
import 'package:wanderer/components/profile/feed_item_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/profile/follow_provider.dart';
import 'package:wanderer/provider/profile/profile_feed_provider.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';

class ProfileScreen extends ConsumerStatefulWidget {
  final String? handle;
  const ProfileScreen({super.key, this.handle});

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  late final ScrollController _scrollController;

  bool get isOwn => widget.handle == null;

  /// Resolves the handle to use for data providers.
  /// For own profile, reads from authProvider; for remote, uses widget.handle.
  String? get _handle =>
      widget.handle ?? ref.read(authProvider).value?.preferredUsername;

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
    final ratio = pos.pixels / pos.maxScrollExtent;
    if (ratio >= 0.8) {
      final h = _handle;
      if (h == null) return;
      final feed = ref.read(profileFeedProvider(h));
      if (feed.value?.hasMore == true && !feed.isLoading) {
        ref.read(profileFeedProvider(h).notifier).loadNextPage();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final actorAsync = widget.handle != null
        ? ref.watch(profileProvider(widget.handle!))
        : ref.watch(ownProfileProvider);

    return Scaffold(
      body: actorAsync.when(
        data: (actor) => _buildProfile(actor),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, stack) => WandererError(err: err, stack: stack),
      ),
    );
  }

  Widget _buildProfile(Actor actor) {
    final h = _handle;
    final feedAsync = h != null ? ref.watch(profileFeedProvider(h)) : null;
    final feedItems = feedAsync?.value?.items ?? [];
    final feedHasMore = feedAsync?.value?.hasMore ?? false;
    final feedIsInitialLoading =
        feedAsync?.isLoading == true && feedItems.isEmpty;

    return RefreshIndicator(
      onRefresh: () async {
        if (widget.handle != null) {
          ref.invalidate(profileProvider(widget.handle!));
        } else {
          ref.invalidate(ownProfileProvider);
        }
        if (h != null) {
          ref.invalidate(profileFeedProvider(h));
          ref.invalidate(profileListsProvider(h));
        }
      },
      child: CustomScrollView(
        controller: _scrollController,
        slivers: [
          SliverAppBar(
            pinned: true,
            expandedHeight: 200,
            flexibleSpace: FlexibleSpaceBar(
              title: Builder(
                builder: (context) {
                  final settings = context
                      .dependOnInheritedWidgetOfExactType<
                        FlexibleSpaceBarSettings
                      >();
                  const fadeRange = 16.0;
                  final opacity = settings == null
                      ? 0.0
                      : 1.0 -
                            ((settings.currentExtent - settings.minExtent) /
                                    fadeRange)
                                .clamp(0.0, 1.0);
                  return Opacity(
                    opacity: opacity,
                    child: Text(actor.preferredUsername),
                  );
                },
              ),
              centerTitle: false,
              background: _ProfileHeaderBackground(actor: actor, isOwn: isOwn),
            ),
            actions: _actionButtons(actor),
          ),

          // Bio section
          SliverToBoxAdapter(child: _BioSection(summary: actor.summary)),

          // Stats row — follower/following counts + follow button
          SliverToBoxAdapter(
            child: _StatsRow(actor: actor, isOwn: isOwn),
          ),

          // Lists heading + preview
          if (h != null)
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Text(
                  AppLocalizations.of(context)!.list(2),
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
          if (h != null) SliverToBoxAdapter(child: _ListsPreview(handle: h)),

          // Feed heading
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: Text(
                "Feed",
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
          ),

          // Feed — loaded items
          if (!feedIsInitialLoading && feedItems.isNotEmpty)
            SliverList(
              delegate: SliverChildBuilderDelegate(
                (context, index) =>
                    FeedItemCard(item: feedItems[index], profileActor: actor),
                childCount: feedItems.length,
              ),
            ),

          // Feed — loading next page footer
          if (!feedIsInitialLoading && feedHasMore)
            const SliverToBoxAdapter(
              child: Center(child: CircularProgressIndicator()),
            ),

          // Feed — error state
          if (feedAsync?.hasError == true)
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  'Failed to load feed.',
                  style: TextStyle(color: Colors.grey[600]),
                  textAlign: TextAlign.center,
                ),
              ),
            ),

          // Bottom padding
          const SliverToBoxAdapter(child: SizedBox(height: 24)),
        ],
      ),
    );
  }

  List<Widget> _actionButtons(Actor actor) {
    if (isOwn) {
      return [
        IconButton(
          icon: const FaIcon(FontAwesomeIcons.gear, size: 16),
          onPressed: () => context.push('/settings'),
        ),
        IconButton(
          icon: const FaIcon(FontAwesomeIcons.shareNodes, size: 16),
          onPressed: () {
            // Share screen wired in Phase 3
          },
        ),
      ];
    } else {
      return [SizedBox.shrink()];
    }
  }
}

// ---------------------------------------------------------------------------
// Header background — avatar, handle, joined date, follower/following counts
// ---------------------------------------------------------------------------

class _ProfileHeaderBackground extends StatelessWidget {
  final Actor actor;
  final bool isOwn;
  const _ProfileHeaderBackground({required this.actor, required this.isOwn});

  @override
  Widget build(BuildContext context) {
    final locale = Localizations.localeOf(context).toString();
    final joinedDate = DateFormat.yMMMM(locale).format(actor.created);
    final handleDisplay =
        '@${actor.preferredUsername}${(actor.domain?.isNotEmpty == true) ? '@${actor.domain}' : ''}';

    return SafeArea(
      child: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            CircleAvatar(
              radius: 44,
              backgroundColor: Colors.grey.shade300,
              backgroundImage: NetworkImage(
                actor.icon?.isNotEmpty == true
                    ? actor.icon!
                    : 'https://api.dicebear.com/7.x/initials/png?seed=${actor.preferredUsername}&backgroundType=gradientLinear',
              ),
              onBackgroundImageError: (e, _) {},
            ),
            const SizedBox(height: 8),
            Text(
              actor.username,
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
            ),
            Text(
              handleDisplay,
              style: TextStyle(fontSize: 12, color: Colors.grey[600]),
            ),
            const SizedBox(height: 8),
            Text(
              '${AppLocalizations.of(context)!.joined} $joinedDate',
              style: TextStyle(fontSize: 12, color: Colors.grey[600]),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Bio section — show more / show less expand/collapse
// ---------------------------------------------------------------------------

class _BioSection extends StatefulWidget {
  final String? summary;

  const _BioSection({required this.summary});

  @override
  State<_BioSection> createState() => _BioSectionState();
}

class _BioSectionState extends State<_BioSection> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final summary = widget.summary;
    if (summary == null || summary.isEmpty) {
      return Padding(
        padding: const EdgeInsets.all(16),
        child: Text('No bio yet.', style: TextStyle(color: Colors.grey[600])),
      );
    }

    final bioMaxLength = 150;
    final truncated = summary.length > bioMaxLength && !_expanded;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Html(
            data: truncated
                ? '${summary.substring(0, bioMaxLength)}…'
                : summary,
          ),
          if (summary.length > bioMaxLength)
            TextButton(
              onPressed: () => setState(() => _expanded = !_expanded),
              child: Text(_expanded ? 'Show less' : 'Show more'),
            ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Lists preview — horizontal scroll of ListCards
// ---------------------------------------------------------------------------

class _ListsPreview extends ConsumerWidget {
  final String handle;

  const _ListsPreview({required this.handle});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final listsAsync = ref.watch(profileListsProvider(handle));

    return listsAsync.when(
      data: (state) {
        final lists = state.lists;
        if (lists.isEmpty) return const SizedBox.shrink();
        return SizedBox(
          height: 170,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 16),
            itemCount: lists.length,
            separatorBuilder: (context, _) => const SizedBox(width: 8),
            itemBuilder: (context, index) => ListCard(list: lists[index]),
          ),
        );
      },
      loading: () => const SizedBox(
        height: 170,
        child: Center(child: CircularProgressIndicator()),
      ),
      error: (err, _) => const SizedBox.shrink(),
    );
  }
}

// ---------------------------------------------------------------------------
// Stats row — follower/following counts and follow button
// ---------------------------------------------------------------------------

class _StatsRow extends StatelessWidget {
  final Actor actor;
  final bool isOwn;
  const _StatsRow({required this.actor, required this.isOwn});

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          _StatColumn(label: l.followers, count: actor.followerCount ?? 0),
          const SizedBox(width: 24),
          _StatColumn(label: l.following, count: actor.followingCount ?? 0),
          const Spacer(),
          if (!isOwn)
            SizedBox(
              width: 100,
              child: _FollowButton(profileActorId: actor.id),
            ),
        ],
      ),
    );
  }
}

class _StatColumn extends StatelessWidget {
  final String label;
  final int count;
  const _StatColumn({required this.label, required this.count});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(fontSize: 13, color: Colors.grey[600])),
        Text(
          '$count',
          style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w700),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Follow button — drives followProvider for remote profiles
// ---------------------------------------------------------------------------

class _FollowButton extends ConsumerWidget {
  final String profileActorId;

  const _FollowButton({required this.profileActorId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final followAsync = ref.watch(followProvider(profileActorId));

    return followAsync.when(
      data: (followState) {
        return Padding(
          padding: const EdgeInsets.only(right: 8),
          child: WandererButton(
            primary: !followState.isFollowing,
            secondary: followState.isFollowing,
            loading: followState.isLoading,
            onPressed: followState.isLoading
                ? () {}
                : () => ref
                      .read(followProvider(profileActorId).notifier)
                      .toggle(),
            child: Text(
              followState.isFollowing
                  ? AppLocalizations.of(context)!.following
                  : AppLocalizations.of(context)!.follow,
            ),
          ),
        );
      },
      loading: () => const Padding(
        padding: EdgeInsets.only(right: 12),
        child: SizedBox(
          width: 20,
          height: 20,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
      ),
      error: (err, _) => const SizedBox.shrink(),
    );
  }
}
