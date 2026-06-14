import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
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

  double _appBarOpacity = 0.0;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.hasClients) {
      const maxScrollDistance = 128.0;

      final double currentOffset = _scrollController.offset;
      final double newOpacity = (currentOffset / maxScrollDistance).clamp(
        0.0,
        1.0,
      );

      if (newOpacity != _appBarOpacity) {
        setState(() {
          _appBarOpacity = newOpacity;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final trailAsync = ref.watch(trailProvider(widget.id));
    final theme = Theme.of(context);

    return trailAsync.when(
      data: (trail) => Scaffold(
        extendBodyBehindAppBar: true,
        appBar: AppBar(
          leading: IconButton(
            icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
            onPressed: () => context.pop(),
            style: IconButton.styleFrom(
              backgroundColor: theme.colorScheme.surface.withValues(
                alpha: 1.0 - _appBarOpacity,
              ),
            ),
          ),

          backgroundColor: theme.colorScheme.surface.withValues(
            alpha: _appBarOpacity,
          ),

          shadowColor: Colors.black.withValues(alpha: _appBarOpacity * 0.15),
          elevation: _appBarOpacity > 0 ? 2 : 0,

          scrolledUnderElevation: 0,

          actions: [TrailDropdown(trail: trail)],
        ),
        body: Stack(
          children: [
            Positioned.fill(
              child: Padding(
                padding: const EdgeInsets.only(bottom: 72),
                child: TrailPanel(
                  trail: trail,
                  scrollController: _scrollController, // Passes perfectly now
                ),
              ),
            ),

            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: Container(
                color: theme.scaffoldBackgroundColor,
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
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
      ),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, stack) => WandererError(err: err, stack: stack),
    );
  }
}
