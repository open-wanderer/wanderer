import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/trail_dropdown.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/navigation_launch_util.dart';

class TrailDetailScreen extends ConsumerStatefulWidget {
  final String id;
  const TrailDetailScreen({super.key, required this.id});

  @override
  ConsumerState<TrailDetailScreen> createState() => _TrailDetailScreenState();
}

class _TrailDetailScreenState extends ConsumerState<TrailDetailScreen> {
  bool _isLaunching = false;
  late final ScrollController _scrollController = ScrollController();

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final trailAsync = ref.watch(trailProvider(widget.id));

    return Scaffold(
      body: SafeArea(
        child: trailAsync.when(
          data: (trail) => Stack(
            children: [
              // Scrollable trail panel with bottom padding so it doesn't hide
              // behind the fixed Navigate button (Pitfall 4)
              Positioned.fill(
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 72),
                  child: TrailPanel(
                    trail: trail,
                    scrollController: _scrollController,
                    actionMenu: TrailDropdown(trail: trail),
                  ),
                ),
              ),

              // Fixed full-width Navigate button pinned to the bottom (D-01, D-03)
              Positioned(
                left: 0,
                right: 0,
                bottom: 0,
                child: Container(
                  color: Theme.of(context).scaffoldBackgroundColor,
                  padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                  child: SizedBox(
                    width: double.infinity,
                    child: ElevatedButton.icon(
                      onPressed: _isLaunching
                          ? null
                          : () async {
                              setState(() => _isLaunching = true);
                              await launchNavigation(
                                context: context,
                                ref: ref,
                                trail: trail,
                              );
                              if (mounted) setState(() => _isLaunching = false);
                            },
                      icon: _isLaunching
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: Colors.white,
                              ),
                            )
                          : const FaIcon(FontAwesomeIcons.locationArrow),
                      label: Text(AppLocalizations.of(context)!.navigate),
                    ),
                  ),
                ),
              ),
            ],
          ),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => WandererError(err: err, stack: stack),
        ),
      ),
    );
  }
}
