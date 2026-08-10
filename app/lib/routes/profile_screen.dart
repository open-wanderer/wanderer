import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/components/list/list_card.dart';
import 'package:wanderer/components/profile/feed_item_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/follow_provider.dart';
import 'package:wanderer/provider/profile/profile_counts_provider.dart';
import 'package:wanderer/provider/profile/profile_feed_provider.dart';
import 'package:wanderer/provider/profile/profile_local_trail_count_provider.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/util/format.dart';

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
      widget.handle ?? "@${ref.read(authProvider).value?.preferredUsername}";

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
    if (pos.extentAfter < kPagedPrefetchExtent) {
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

    // Own-profile offline resilience lives in `ownProfileProvider`, which
    // falls back to the actor cached on UserEntity — so the data branch below
    // renders the real profile layout offline and only the network-bound
    // sections (counts, lists, feed) degrade. The error branch is reached
    // solely when there is no cached actor at all (a pre-upgrade install that
    // has not completed an auth refresh yet); offline, that still needs to
    // keep the settings gear reachable.
    return Scaffold(
      body: actorAsync.when(
        data: (actor) => _buildProfile(actor),
        loading: () => const Center(child: CircularProgressIndicator()),
        error: isOwn ? _buildOwnProfileError : _buildRemoteProfileError,
      ),
    );
  }

  /// Own-profile error fallback for the no-cached-actor case. Offline it shows
  /// the shared offline state under an app bar that still carries the settings
  /// gear, so settings never become unreachable; online it is a plain error.
  Widget _buildOwnProfileError(Object err, StackTrace? stack) {
    if (ref.watch(onlineStatusProvider)) {
      return WandererError(err: err, stack: stack);
    }

    final l10n = AppLocalizations.of(context)!;
    return Column(
      children: [
        AppBar(
          actions: [
            IconButton(
              icon: const FaIcon(FontAwesomeIcons.gear, size: 16),
              onPressed: () => context.push('/settings'),
            ),
          ],
        ),
        Expanded(
          child: WandererOfflineState(
            title: l10n.offline_title,
            body: l10n.offline_profile_body,
            retryLabel: l10n.offline_try_again,
            onRetry: () async {
              final online = await ref
                  .read(onlineStatusProvider.notifier)
                  .refresh();
              if (online) ref.invalidate(ownProfileProvider);
            },
          ),
        ),
      ],
    );
  }

  /// The remote-profile (another user's) fetch-error fallback — there is no
  /// cached identity to fall back to for someone else's profile, so this
  /// stays a plain [WandererError], unlike the own-profile path above.
  Widget _buildRemoteProfileError(Object err, StackTrace? stack) =>
      WandererError(err: err, stack: stack);

  Widget _buildProfile(Actor actor) {
    final h = _handle;

    return RefreshIndicator(
      color: Theme.of(context).colorScheme.onSurface,
      onRefresh: () async {
        if (widget.handle != null) {
          ref.invalidate(profileProvider(widget.handle!));
        } else {
          ref.invalidate(ownProfileProvider);
        }
        if (h != null) {
          ref.invalidate(profileFeedProvider(h));
          ref.invalidate(profileListsProvider(h));
          // Recomputed from the store rather than fetched, so nothing else
          // pushes it: a capture saved since this screen was built only shows
          // up on an explicit refresh.
          ref.invalidate(profileLocalTrailCountProvider(h));
        }
      },
      child: CustomScrollView(
        controller: _scrollController,
        slivers: [
          SliverAppBar(
            pinned: true,
            expandedHeight: 200,
            surfaceTintColor: Theme.of(context).colorScheme.surface,
            flexibleSpace: FlexibleSpaceBar(
              titlePadding: EdgeInsets.fromLTRB(isOwn ? 16 : 72, 0, 0, 12),
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
                    child: Row(
                      children: [
                        ActorAvatar.fromActor(actor: actor, radius: 16),
                        const SizedBox(width: 16),
                        Column(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              actor.preferredUsername,
                              style: Theme.of(context).textTheme.bodyLarge!
                                  .copyWith(fontWeight: FontWeight.w600),
                            ),
                            Text(
                              '@${actor.preferredUsername}${(actor.domain?.isNotEmpty == true) ? '@${actor.domain}' : ''}',
                              style: Theme.of(context).textTheme.labelMedium!
                                  .copyWith(fontWeight: FontWeight.normal),
                            ),
                          ],
                        ),
                      ],
                    ),
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
            child: _StatsRow(actor: actor, isOwn: isOwn, handle: _handle),
          ),

          if (h != null)
            // Trail / list count cards
            SliverToBoxAdapter(
              child: _CountsRow(handle: h, actorId: actor.id),
            ),

          // Lists heading + preview
          if (h != null) SliverToBoxAdapter(child: _ListsPreview(handle: h)),

          // Feed
          if (h != null)
            SliverToBoxAdapter(
              child: _FeedSection(handle: h, actor: actor),
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
          onPressed: () => context.push('/profile/share'),
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
            ActorAvatar.fromActor(actor: actor, radius: 44),
            const SizedBox(height: 8),
            Text(
              actor.username,
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
            ),
            Text(
              handleDisplay,
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(
                  context,
                ).colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              '${AppLocalizations.of(context)!.joined} $joinedDate',
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(
                  context,
                ).colorScheme.onSurface.withValues(alpha: 0.6),
              ),
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
        child: Text(
          AppLocalizations.of(context)!.no_bio_yet,
          style: TextStyle(color: Colors.grey[600]),
        ),
      );
    }

    const bioMaxLength = 150;
    final preview = formatHtmlAsTextPreview(summary, bioMaxLength);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (preview.truncated && !_expanded)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                '${preview.text}…',
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            )
          else
            Html(data: summary),
          if (preview.truncated)
            TextButton(
              onPressed: () => setState(() => _expanded = !_expanded),
              child: Text(
                _expanded
                    ? AppLocalizations.of(context)!.show_less
                    : AppLocalizations.of(context)!.show_more,
              ),
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

    // Lists are network-only and already hide themselves when empty, so
    // offline they hide too rather than surfacing a raw error block.
    if (listsAsync.hasError && !ref.watch(onlineStatusProvider)) {
      return const SizedBox.shrink();
    }

    return AsyncLoader<ProfileListsState>(
      asyncValue: listsAsync,
      mockData: ProfileListsState.mock(),
      builder: (state) {
        final lists = state.lists;
        if (lists.isEmpty) return const SizedBox.shrink();
        return Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                AppLocalizations.of(context)!.list(2),
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                ),
              ),
              SizedBox(
                height: 170,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: lists.length,
                  separatorBuilder: (context, _) => const SizedBox(width: 8),
                  itemBuilder: (context, index) => ListCard(
                    list: lists[index],
                    mini: true,
                    onListSelect: () =>
                        context.push('/list/${lists[index].id}'),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

// ---------------------------------------------------------------------------
// Feed section — skeleton on initial load, list of FeedItemCards when ready
// ---------------------------------------------------------------------------

class _FeedSection extends ConsumerWidget {
  final String handle;
  final Actor actor;

  const _FeedSection({required this.handle, required this.actor});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final feedAsync = ref.watch(profileFeedProvider(handle));

    // The feed is network-only. Offline, keep the heading and say why it's
    // missing — silently dropping it makes the profile look empty rather than
    // degraded — but never surface the raw error block.
    if (feedAsync.hasError && !ref.watch(onlineStatusProvider)) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Text(
              AppLocalizations.of(context)!.feed,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: Text(
              AppLocalizations.of(context)!.offline_profile_body,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        ],
      );
    }

    return AsyncLoader<ProfileFeedState>(
      asyncValue: feedAsync,
      mockData: ProfileFeedState.mock(),
      builder: (state) {
        if (state.items.isEmpty) return const SizedBox.shrink();
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: Text(
                AppLocalizations.of(context)!.feed,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            ...state.items.map(
              (item) => FeedItemCard(item: item, profileActor: actor),
            ),
            // Gated on the fetch actually being in flight, never on `hasMore`
            // alone -- the spinner's ticker schedules a frame every vsync for
            // as long as it is mounted, and this sliver builds eagerly, so a
            // `hasMore` gate kept an idle screen rendering forever.
            //
            // Not `feedAsync.isLoading` either: `loadNextPage` deliberately
            // stays in AsyncData for the whole fetch, so that flag is false
            // throughout and the spinner would never appear at all.
            if (state.isLoadingMore)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 16),
                child: Center(child: CircularProgressIndicator()),
              ),
          ],
        );
      },
    );
  }
}

