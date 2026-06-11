import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
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

  @override
  Widget build(BuildContext context) {
    final trailsAsync = ref.watch(profileTrailsProvider(widget.handle));

    return Scaffold(
      appBar: AppBar(
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(AppLocalizations.of(context)!.trail(2)),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: WandererSearchBar(
              hintText: AppLocalizations.of(context)!.search,
              onChanged: (q) => ref
                  .read(profileTrailsProvider(widget.handle).notifier)
                  .search(q),
            ),
          ),
          Expanded(
            child: trailsAsync.when(
              data: (state) => RefreshIndicator(
                onRefresh: () async =>
                    ref.invalidate(profileTrailsProvider(widget.handle)),
                child: state.trails.isEmpty
                    ? _EmptyState()
                    : ListView.builder(
                        controller: _scrollController,
                        padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                        itemCount:
                            state.trails.length + (state.hasMore ? 1 : 0),
                        itemBuilder: (context, index) {
                          if (index == state.trails.length) {
                            return const Padding(
                              padding: EdgeInsets.symmetric(vertical: 16),
                              child: Center(child: CircularProgressIndicator()),
                            );
                          }
                          final trail = state.trails[index];
                          return TrailListItem(
                            trail: trail,
                            onTrailSelect: () =>
                                context.push('/trail/${trail.id}'),
                          );
                        },
                      ),
              ),
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
      child: Text('No trails yet.', style: TextStyle(color: Colors.grey[600])),
    );
  }
}
