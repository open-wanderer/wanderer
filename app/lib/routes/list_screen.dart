import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/list/list_card.dart';
import 'package:wanderer/components/list/list_quick_filter_bar.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';
import 'package:wanderer/provider/list/list_search_provider.dart';

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
    if (pos.pixels / pos.maxScrollExtent >= 0.8) {
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

  @override
  Widget build(BuildContext context) {
    final searchState = ref.watch(listSearchProvider);

    return Scaffold(
      body: SafeArea(
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
                    hintText: 'Search lists…',
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
                  onRefresh: () async => ref.invalidate(listSearchProvider),
                  child: AsyncLoader<ListSearchState>(
                    asyncValue: searchState,
                    mockData: ListSearchState.mock(),
                    builder: (state) {
                      if (state.lists.isEmpty) {
                        return const Center(child: Text('No lists found'));
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
                              child: Center(child: CircularProgressIndicator()),
                            );
                          }
                          return ListCard(
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
      ),
    );
  }
}