// ---------------------------------------------------------------------------
// Stats row — follower/following counts and follow button
// ---------------------------------------------------------------------------

class _StatsRow extends StatelessWidget {
  final Actor actor;
  final bool isOwn;
  final String? handle;
  const _StatsRow({
    required this.actor,
    required this.isOwn,
    required this.handle,
  });

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Expanded(
            child: _CountCard(
              icon: FontAwesomeIcons.userGroup,
              label: l.followers,
              count: actor.followerCount,
              onTap: handle != null
                  ? () => context.push('/profile/$handle/followers')
                  : null,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _CountCard(
              icon: FontAwesomeIcons.userCheck,
              label: l.following,
              count: actor.followingCount,
              onTap: handle != null
                  ? () => context.push('/profile/$handle/following')
                  : null,
            ),
          ),
          if (!isOwn) ...[
            const SizedBox(width: 12),
            SizedBox(
              width: 100,
              child: _FollowButton(profileActorId: actor.id),
            ),
          ],
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Counts row — trail count card + list count card
// ---------------------------------------------------------------------------

class _CountsRow extends ConsumerWidget {
  final String actorId;
  final String handle;
  const _CountsRow({required this.handle, required this.actorId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final counts = ref.watch(profileCountsProvider(actorId)).value;
    // Counts are network-only. Offline they never resolve, so surface a dash
    // instead of a skeleton that would shimmer indefinitely.
    final unavailable = !ref.watch(onlineStatusProvider) && counts == null;

    // The trail card is the one count this device can still answer while
    // offline: unsynced captures plus downloaded trails, i.e. exactly what
    // `/profile/<handle>/trails` renders offline. Null for another hiker's
    // handle, which keeps the dash. Labelled so the number is not mistaken
    // for the (larger) server-side total.
    final localTrailCount = unavailable
        ? ref.watch(profileLocalTrailCountProvider(handle))
        : null;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Row(
        children: [
          Expanded(
            child: _CountCard(
              icon: FontAwesomeIcons.route,
              label: localTrailCount != null
                  ? l10n.trails_on_device
                  : l10n.trail(2),
              count: localTrailCount ?? counts?.trailCount,
              unavailable: unavailable && localTrailCount == null,
              onTap: () => context.push('/profile/$handle/trails'),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _CountCard(
              icon: FontAwesomeIcons.layerGroup,
              label: l10n.list(2),
              count: counts?.listCount,
              unavailable: unavailable,
              onTap: () => context.push('/profile/$handle/lists'),
            ),
          ),
        ],
      ),
    );
  }
}

class _CountCard extends StatelessWidget {
  final FaIconData icon;
  final String label;
  final int? count;
  final VoidCallback? onTap;

  /// When true the count is known to be unobtainable (offline) rather than
  /// merely in flight, so render a placeholder instead of a skeleton.
  final bool unavailable;

  const _CountCard({
    required this.icon,
    required this.label,
    required this.count,
    required this.onTap,
    this.unavailable = false,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: colorScheme.outline),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Row(
            children: [
              FaIcon(icon, size: 16),
              const SizedBox(width: 10),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(
                        context,
                      ).colorScheme.onSurface.withValues(alpha: 0.6),
                    ),
                  ),
                  Skeletonizer(
                    enabled: count == null && !unavailable,
                    child: Text(
                      count == null && unavailable ? '—' : '$count',
                      style: const TextStyle(
                        fontSize: 20,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
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
      loading: () =>
          WandererButton(primary: true, disabled: true, loading: true),
      error: (err, _) => const SizedBox.shrink(),
    );
  }
}
