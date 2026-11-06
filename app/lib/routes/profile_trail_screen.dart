import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_empty_state.dart';
import 'package:wanderer/components/base/wanderer_no_results.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/trail/route_location.dart';

class ProfileTrailScreen extends ConsumerStatefulWidget {
  final String handle;
  const ProfileTrailScreen({super.key, required this.handle});

  @override
  ConsumerState<ProfileTrailScreen> createState() => _ProfileTrailScreenState();
}

class _ProfileTrailScreenState extends ConsumerState<ProfileTrailScreen> {
  late final ScrollController _scrollController;
  String _query = '';

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
      final state = ref.read(profileTrailsProvider(widget.handle));
      if (state.value?.hasMore == true && !state.isLoading) {
        ref.read(profileTrailsProvider(widget.handle).notifier).loadNextPage();
      }
    }
  }

  void _onTrailSelect(BuildContext context, TrailSummary trail) {
    // An unsynced trail is now addressed through `/trail/local/<localId>`
    // so it opens the ordinary detail screen like every other trail and the
    // hiker chooses Edit from there -- the old divert to
    // `/trail/create/edit` was a workaround for the model id being blanked,
    // and the model id is still blank; the local id is what carries
    // identity now.
    final location = trailDetailLocation(trail);
    if (location != null) {
      context.push(location);
      return;
    }
    // Unaddressable: an unsynced row with no localId. Impossible for a
    // row written by `saveNewLocalTrail`, but representable in the
    // model -- fall back to the offline-capable edit screen, which
    // takes the Trail itself as `extra` and needs no id at all, rather
    // than pushing a path that go_router will canonicalize into a
    // no-route error page.
    if (trail is Trail) {
      context.push('/trail/create/edit', extra: trail);
    }
  }

  @override
  Widget build(BuildContext context) {
    final trailsAsync = ref.watch(profileTrailsProvider(widget.handle));
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        scrolledUnderElevation: 0,
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(l10n.trail(2)),
        actions: [
          IconButton(
            icon: const FaIcon(FontAwesomeIcons.mapLocationDot, size: 18),
            tooltip: l10n.map,
            onPressed: () =>
                context.push('/profile/${widget.handle}/trails/map'),
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: WandererSearchBar(
              hintText: l10n.search,
              onChanged: (q) {
                setState(() => _query = q.trim());
                ref
                    .read(profileTrailsProvider(widget.handle).notifier)
                    .search(q);
              },
            ),
          ),
          TrailQuickFilterBar(filterId: 'profile_trail_${widget.handle}'),
          Expanded(
            child: AsyncLoader<ProfileTrailsState>(
              asyncValue: trailsAsync,
              mockData: ProfileTrailsState.mock(),
              // Without this a dependency-driven reload (e.g. the filter
              // provider settling after a retry) would replace an
              // already-rendered list with the skeleton.
              keepValueWhileLoading: true,
              builder: (state) {
                // Standing condition, not a one-off event -- a persistent
                // banner, not a toast.
                final showOfflineBanner = state.offline && state.isOwnHandle;

                return Column(
                  children: [
                    // A reload keeps the list on screen (see
                    // keepValueWhileLoading above), so this thin indicator
                    // keeps it perceptible. The first load has the skeleton.
                    if (trailsAsync.isLoading && trailsAsync.hasValue)
                      const LinearProgressIndicator(minHeight: 2),
                    if (showOfflineBanner)
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.symmetric(
                          horizontal: 20,
                          vertical: 12,
                        ),
                        color: theme.colorScheme.surfaceContainerHighest,
                        child: Row(
                          children: [
                            Icon(
                              Icons.cloud_off,
                              size: 14,
                              color: theme.colorScheme.onSurface.withValues(
                                alpha: 0.6,
                              ),
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                l10n.own_trails_offline_banner,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurface.withValues(
                                    alpha: 0.6,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    Expanded(
                      child: RefreshIndicator(
                        color: Theme.of(context).colorScheme.onSurface,
                        onRefresh: () async => ref.invalidate(
                          profileTrailsProvider(widget.handle),
                        ),
                        child: state.trails.isEmpty
                            ? (_query.isNotEmpty
                                  ? WandererNoResults(query: _query)
                                  : state.isOwnHandle
                                  ? WandererEmptyState(
                                      icon: FontAwesomeIcons.route,
                                      title: l10n.own_trails_empty_title,
                                      body: l10n.own_trails_empty_body,
                                      actionLabel: l10n.new_trail,
                                      onAction: () => context.pushReplacement(
                                        '/trail/create',
                                      ),
                                    )
                                  : WandererEmptyState(
                                      icon: FontAwesomeIcons.route,
                                      title:
                                          l10n.profile_trails_empty_other_title,
                                      body: l10n
                                          .profile_trails_empty_other_body(
                                            usernameFromHandle(widget.handle),
                                          ),
                                    ))
                            : ListView.builder(
                                // Not on the skeleton: it is on screen
                                // alongside the real list during the
                                // cross-fade.
                                controller: trailsAsync.hasValue
                                    ? _scrollController
                                    : null,
                                padding: const EdgeInsets.fromLTRB(
                                  16,
                                  0,
                                  16,
                                  16,
                                ),
                                itemCount:
                                    state.trails.length +
                                    (state.hasMore ? 1 : 0),
                                itemBuilder: (context, index) {
                                  if (index == state.trails.length) {
                                    return const Padding(
                                      padding: EdgeInsets.symmetric(
                                        vertical: 16,
                                      ),
                                      child: Center(
                                        child: CircularProgressIndicator(),
                                      ),
                                    );
                                  }
                                  final trail = state.trails[index];
                                  return TrailListItem(
                                    key: ValueKey(trail.localId ?? trail.id),
                                    trail: trail,
                                    onTrailSelect: () =>
                                        _onTrailSelect(context, trail),
                                  );
                                },
                              ),
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
