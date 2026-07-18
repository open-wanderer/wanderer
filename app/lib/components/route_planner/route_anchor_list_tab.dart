import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/format_util.dart';

/// The sheet's "Route Anchors" tab: a [ReorderableListView] listing every
/// in-progress route anchor in order, each row showing a numbered badge,
/// its coordinates, and a trailing delete button.
///
/// Uses an optimistic working-copy reorder pattern: a local [_orderedIds]
/// list seeded from the provider on every build, guarded by [_reordering]
/// so a mid-drag provider re-emission never snaps the dragged row back.
/// `reorderAnchors` is a synchronous in-memory mutation, so there's no
/// revert-on-error path.
///
/// Delete is immediate — no confirmation dialog, no snackbar-undo; the
/// app-bar Undo/Redo is the sole safety net. The delete button is never
/// disabled — deleting to zero anchors is valid, and the parent screen
/// un-mounts the sheet once `anchors` is empty.
///
/// This is the only genuinely scrollable tab, so it's the one wired to the
/// sheet's builder-supplied [ScrollController].
class RouteAnchorListTab extends ConsumerStatefulWidget {
  final ScrollController? scrollController;

  const RouteAnchorListTab({super.key, this.scrollController});

  @override
  ConsumerState<RouteAnchorListTab> createState() => _RouteAnchorListTabState();
}

class _RouteAnchorListTabState extends ConsumerState<RouteAnchorListTab> {
  /// Working copy of the anchor id order, seeded from the provider on every
  /// build while [_reordering] is false. Never reseed mid-drag.
  List<String> _orderedIds = const [];

  /// True for the duration of a drag gesture. While true, `build()` must
  /// not reseed [_orderedIds] from the provider.
  bool _reordering = false;

  /// If every anchor shares one country, [_anchorTitle] omits it for a
  /// cleaner row title (mirrors web's `commonAnchorCountry`).
  String? _commonAnchorCountry(List<RouteAnchor> anchors) {
    final countries = anchors
        .map((a) => a.location?.country)
        .whereType<String>()
        .where((c) => c.isNotEmpty)
        .toList();

    if (countries.length < 2) {
      return null;
    }

    final first = countries.first;
    return countries.every((c) => c == first) ? first : null;
  }

  String _anchorTitle(RouteAnchor anchor, int index, String? commonCountry) {
    final location = anchor.location;
    if (location == null) {
      return 'Anchor ${index + 1}';
    }

    return (commonCountry != null && location.country == commonCountry)
        ? location.label
        : location.fullLabel;
  }

  /// Formats the cumulative distance/duration/elevation-gain "so far" line
  /// shown under each anchor after the first (whose stats are always zero).
  String _statsLabel(AnchorRouteStats stats, String unit) {
    final duration = Duration(seconds: stats.durationSeconds.round());
    return '${formatDistance(stats.distanceMeters, unit: unit)} · '
        '${duration.pretty(abbreviated: true, tersity: DurationTersity.minute)} · '
        '↑ ${formatElevation(stats.elevationGainMeters, unit: unit)}';
  }

