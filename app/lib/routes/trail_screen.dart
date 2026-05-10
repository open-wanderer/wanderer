import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';

class TrailScreen extends ConsumerStatefulWidget {
  const TrailScreen({super.key});

  @override
  ConsumerState<TrailScreen> createState() => _TrailScreenState();
}

class _TrailScreenState extends ConsumerState<TrailScreen> {
  @override
  Widget build(BuildContext context) {
    final router = ref.watch(routerProvider);
    final trailsAsync = ref.watch(trailSearchProvider);

    return Scaffold(
      body: SafeArea(
        child: CustomScrollView(
          slivers: [
            SliverAppBar(
              floating: true,
              snap: true,
              stretch: true,
              toolbarHeight: 70,
              backgroundColor: Theme.of(context).canvasColor,
              surfaceTintColor: Theme.of(context).canvasColor,
              title: Padding(
                padding: const EdgeInsets.all(16.0),
                child: Row(
                  children: [
                    Expanded(
                      child: Material(
                        color: Colors.white,
                        elevation: 4,
                        shadowColor: Colors.black26,
                        borderRadius: BorderRadius.circular(30),
                        clipBehavior: Clip.antiAlias,
                        child: WandererSearchBar(
                          onChanged: (value) {
                            ref
                                .read(trailFilterProvider.notifier)
                                .updateFilter((f) => f.copyWith(q: value));
                          },
                          decoration: InputDecoration(
                            hintText: AppLocalizations.of(
                              context,
                            )!.search_trails,
                            hintStyle: const TextStyle(color: Colors.grey),
                            prefixIcon: const Icon(
                              Icons.search,
                              color: Colors.black,
                            ),
                            border: InputBorder.none,
                            contentPadding: const EdgeInsets.symmetric(
                              vertical: 15,
                            ),
                          ),
                        ),
                      ),
                    ),
                    IconButton(
                      onPressed: () => router.push('/trail/filter'),
                      iconSize: 20,
                      icon: const FaIcon(FontAwesomeIcons.filter),
                    ),
                  ],
                ),
              ),
            ),

            trailsAsync.when(
              data: (state) => SliverPadding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                sliver: SliverList(
                  delegate: SliverChildBuilderDelegate((context, index) {
                    final trail = state.trails[index];
                    return TrailCard(
                      trail: trail,
                      onTrailSelect: () =>
                          router.push("/trail/${trail.id}", extra: trail),
                    );
                  }, childCount: state.trails.length),
                ),
              ),
              loading: () => SliverFillRemaining(
                child: Center(
                  child: CircularProgressIndicator(color: Colors.grey[400]),
                ),
              ),
              error: (err, stack) => SliverFillRemaining(
                child: Center(
                  child: Padding(
                    padding: const EdgeInsets.all(24.0),
                    child: Text(
                      'Error loading trails: $err',
                      textAlign: TextAlign.center,
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
