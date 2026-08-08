import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/list/list_list_item.dart';
import 'package:wanderer/components/list/list_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';

class ProfileListScreen extends ConsumerStatefulWidget {
  final String handle;
  const ProfileListScreen({super.key, required this.handle});

  @override
  ConsumerState<ProfileListScreen> createState() => _ProfileListScreenState();
}

class _ProfileListScreenState extends ConsumerState<ProfileListScreen> {
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
    if (pos.extentAfter < kPagedPrefetchExtent) {
      final state = ref.read(profileListsProvider(widget.handle));
      if (state.value?.hasMore == true && !state.isLoading) {
        ref.read(profileListsProvider(widget.handle).notifier).loadNextPage();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final listsAsync = ref.watch(profileListsProvider(widget.handle));

    return Scaffold(
      appBar: AppBar(
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(AppLocalizations.of(context)!.list(2)),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: WandererSearchBar(
              hintText: AppLocalizations.of(context)!.search,
              onChanged: (q) => ref
                  .read(profileListsProvider(widget.handle).notifier)
                  .search(q),
            ),
          ),
          Align(
            alignment: Alignment.centerLeft,
            child: ListQuickFilterBar(
              filterId: 'profile_list_${widget.handle}',
            ),
          ),
          Expanded(
            child: listsAsync.when(
              data: (state) => RefreshIndicator(
                color: Theme.of(context).colorScheme.onSurface,
                onRefresh: () async =>
                    ref.invalidate(profileListsProvider(widget.handle)),
                child: state.lists.isEmpty
                    ? _EmptyState()
                    : ListView.builder(
                        controller: _scrollController,
                        padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                        itemCount: state.lists.length + (state.hasMore ? 1 : 0),
                        itemBuilder: (context, index) {
                          if (index == state.lists.length) {
                            return const Padding(
                              padding: EdgeInsets.symmetric(vertical: 16),
                              child: Center(child: CircularProgressIndicator()),
                            );
                          }
                          final list = state.lists[index];
                          return ListListItem(
                            key: ValueKey(list.id),
                            list: list,
                            onListSelect: () =>
                                context.push('/list/${list.id}'),
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
      child: Text(
        AppLocalizations.of(context)!.no_lists_yet,
        style: TextStyle(color: Colors.grey[600]),
      ),
    );
  }
}