  @override
  Widget build(BuildContext context) {
    final anchors = ref.watch(routeAnchorsProvider).anchors;
    final cumulativeStats = ref.watch(
      routeAnchorsProvider.select((s) => s.cumulativeStatsByAnchorId),
    );
    final unit = ref.watch(unitProvider);

    if (!_reordering) {
      _orderedIds = anchors.map((a) => a.id).toList();
    }

    final commonCountry = _commonAnchorCountry(anchors);

    final byId = {for (final a in anchors) a.id: a};
    final theme = Theme.of(context);
    final accent = theme.brightness == Brightness.dark
        ? const Color(0xff3E435B)
        : const Color(0xff242734);

    // The chip row is a fixed header; the ReorderableListView below is the
    // sole owner of widget.scrollController.
    return Column(
      children: [
        _AnchorActionChips(anchors: anchors),
        Expanded(
          child: ReorderableListView.builder(
            scrollController: widget.scrollController,
            buildDefaultDragHandles: true,
            itemCount: _orderedIds.length,
            itemBuilder: (context, index) {
              final anchor = byId[_orderedIds[index]];
              if (anchor == null) {
                return SizedBox.shrink(key: ValueKey(_orderedIds[index]));
              }

              return ListTile(
                key: ValueKey(anchor.id),
                leading: Container(
                  width: 24,
                  height: 24,
                  decoration: BoxDecoration(
                    color: accent,
                    shape: BoxShape.circle,
                  ),
                  child: Center(
                    child: Text(
                      '${index + 1}',
                      style: const TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: Colors.white,
                      ),
                    ),
                  ),
                ),
                title: Text(
                  _anchorTitle(anchor, index, commonCountry),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      '${anchor.lat.toStringAsFixed(5)}, ${anchor.lon.toStringAsFixed(5)}',
                      style: TextStyle(color: Colors.grey[600]),
                    ),
                    if (index > 0 && cumulativeStats[anchor.id] != null)
                      Text(
                        _statsLabel(cumulativeStats[anchor.id]!, unit),
                        style: TextStyle(
                          color: Colors.grey[600],
                          fontSize: 12,
                        ),
                      ),
                  ],
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      icon: FaIcon(
                        FontAwesomeIcons.personWalkingArrowLoopLeft,
                        size: 14,
                      ),
                      onPressed: () => ref
                          .read(routeAnchorsProvider.notifier)
                          .appendAnchor(
                            Geographic(lon: anchor.lon, lat: anchor.lat),
                          ),
                      visualDensity: VisualDensity.compact,
                    ),
                    IconButton(
                      icon: FaIcon(
                        FontAwesomeIcons.trash,
                        size: 14,
                        color: Colors.red.shade400,
                      ),
                      onPressed: () => ref
                          .read(routeAnchorsProvider.notifier)
                          .deleteAnchor(anchor.id),
                      visualDensity: VisualDensity.compact,
                    ),
                  ],
                ),
              );
            },
            onReorderStart: (_) => setState(() => _reordering = true),
            // ignore: deprecated_member_use
            onReorder: _onReorder,
          ),
        ),
      ],
    );
  }

  void _onReorder(int oldIndex, int newIndex) {
    if (newIndex > oldIndex) newIndex -= 1;

    final reordered = List<String>.from(_orderedIds);
    final moved = reordered.removeAt(oldIndex);
    reordered.insert(newIndex, moved);
    setState(() => _orderedIds = reordered);

    ref.read(routeAnchorsProvider.notifier).reorderAnchors(_orderedIds);

    setState(() => _reordering = false);
  }
}

/// "Delete all" (immediate, no confirmation — Undo is the sole safety net)
/// and "Reverse direction" chips, shown only on the Route Anchors tab. Both
/// disable themselves rather than no-op silently when there's nothing to act on.
class _AnchorActionChips extends ConsumerWidget {
  final List<RouteAnchor> anchors;

  const _AnchorActionChips({required this.anchors});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final notifier = ref.read(routeAnchorsProvider.notifier);

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      child: Wrap(
        spacing: 8,
        children: [
          ActionChip(
            visualDensity: VisualDensity.compact,
            avatar: const FaIcon(
              FontAwesomeIcons.arrowRightArrowLeft,
              size: 12,
            ),
            label: const Text('Reverse direction'),
            onPressed: anchors.length < 2 ? null : notifier.reverseRoute,
          ),
          ActionChip(
            visualDensity: VisualDensity.compact,
            avatar: FaIcon(
              FontAwesomeIcons.trash,
              size: 12,
              color: anchors.isEmpty
                  ? null
                  : theme.colorScheme.onErrorContainer,
            ),
            label: const Text('Delete all'),
            backgroundColor: anchors.isEmpty
                ? null
                : theme.colorScheme.errorContainer,
            labelStyle: anchors.isEmpty
                ? null
                : TextStyle(color: theme.colorScheme.onErrorContainer),
            onPressed: anchors.isEmpty ? null : notifier.deleteAllAnchors,
          ),
        ],
      ),
    );
  }
}
