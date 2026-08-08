import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/util/trail/filter.dart';
import 'package:wanderer/util/trail/route_location.dart';
import 'dart:math' as math;

class LibraryScreen extends ConsumerStatefulWidget {
  const LibraryScreen({super.key});

  @override
  ConsumerState<LibraryScreen> createState() => _LibraryScreenState();
}

class _LibraryScreenState extends ConsumerState<LibraryScreen> {
  final _searchController = TextEditingController();
  String _query = '';

  /// Debounces the search field: [_filtered] runs a lowercase scan + full
  /// filter/sort over the whole library, and doing that per keystroke made
  /// fast typing pay the entire pipeline once per character. The text field
  /// itself stays live (it owns its controller) — only the list refresh
  /// waits for a typing pause.
  Timer? _searchDebounce;

  void _onQueryChanged(String value) {
    _searchDebounce?.cancel();
    _searchDebounce = Timer(const Duration(milliseconds: 250), () {
      if (mounted) setState(() => _query = value);
    });
  }

  @override
  void dispose() {
    _searchDebounce?.cancel();
    _searchController.dispose();
    super.dispose();
  }

  List<Trail> _filtered(List<Trail> trails, TrailFilter? filter) {
    final q = _query.trim().toLowerCase();
    List<Trail> result = q.isEmpty
        ? trails
        : trails.where((t) {
            return t.name.toLowerCase().contains(q) ||
                t.description.toLowerCase().contains(q);
          }).toList();

    if (filter != null) {
      result = applyTrailFilter(result, filter);
    }

    return result;
  }

  @override
  Widget build(BuildContext context) {
    final trailLibrary = ref.watch(trailLibraryProvider);
    final router = ref.watch(routerProvider);
    final filterAsync = ref.watch(trailFilterProvider('library'));
    final visible = _filtered(trailLibrary, filterAsync.value);
    final l10n = AppLocalizations.of(context)!;

    return SafeArea(
      bottom: false,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0),
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8.0),
              child: TextField(
                controller: _searchController,
                onChanged: _onQueryChanged,
                cursorColor: Theme.of(context).colorScheme.onSurface,
                decoration: InputDecoration(
                  hintText: l10n.search_library,
                  hintStyle: TextStyle(color: Colors.grey),
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: _query.isNotEmpty
                      ? IconButton(
                          icon: const Icon(Icons.clear),
                          onPressed: () {
                            _searchDebounce?.cancel();
                            _searchController.clear();
                            setState(() => _query = '');
                          },
                        )
                      : null,
                  enabledBorder: OutlineInputBorder(
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.outline,
                    ),
                    borderRadius: BorderRadius.circular(56),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(56),
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.onSurface,
                      width: 1,
                    ),
                  ),
                  isDense: true,
                ),
              ),
            ),
            const TrailQuickFilterBar(filterId: 'library'),
            Expanded(
              child: visible.isEmpty
                  // An empty library and an over-narrow search/filter are
                  // different problems, so they get different artwork and a
                  // body line that names the actual way out.
                  ? trailLibrary.isEmpty
                        ? _LibraryEmptyState.icon(
                            icon: FontAwesomeIcons.cloudArrowDown,
                            title: l10n.library_empty_title,
                            body: l10n.library_empty_body,
                          )
                        : _LibraryEmptyState.artwork(
                            asset: 'search',
                            title: l10n.no_trails_found,
                            body: l10n.library_empty_search_body,
                          )
                  : ListView.builder(
                      itemCount: visible.length,
                      itemBuilder: (context, i) {
                        final trail = visible[i];
                        final location = trailDetailLocation(trail);
                        return TrailCard(
                          // Key so a filter/sort reorder MOVES elements
                          // instead of rebuilding every card's subtree from
                          // scratch. localId disambiguates unsynced rows,
                          // whose server id is blanked.
                          key: ValueKey(trail.localId ?? trail.id),
                          trail: trail,
                          onTrailSelect: location == null
                              ? null
                              : () => router.push(location),
                          onLongPress: () =>
                              _showContextMenu(context, trail, router),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }

  void _showContextMenu(BuildContext context, Trail trail, GoRouter router) {
    final l18n = AppLocalizations.of(context)!;
    final location = trailDetailLocation(trail);

    // Resolved once, up front -- this is an event
    // handler (the sheet is about to open), so `ref.read`, not `ref.watch`.
    // Every row this screen renders comes from `trailLibraryProvider`,
    // which filters only on `savedByUserIds`, so this sheet fires
    // unconditionally for a row that could still be the hiker's own
    // pending recording -- while `_confirmRemoveDownload`'s copy
    // (`remove_download_confirm_body`) promises "the trail itself is not
    // deleted". The store guard makes that promise true; this guard keeps
    // the action from being offered where it is meaningless in the first
    // place. `ownLiveCaptureProvider` is the exact same predicate
    // `trail_dropdown.dart` reads -- one predicate, two surfaces -- so both
    // behave identically.
    final isOwnLiveCapture = ref.read(ownLiveCaptureProvider(trail.localId));

    showModalBottomSheet(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const FaIcon(
                FontAwesomeIcons.arrowUpRightFromSquare,
                size: 18,
              ),
              title: Text(l18n.open),
              enabled: location != null,
              onTap: location == null
                  ? null
                  : () {
                      Navigator.of(ctx).pop();
                      router.push(location);
                    },
            ),
            // Hidden entirely (not disabled) when this account owns
            // the row as a live, not-yet-uploaded capture -- there is
            // nothing meaningful to un-download, and unlike Edit, waiting
            // for connectivity never makes it available.
            if (!isOwnLiveCapture)
              ListTile(
                // Not a delete -- this only drops this account's library
                // membership. No red colour: removing a download is not
                // destructive the way a server delete is. The dropdown menu
                // uses the same icon for the same action, so the two
                // surfaces stay consistent.
                leading: const FaIcon(FontAwesomeIcons.circleMinus, size: 18),
                title: Text(l18n.remove),
                onTap: () {
                  Navigator.of(ctx).pop();
                  _confirmRemoveDownload(context, trail);
                },
              ),
          ],
        ),
      ),
    );
  }

  /// Mirrors `settings_offline_regions_screen.dart`'s
  /// `_onDeleteRegion` -- one dialog, no connectivity branching. The body
  /// (`remove_download_confirm_body`) states the trail itself is not
  /// deleted -- unlike the server-delete confirm's false "cannot be
  /// undone" claim, which this action must never reuse.
  Future<void> _confirmRemoveDownload(BuildContext context, Trail trail) async {
    final l18n = AppLocalizations.of(context)!;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(trail.name),
        content: Text(l18n.remove_download_confirm_body),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l18n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l18n.remove, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (confirmed != true) return;
    if (!context.mounted) return;

    // Scopes membership only -- never a server delete. `deleteTrail`
    // drops this account's `savedByUserIds`
    // entry and removes on-device files only once the last account
    // sharing the row gives it up.
    //
    // Awaited: discarding this future dropped any error into
    // the zone as an unhandled async error instead of surfacing it, and
    // skipped `deleteTrail`'s own `state` update.
    await ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
  }
}

