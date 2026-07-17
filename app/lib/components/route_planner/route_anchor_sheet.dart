import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/route_planner/elevation_tab.dart';
import 'package:wanderer/components/route_planner/route_anchor_list_tab.dart';

/// The route planner's docked, tabbed [DraggableScrollableSheet] (PLANUI-01,
/// D-02/D-03) hosting the Route Anchors tab (Plan 04) and the Elevation tab
/// (Plan 03).
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
  final String travelProfile;

  const RouteAnchorSheet({super.key, required this.travelProfile});

  @override
  ConsumerState<RouteAnchorSheet> createState() => _RouteAnchorSheetState();
}

class _RouteAnchorSheetState extends ConsumerState<RouteAnchorSheet> {
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  @override
  void dispose() {
    _sheetController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return DraggableScrollableSheet(
      controller: _sheetController,
      initialChildSize: 0.14,
      minChildSize: 0.14, // no full-dismiss (D-03) — never 0.0
      maxChildSize: 0.6,
      snap: true,
      snapSizes: const [0.14, 0.6],
      builder: (context, scrollController) {
        return DefaultTabController(
          length: 2,
          child: Container(
            decoration: BoxDecoration(
              color: theme.canvasColor,
              borderRadius: const BorderRadius.vertical(
                top: Radius.circular(16),
              ),
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
                        (_sheetController.size -
                                details.delta.dy / screenHeight)
                            .clamp(0.14, 0.6);
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
                        dividerColor: Colors.transparent,
                        labelColor: theme.brightness == Brightness.dark
                            ? const Color(0xff3E435B)
                            : const Color(0xff242734),
                        indicatorColor: theme.brightness == Brightness.dark
                            ? const Color(0xff3E435B)
                            : const Color(0xff242734),
                        tabs: const [
                          Tab(
                            icon: FaIcon(FontAwesomeIcons.listOl, size: 16),
                            text: 'Route Anchors',
                          ),
                          Tab(
                            icon: FaIcon(FontAwesomeIcons.mountain, size: 16),
                            text: 'Elevation',
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: TabBarView(
                    children: [
                      // CRITICAL (Pitfall 1): the builder's scrollController
                      // is attached to ONLY this tab. ElevationTab below gets
                      // no shared controller — passing it into both would
                      // throw at runtime since TabBarView keeps both children
                      // built simultaneously.
                      RouteAnchorListTab(
                        travelProfile: widget.travelProfile,
                        scrollController: scrollController,
                      ),
                      ElevationTab(travelProfile: widget.travelProfile),
                    ],
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}
