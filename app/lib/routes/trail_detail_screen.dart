import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/trail/like_button.dart';
import 'package:wanderer/components/trail/trail_dropdown.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/provider/trail/trail_download_state_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/actions/launch_navigation.dart';

class TrailDetailScreen extends ConsumerStatefulWidget {
  final String id;

  /// The local identity of a not-yet-uploaded trail. When set, [id] is empty
  /// (a local-sentinel id is blanked at the model boundary) and the screen
  /// reads its data from [localTrailProvider] instead of [trailProvider].
  final String? localId;

  const TrailDetailScreen({super.key, required this.id, this.localId});

  @override
  ConsumerState<TrailDetailScreen> createState() => _TrailDetailScreenState();
}

class _TrailDetailScreenState extends ConsumerState<TrailDetailScreen> {
  bool _isLaunching = false;
  late final ScrollController _scrollController = ScrollController();

  /// AppBar fade driven by scroll. A [ValueNotifier] consumed by a scoped
  /// builder around the AppBar alone — the old per-scroll-frame `setState`
  /// rebuilt the whole screen including `TrailPanel` (a full stats/photo
  /// subtree) on every scrolled pixel of the first 128.
  final ValueNotifier<double> _appBarOpacity = ValueNotifier<double>(0.0);

  /// Guards the retired-row redirect below: `addPostFrameCallback` inside
  /// `build()` re-schedules per rebuild, so without this the redirect was
  /// pushed once per frame until the route actually changed.
  bool _redirectScheduled = false;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    _appBarOpacity.dispose();
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

