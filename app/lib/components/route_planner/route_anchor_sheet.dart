import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/route_planner/elevation_tab.dart';
import 'package:wanderer/components/route_planner/route_anchor_list_tab.dart';
import 'package:wanderer/components/route_planner/settings_tab.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';

/// The route planner's docked, tabbed [DraggableScrollableSheet] (PLANUI-01,
/// D-02/D-03) hosting the Route Anchors tab (Plan 04), the Elevation tab
/// (Plan 03), and the Settings tab (quick-260717-t7q).
///
/// This composition is the phase's hardest problem (RESEARCH.md Pattern 1):
/// `TabBarView` keeps both tab pages built/mounted simultaneously (it
/// preloads the adjacent page for the swipe animation), so passing the same
/// `scrollController` to more than one tab's scrollable throws
/// `"ScrollController attached to multiple scroll views"` at runtime
/// (flutter/flutter#55388). The fix: the sheet's builder-supplied
/// `scrollController` is wired to ONLY [RouteAnchorListTab] — the one
/// genuinely-scrollable tab — while [ElevationTab] receives no shared
/// controller at all. Expand/collapse is driven by a dedicated
/// [DraggableScrollableController] (already an established pattern via
/// `WaypointSheet`) attached to a manual `GestureDetector.onVerticalDragUpdate`
/// on the handle-bar row, entirely decoupled from the scrollController
/// plumbing.
///
/// The sheet never fully dismisses (D-03) — `minChildSize` is the non-zero
/// peek height, and there is no `onClose`/drag-to-dismiss path (unlike
/// `WaypointSheet`, which this chrome is otherwise modeled on).
class RouteAnchorSheet extends ConsumerStatefulWidget {
  const RouteAnchorSheet({super.key});

  @override
  ConsumerState<RouteAnchorSheet> createState() => _RouteAnchorSheetState();
}

class _RouteAnchorSheetState extends ConsumerState<RouteAnchorSheet>
    with SingleTickerProviderStateMixin {
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  late TabController _tabController;
  int _activeIndex = 0;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
    _tabController.addListener(() {
      if (_tabController.indexIsChanging) return;
      setState(() {
        _activeIndex = _tabController.index;
      });
    });
  }

  @override
  void dispose() {
    _sheetController.dispose();
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final sheetMinSize = 103 / MediaQuery.of(context).size.height;
    final sheetMaxSize = 0.6;

    return DraggableScrollableSheet(
      controller: _sheetController,
      initialChildSize: sheetMinSize,
      minChildSize: sheetMinSize, // no full-dismiss (D-03) — never 0.0
      maxChildSize: sheetMaxSize,
      snap: true,
      snapSizes: [sheetMinSize, sheetMaxSize],
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: theme.canvasColor,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.15),
                blurRadius: 10,
                offset: const Offset(0, -2),
              ),
            ],
          ),
          child: Column(
            children: [
              // Handle-bar + TabBar header — the ONLY expand/collapse
              // driver (Pattern 1). A vertical drag anywhere in this header
              // (not just the thin handle strip) resizes the sheet; a
              // stationary tap still resolves to TabBar's own tap
              // recognizer via the gesture arena, so tab switching is
              // unaffected. No close button (D-03: peek is the floor, sheet
              // never dismisses).
              GestureDetector(
                behavior: HitTestBehavior.opaque,
                onVerticalDragUpdate: (details) {
                  if (!_sheetController.isAttached) return;
                  final screenHeight = MediaQuery.sizeOf(context).height;
                  final target =
                      (_sheetController.size - details.delta.dy / screenHeight)
                          .clamp(sheetMinSize, sheetMaxSize);
                  _sheetController.jumpTo(target);
                },
                child: Column(
                  children: [
                    Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8.0),
                      child: Center(
                        child: Container(
                          width: 30,
                          height: 5,
                          decoration: BoxDecoration(
                            borderRadius: const BorderRadius.all(
                              Radius.circular(24),
                            ),
                            color: theme.colorScheme.secondaryContainer,
                          ),
                        ),
                      ),
                    ),
                    TabBar(
                      controller: _tabController,
                      onTap: (_) => {
                        if (_sheetController.size == sheetMinSize)
                          {
                            _sheetController.animateTo(
                              sheetMaxSize,
                              duration: Duration(milliseconds: 250),
                              curve: Curves.easeOut,
                            ),
                          },
                      },
                      dividerColor: Colors.transparent,
                      labelColor: theme.brightness == Brightness.dark
                          ? Colors.white
                          : Theme.of(context).colorScheme.primary,
                      indicatorColor: theme.brightness == Brightness.dark
                          ? Colors.white
                          : Theme.of(context).colorScheme.primary,
                      unselectedLabelColor: Theme.of(
                        context,
                      ).colorScheme.onSurface.withValues(alpha: 0.3),
                      tabs: const [
                        Tab(
                          icon: FaIcon(FontAwesomeIcons.listOl, size: 16),
                          text: 'Route Anchors',
                        ),
                        Tab(
                          icon: FaIcon(FontAwesomeIcons.mountain, size: 16),
                          text: 'Elevation',
                        ),
                        Tab(
                          icon: FaIcon(FontAwesomeIcons.gear, size: 16),
                          text: 'Settings',
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              _SheetActionChips(),
              Expanded(
                child: TabBarView(
                  controller: _tabController,
                  children: [
                    // CRITICAL (Pitfall 1): the builder's scrollController
                    // is attached to ONLY this tab. ElevationTab and
                    // SettingsTab below get no shared controller — passing
                    // it into more than one would throw at runtime since
                    // TabBarView keeps every child built simultaneously.
                    RouteAnchorListTab(
                      scrollController: _activeIndex == 0
                          ? scrollController
                          : null,
                    ),
                    ElevationTab(
                      scrollController: _activeIndex == 1
                          ? scrollController
                          : null,
                      isVisible: _activeIndex == 1,
                    ),
                    SettingsTab(
                      scrollController: _activeIndex == 2
                          ? scrollController
                          : null,
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

/// Two chip-sized actions at the top of the sheet, above the tab content
/// (visible regardless of the active tab): "Delete all" (immediate, no
/// confirmation — [RouteAnchors.deleteAllAnchors] mirrors [deleteAnchor]'s
/// own D-06 discipline; Undo is the sole safety net) and "Reverse
/// direction" ([RouteAnchors.reverseRoute]). Both disable themselves rather
/// than no-op silently when there's nothing to act on.
class _SheetActionChips extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final anchors = ref.watch(routeAnchorsProvider).anchors;
    final notifier = ref.read(routeAnchorsProvider.notifier);

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      child: Wrap(
        spacing: 8,
        children: [
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
          ActionChip(
            visualDensity: VisualDensity.compact,
            avatar: const FaIcon(
              FontAwesomeIcons.arrowRightArrowLeft,
              size: 12,
            ),
            label: const Text('Reverse direction'),
            onPressed: anchors.length < 2 ? null : notifier.reverseRoute,
          ),
        ],
      ),
    );
  }
}
