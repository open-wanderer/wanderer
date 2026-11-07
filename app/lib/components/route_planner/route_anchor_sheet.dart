import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/route_planner/elevation_tab.dart';
import 'package:wanderer/components/route_planner/route_anchor_list_tab.dart';
import 'package:wanderer/components/route_planner/settings_tab.dart';

/// The route planner's docked, tabbed [DraggableScrollableSheet], hosting
/// the Route Anchors, Elevation, and Settings tabs.
///
/// `TabBarView` keeps all tab pages built/mounted simultaneously (it
/// preloads the adjacent page for the swipe animation), so passing the same
/// `scrollController` to more than one tab's scrollable throws
/// `"ScrollController attached to multiple scroll views"` at runtime
/// (flutter/flutter#55388). The fix: the sheet's builder-supplied
/// `scrollController` is wired to ONLY [RouteAnchorListTab] — the one
/// genuinely-scrollable tab. Expand/collapse is driven by a dedicated
/// [DraggableScrollableController] via a manual
/// `GestureDetector.onVerticalDragUpdate` on the handle-bar row, decoupled
/// from the scrollController plumbing.
///
/// The sheet never fully dismisses — `minChildSize` is the non-zero peek
/// height, and there is no `onClose`/drag-to-dismiss path.
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

    final sheetMinSize = 164 / MediaQuery.of(context).size.height;
    final sheetMaxSize = 0.6;

    return DraggableScrollableSheet(
      controller: _sheetController,
      initialChildSize: sheetMinSize,
      minChildSize: sheetMinSize, // never 0.0 — no full-dismiss
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
              // Handle-bar + TabBar header is the only expand/collapse driver:
              // a vertical drag anywhere here resizes the sheet, while a
              // stationary tap still resolves to TabBar's tap recognizer via
              // the gesture arena.
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
              Expanded(
                child: TabBarView(
                  controller: _tabController,
                  children: [
                    // The builder's scrollController is attached to ONLY this
                    // tab — passing it to more than one throws at runtime
                    // since TabBarView keeps every child built simultaneously.
                    RouteAnchorListTab(
                      scrollController: _activeIndex == 0
                          ? scrollController
                          : null,
                    ),
                    ElevationTab(
                      scrollController: _activeIndex == 1
                          ? scrollController
                          : null,
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
