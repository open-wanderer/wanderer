import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/reverse_geocode_util.dart';

/// The sheet's "Route Anchors" tab (WAYP-04/05, D-05..D-08): a
/// [ReorderableListView] listing every in-progress route anchor in order,
/// each row showing a numbered badge, its coordinates, and a trailing
/// delete button.
///
/// Follows `settings_categories_screen.dart`'s proven optimistic
/// working-copy reorder pattern (D-08): a local [_orderedIds] list seeded
/// from the provider on every build, guarded by [_reordering] so a mid-drag
/// provider re-emission never snaps the dragged row back to its old slot.
/// Unlike that analog, `reorderAnchors` is a synchronous in-memory mutation
/// (no network, cannot throw) — there is no revert-on-error path here.
///
/// Delete is immediate (D-06): no confirmation dialog, no snackbar-undo.
/// Phase 19's app-bar Undo/Redo is the sole safety net. The delete button is
/// never disabled (D-07) — deleting to zero anchors is valid; the parent
/// screen un-mounts the sheet once `anchors` is empty.
///
/// Accepts the sheet's [ScrollController] directly (Pattern 1) — this is the
/// only genuinely scrollable tab, so the sheet's builder-supplied controller
/// is wired to THIS tab's [ReorderableListView], not the Elevation tab.
class RouteAnchorListTab extends ConsumerStatefulWidget {
  final ScrollController scrollController;

  const RouteAnchorListTab({super.key, required this.scrollController});

  @override
  ConsumerState<RouteAnchorListTab> createState() => _RouteAnchorListTabState();
}

class _RouteAnchorListTabState extends ConsumerState<RouteAnchorListTab> {
  /// Working copy of the anchor id order, seeded from the provider on every
  /// build while [_reordering] is false. Mutated optimistically during a
  /// drag (Pitfall 2 — never reseed from the provider mid-drag).
  List<String> _orderedIds = const [];

  /// True for the duration of a drag gesture (set in `onReorderStart`,
  /// cleared at the end of `_onReorder`). While true, `build()` must not
  /// reseed [_orderedIds] from the provider.
  bool _reordering = false;

  /// Resolved reverse-geocode results, keyed by rounded coordinate
  /// ([_locationCacheKey]). Mirrors web's `trail_anchor_list.svelte`
  /// module-level `locations` record — this is LOCAL widget state, not part
  /// of the immutable [RouteAnchor] model or a new provider (D-05/D-06
  /// analog, per this plan's action text).
  final Map<String, ReverseLocationResult> _locations = {};

  /// Keys with an in-flight reverse-geocode request, so a rapid re-trigger
  /// of the batch never double-fetches the same coordinate.
  final Set<String> _pending = {};

  /// The single shared cancel token for the current batch. Aborted before a
  /// new batch starts, mirroring web's single `locationAbortController`.
  CancelToken? _batchToken;

  /// Guards against re-kicking an identical batch on every build — only a
  /// genuine change to the anchors' coordinates/order re-triggers the
  /// sequential geocode batch.
  String? _lastAnchorSignature;

  String _locationCacheKey(RouteAnchor a) =>
      '${a.lat.toStringAsFixed(5)},${a.lon.toStringAsFixed(5)}';

  Future<void> _loadAnchorLocation(
    RouteAnchor anchor,
    CancelToken token,
  ) async {
    final key = _locationCacheKey(anchor);
    if (_locations.containsKey(key) || _pending.contains(key)) {
      return;
    }

    _pending.add(key);
    try {
      final result = await searchLocationReverseStructured(
        ref.read(apiProvider),
        anchor.lat,
        anchor.lon,
        includeRoad: true,
        cancelToken: token,
      );
      if (result != null && mounted) {
        setState(() => _locations[key] = result);
      }
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel) {
        // Superseded by a newer batch — silent, expected.
        return;
      }
      debugPrint('Failed to resolve anchor location: $e');
    } catch (e) {
      debugPrint('Failed to resolve anchor location: $e');
    } finally {
      _pending.remove(key);
    }
  }

  Future<void> _loadAnchorLocations(List<RouteAnchor> anchors) async {
    _batchToken?.cancel();
    final token = CancelToken();
    _batchToken = token;

    for (final anchor in anchors) {
      if (token.isCancelled) return;
      await _loadAnchorLocation(anchor, token);
    }
  }

  /// Optional cross-country nuance (mirror web's `commonAnchorCountry`): if
  /// every resolved anchor shares one country, [_anchorTitle] omits it for a
  /// cleaner, less redundant row title.
  String? _commonAnchorCountry(List<RouteAnchor> anchors) {
    final countries = anchors
        .map((a) => _locations[_locationCacheKey(a)]?.country)
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
    final location = _locations[_locationCacheKey(anchor)];
    if (location == null) {
      return 'Anchor ${index + 1}';
    }

    return (commonCountry != null && location.country == commonCountry)
        ? location.label
        : location.fullLabel;
  }

  @override
  void dispose() {
    _batchToken?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final anchors = ref.watch(routeAnchorsProvider).anchors;

    if (!_reordering) {
      _orderedIds = anchors.map((a) => a.id).toList();
    }

    final signature = anchors.map(_locationCacheKey).join('|');
    if (signature != _lastAnchorSignature) {
      _lastAnchorSignature = signature;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _loadAnchorLocations(anchors);
      });
    }

    final commonCountry = _commonAnchorCountry(anchors);

    final byId = {for (final a in anchors) a.id: a};
    final theme = Theme.of(context);
    final accent = theme.brightness == Brightness.dark
        ? const Color(0xff3E435B)
        : const Color(0xff242734);

    return ReorderableListView.builder(
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
            decoration: BoxDecoration(color: accent, shape: BoxShape.circle),
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
          subtitle: Text(
            '${anchor.lat.toStringAsFixed(5)}, ${anchor.lon.toStringAsFixed(5)}',
            style: TextStyle(color: Colors.grey[600]),
          ),
          trailing: IconButton(
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
        );
      },
      onReorderStart: (_) => setState(() => _reordering = true),
      // The index-shift `if (newIndex > oldIndex) newIndex -= 1` inside
      // _onReorder is the canonical onReorder contract (Pitfall 1);
      // onReorderItem is not used so that shift stays explicit and testable,
      // matching settings_categories_screen.dart's established pattern.
      // ignore: deprecated_member_use
      onReorder: _onReorder,
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
