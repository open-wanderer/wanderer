import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/util/trail/route_location.dart';

class ProfileTrailScreen extends ConsumerStatefulWidget {
  final String handle;
  const ProfileTrailScreen({super.key, required this.handle});

  @override
  ConsumerState<ProfileTrailScreen> createState() => _ProfileTrailScreenState();
}

class _ProfileTrailScreenState extends ConsumerState<ProfileTrailScreen> {
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
              onChanged: (q) => ref
                  .read(profileTrailsProvider(widget.handle).notifier)
                  .search(q),
            ),
          ),
          TrailQuickFilterBar(filterId: 'profile_trail_${widget.handle}'),
          Expanded(
            child: trailsAsync.when(
              // The default `false` is what let a dependency-driven reload
              // (e.g. the filter provider settling after a retry) replace an
              // already-rendered list with a full-screen
              // CircularProgressIndicator. This is the only `.when` in this
              // file.
              skipLoadingOnReload: true,
              data: (state) {
                // Standing condition, not a one-off event -- a persistent
                // banner, not a toast.
                final showOfflineBanner = state.offline && state.isOwnHandle;

                return Column(
                  children: [
                    // Because a reload is now invisible (skipLoadingOnReload
                    // above), this thin indicator keeps it perceptible. Only
                    // rendered here (the "isLoading && hasValue" half) --
                    // with skipLoadingOnReload: true, this
                    // `data:` branch only runs when a value exists, so the
                    // hasValue half is implied.
                    if (trailsAsync.isLoading)
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
                            ? (state.isOwnHandle
                                  ? _OwnTrailsEmptyState(
                                      icon: FontAwesomeIcons.cloudArrowUp,
                                      title: l10n.own_trails_empty_title,
                                      body: l10n.own_trails_empty_body,
                                    )
                                  : _EmptyState())
                            : ListView.builder(
                                controller: _scrollController,
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
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (err, stack) => WandererError(err: err, stack: stack),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Text(
        AppLocalizations.of(context)!.no_trails_yet,
        style: TextStyle(color: Colors.grey[600]),
      ),
    );
  }
}

/// Own-trails offline/empty-library placeholder.
///
/// Duplicates `library_screen.dart`'s `_LibraryEmptyState.icon` structure
/// exactly (LayoutBuilder/ConstrainedBox/SingleChildScrollView, 24px
/// horizontal padding) -- the UI-SPEC requires reusing that treatment tier
/// rather than inventing a third empty-state visual language.
class _OwnTrailsEmptyState extends StatelessWidget {
  final FaIconData icon;
  final String title;
  final String body;

  const _OwnTrailsEmptyState({
    required this.icon,
    required this.title,
    required this.body,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // Centred in the space the trail list would have filled, but still
    // scrollable, matching library_screen.dart's own rationale.
    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        child: ConstrainedBox(
          constraints: BoxConstraints(
            minHeight: math.max(0, constraints.maxHeight),
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                FaIcon(
                  icon,
                  size: 64,
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
                ),
                const SizedBox(height: 16),
                Text(
                  title,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.titleMedium!.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  body,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium!.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