      _appBarOpacity.value = newOpacity;
    }
  }

  @override
  Widget build(BuildContext context) {
    final localId = widget.localId;
    if (localId != null) {
      final trail = ref.watch(localTrailProvider(localId));
      if (trail == null) {
        // The second half: the row is gone because the upload
        // succeeded, and the trail now lives at its server route --
        // `serverIdForRetired` is the memo `_drainOne` populated the instant
        // it retired this row. Redirect there instead of showing a
        // dead end when it recorded one.
        final retiredServerId = ref
            .read(trailSyncProvider.notifier)
            .serverIdForRetired(localId);
        if (retiredServerId != null && !_redirectScheduled) {
          _redirectScheduled = true;
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) context.pushReplacement('/trail/$retiredServerId');
          });
        }
        return Scaffold(
          appBar: AppBar(
            leading: IconButton(
              icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
              onPressed: () => context.pop(),
            ),
          ),
          body: Center(
            child: Text(AppLocalizations.of(context)!.trail_not_on_this_device),
          ),
        );
      }
      return _buildScreen(
        context,
        trail,
        panel: TrailPanel(
          trail: trail,
          scrollController: _scrollController,
          availableOffline: _availableOffline(trail),
        ),
      );
    }

    final trailAsync = ref.watch(trailProvider(widget.id));

    // Handled here rather than left to AsyncLoader: its `errorBuilder` is a
    // method, not a constructor param, and it renders `WandererError` with
    // neither a Scaffold nor the stack trace. Keep the error UX as it was.
    if (trailAsync.hasError && !trailAsync.hasValue) {
      return Scaffold(
        body: WandererError(
          err: trailAsync.error!,
          stack: trailAsync.stackTrace,
        ),
      );
    }

    // Only the panel goes through AsyncLoader. Skeletonizing the chrome too
    // turned every icon button into a filled circle bone -- three of them,
    // with the leading one butting into the date bone below -- and boned the
    // two bottom-bar buttons for actions that cannot be taken yet. The chrome
    // instead renders for real with its actions withheld until there is a
    // trail to act on, which also keeps the back button live during the fetch.
    //
    // `trailAsync.value` is null only on a first load; a refresh retains it,
    // so an optimistic like never flashes the mock.
    return _buildScreen(
      context,
      trailAsync.value,
      panel: AsyncLoader<Trail>(
        asyncValue: trailAsync,
        mockData: Trail.mock(),
        builder: (trail) => TrailPanel(
          trail: trail,
          scrollController: _scrollController,
          availableOffline: _availableOffline(trail),
        ),
      ),
    );
  }

  bool _availableOffline(Trail trail) => ref
      .watch(trailLibraryProvider)
      .any((t) => t.id.isNotEmpty && t.id == trail.id);

  /// The screen's chrome. [trail] is null while the first fetch is in flight —
  /// the app bar actions and the bottom action bar are withheld until then,
  /// since none of them has anything to act on. The body's reserved bottom
  /// padding stays constant either way, so nothing shifts when data lands.
  Widget _buildScreen(
    BuildContext context,
    Trail? trail, {
    required Widget panel,
  }) {
    final theme = Theme.of(context);
    final isUnsynced = trail != null && isUnsyncedState(trail.syncState);
    final availableOffline = trail != null && _availableOffline(trail);
    final isDownloading =
        trail != null &&
        trail.id.isNotEmpty &&
        ref.watch(downloadingTrailIdsProvider).contains(trail.id);
    return Scaffold(
      extendBodyBehindAppBar: true,
      // Scoped to the notifier: only the AppBar rebuilds as the scroll fade
      // progresses — see [_appBarOpacity].
      appBar: PreferredSize(
        preferredSize: const Size.fromHeight(kToolbarHeight),
        child: ValueListenableBuilder<double>(
          valueListenable: _appBarOpacity,
          builder: (context, opacity, _) => AppBar(
            leading: IconButton(
              icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
              onPressed: () => context.pop(),
              style: IconButton.styleFrom(
                backgroundColor: theme.colorScheme.surface.withValues(
                  alpha: 1.0 - opacity,
                ),
              ),
            ),

            backgroundColor: theme.colorScheme.surface.withValues(
              alpha: opacity,
            ),

            shadowColor: Colors.black.withValues(alpha: opacity * 0.15),
            elevation: opacity > 0 ? 2 : 0,

            scrolledUnderElevation: 0,

            actions: [
              if (trail != null) ...[
                if (!isUnsynced) ...[
                  LikeButton(trail: trail),
                  const SizedBox(width: 8),
                ],
                TrailDropdown(trail: trail, availableOffline: availableOffline),
              ],
            ],
          ),
        ),
      ),
      body: Stack(
        children: [
          Positioned.fill(
            child: Padding(
              padding: const EdgeInsets.only(bottom: 72),
              child: panel,
            ),
          ),

          if (trail != null)
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: Container(
                color: theme.scaffoldBackgroundColor,
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
                child: Row(
                  children: isUnsynced
                      ? [
                          Expanded(
                            child: ElevatedButton.icon(
                              onPressed: () async {
                                await context.push(
                                  '/trail/create/edit',
                                  extra: trail,
                                );
                                if (!mounted) return;
                                final lid = trail.localId;
                                if (lid != null) {
                                  ref.invalidate(localTrailProvider(lid));
                                }
                              },
                              icon: const FaIcon(FontAwesomeIcons.pen),
                              label: Text(AppLocalizations.of(context)!.edit),
                            ),
                          ),
                        ]
                      : [
                          if (!availableOffline) ...[
                            Expanded(
                              child: OutlinedButton.icon(
                                onPressed: isDownloading
                                    ? null
                                    : () => ref
                                          .read(
                                            downloadingTrailIdsProvider
                                                .notifier,
                                          )
                                          .download(trail),
                                icon: isDownloading
                                    ? SizedBox(
                                        width: 18,
                                        height: 18,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                          color: Theme.of(
                                            context,
                                          ).colorScheme.onSurface,
                                        ),
                                      )
                                    : FaIcon(
                                        FontAwesomeIcons.download,
                                        color: Theme.of(
                                          context,
                                        ).colorScheme.onSurface,
                                      ),
                                label: Text(
                                  AppLocalizations.of(context)!.download,
                                  style: TextStyle(
                                    color: Theme.of(
                                      context,
                                    ).colorScheme.onSurface,
                                  ),
                                ),
                              ),
                            ),
                            const SizedBox(width: 8),
                          ],
                          Expanded(
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
                                      if (mounted) {
                                        setState(() => _isLaunching = false);
                                      }
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
                                  : const FaIcon(
                                      FontAwesomeIcons.locationArrow,
                                    ),
                              label: Text(
                                AppLocalizations.of(context)!.navigate,
                              ),
                            ),
                          ),
                        ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