/// Placeholder shown where the trail list would be.
///
/// Two flavours because the two cases have different vocabularies. The
/// no-search-results case reuses `empty_state_search_*.svg`, which already
/// means exactly that in `global_search_screen`, `location_search_screen` and
/// `map_screen`. The empty-library case has no matching artwork — the
/// `empty_state_trail_*.svg` pair is the missing-photo placeholder, not an
/// empty state — so it falls back to a Font Awesome glyph styled like
/// `settings_offline_regions_screen.dart`'s `_buildEmptyState`, the closest
/// sibling in the downloads domain.
class _LibraryEmptyState extends StatelessWidget {
  /// Base name of the `empty_state_<asset>_<brightness>.svg` pair to draw.
  final String? asset;
  final FaIconData? icon;
  final String title;
  final String body;

  const _LibraryEmptyState.artwork({
    required String this.asset,
    required this.title,
    required this.body,
  }) : icon = null;

  const _LibraryEmptyState.icon({
    required FaIconData this.icon,
    required this.title,
    required this.body,
  }) : asset = null;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    // Centred in the space the trail list would have filled, but still
    // scrollable: `minHeight` lets the Center do its job at full height, and
    // the scroll view only starts scrolling once the viewport is shorter than
    // the content — a short screen with the keyboard up.
    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        child: ConstrainedBox(
          constraints: BoxConstraints(
            minHeight: math.max(
              0,
              constraints.maxHeight - kBottomNavigationBarHeight,
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24.0),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                if (asset != null)
                  SvgPicture.asset(
                    "assets/svgs/empty_state_${asset}_${theme.brightness.name}.svg",
                    // The localized title, not a fixed English string: this
                    // widget serves both flavours, so a hardcoded "library
                    // empty" label would misdescribe the no-search-results
                    // one to a screen reader.
                    semanticsLabel: title,
                    height: 120,
                  )
                else
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
