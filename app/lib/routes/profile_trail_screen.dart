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
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';

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
    if (pos.pixels / pos.maxScrollExtent >= 0.8) {
      final state = ref.read(profileTrailsProvider(widget.handle));
      if (state.value?.hasMore == true && !state.isLoading) {
        ref.read(profileTrailsProvider(widget.handle).notifier).loadNextPage();
      }
    }
  }

  void _onTrailSelect(BuildContext context, TrailSummary trail) {
    // A locally-captured trail that hasn't uploaded yet has no server id to
    // push `/trail/${trail.id}` with -- route it to the offline-capable edit
    // screen instead (REC-05, D-16).
    if (trail is Trail && isUnsyncedState(trail.syncState)) {
      context.push('/trail/create/edit', extra: trail);
      return;
    }
    context.push('/trail/${trail.id}');
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
                // banner, not a toast (REC-06).
                final showOfflineBanner = state.offline && state.isOwnHandle;

                return Column(
                  children: [
                    // Because a reload is now invisible (skipLoadingOnReload
                    // above), this thin indicator keeps it perceptible. Only
                    // rendered here (the "isLoading && hasValue" half of the
                    // UAT's remedy) -- with skipLoadingOnReload: true, this
                    // `data:` branch only runs when a value exists, so the
                    // hasValue half is implied.
                    if (trailsAsync.isLoading)
                      const LinearProgressIndicator(minHeight: 2),
                    if (showOfflineBanner)
                      Container(
                        width: double.infinity,
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 12,
                        ),
                        color: theme.colorScheme.surfaceContainerHighest,
                        child: Row(
                          children: [
                            FaIcon(
                              FontAwesomeIcons.cloudArrowUp,
                              size: 14,
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                l10n.own_trails_offline_banner,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    Expanded(
                      child: RefreshIndicator(
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
