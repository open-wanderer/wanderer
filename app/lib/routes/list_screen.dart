import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/components/list/list_card.dart';
import 'package:wanderer/components/list/list_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';
import 'package:wanderer/provider/list/list_search_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';

class ListScreen extends ConsumerStatefulWidget {
  const ListScreen({super.key});

  @override
  ConsumerState<ListScreen> createState() => _ListScreenState();
}

class _ListScreenState extends ConsumerState<ListScreen> {
  final _searchController = TextEditingController();
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    final pos = _scrollController.position;
    if (!pos.hasContentDimensions) return;
    if (pos.maxScrollExtent <= 0) return;
    if (pos.extentAfter < kPagedPrefetchExtent) {
      final state = ref.read(listSearchProvider);
      if (state.value?.hasMore == true && !state.isLoading) {
        ref.read(listSearchProvider.notifier).loadNextPage();
      }
    }
  }

  void _onQueryChanged(String value) {
    ref
        .read(listFilterProvider('lists').notifier)
        .updateFilter((f) => f.copyWith(q: value));
    ref.invalidate(listSearchProvider);
  }

  Future<void> _retryOnline() async {
    final online = await ref.read(onlineStatusProvider.notifier).refresh();
    if (!online) return;
    ref.invalidate(listSearchProvider);
  }

  @override
  Widget build(BuildContext context) {
    final searchState = ref.watch(listSearchProvider);
    final isOnline = ref.watch(onlineStatusProvider);
    final l10n = AppLocalizations.of(context)!;

    return SafeArea(
      bottom: false,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8.0),
              child: TextField(
                controller: _searchController,
                onChanged: _onQueryChanged,
                cursorColor: Theme.of(context).colorScheme.onSurface,
                decoration: InputDecoration(
                  hintText: l10n.search_lists,
                  hintStyle: TextStyle(color: Colors.grey),
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: _searchController.text.isNotEmpty
                      ? IconButton(
                          icon: const Icon(Icons.clear),
                          onPressed: () {
                            _searchController.clear();
                            _onQueryChanged('');
                          },
                        )
                      : null,
                  enabledBorder: OutlineInputBorder(
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.outline,
                    ),
                    borderRadius: BorderRadius.circular(56),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(56),
                    borderSide: BorderSide(
                      color: Theme.of(context).colorScheme.onSurface,
                      width: 1,
                    ),
                  ),
                  isDense: true,
                ),
              ),
            ),
            const ListQuickFilterBar(filterId: 'lists'),
            Expanded(
              child: RefreshIndicator(
                color: Theme.of(context).colorScheme.onSurface,
                onRefresh: () async => ref.invalidate(listSearchProvider),
                child: !isOnline
                    ? WandererOfflineState(
                        title: l10n.offline_title,
                        body: l10n.offline_list_body,
                        retryLabel: l10n.offline_try_again,
                        onRetry: _retryOnline,
                      )
                    : AsyncLoader<ListSearchState>(
                        asyncValue: searchState,
                        mockData: ListSearchState.mock(),
                        builder: (state) {
                          if (state.lists.isEmpty) {
                            return Center(child: Text(l10n.no_lists_found));
                          }
                          return ListView.builder(
                            controller: _scrollController,
                            padding: const EdgeInsets.fromLTRB(0, 0, 0, 16),
                            itemCount:
                                state.lists.length + (state.hasMore ? 1 : 0),
                            itemBuilder: (context, i) {
                              if (i == state.lists.length) {
                                return const Padding(
                                  padding: EdgeInsets.symmetric(vertical: 16),
                                  child: Center(
                                    child: CircularProgressIndicator(),
                                  ),
                                );
                              }
                              return ListCard(
                                key: ValueKey(state.lists[i].id),
                                list: state.lists[i],
                                mini: false,
                                onListSelect: () =>
                                    context.push('/list/${state.lists[i].id}'),
                              );
                            },
                          );
                        },
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
