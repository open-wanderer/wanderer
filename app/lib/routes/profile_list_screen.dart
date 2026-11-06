import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_empty_state.dart';
import 'package:wanderer/components/base/wanderer_no_results.dart';
import 'package:wanderer/components/base/wanderer_searchbar.dart';
import 'package:wanderer/components/list/list_list_item.dart';
import 'package:wanderer/components/list/list_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/paged_load_more.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';
import 'package:wanderer/util/format.dart';

class ProfileListScreen extends ConsumerStatefulWidget {
  final String handle;
  const ProfileListScreen({super.key, required this.handle});

  @override
  ConsumerState<ProfileListScreen> createState() => _ProfileListScreenState();
}

class _ProfileListScreenState extends ConsumerState<ProfileListScreen> {
  late final ScrollController _scrollController;
  String _query = '';

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
    // A search that returns within the cross-fade has the outgoing list and
    // its replacement both attached, and `.position` throws on two.
    if (_scrollController.positions.length != 1) return;
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
    final ownUsername = ref.watch(authProvider).value?.preferredUsername;
    final isOwn = ownUsername != null && widget.handle == '@$ownUsername';

    return Scaffold(
      appBar: AppBar(
        scrolledUnderElevation: 0,
        leading: BackButton(onPressed: () => context.pop()),
        title: Text(AppLocalizations.of(context)!.list(2)),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: WandererSearchBar(
              hintText: AppLocalizations.of(context)!.search,
              onChanged: (q) {
                setState(() => _query = q.trim());
                ref
                    .read(profileListsProvider(widget.handle).notifier)
                    .search(q);
              },
            ),
          ),
          Align(
            alignment: Alignment.centerLeft,
            child: ListQuickFilterBar(
              filterId: 'profile_list_${widget.handle}',
            ),
          ),
          // A filter change reloads behind the list it keeps on screen.
          if (listsAsync.isLoading && listsAsync.hasValue)
            const LinearProgressIndicator(minHeight: 2),
          Expanded(
            child: AsyncLoader<ProfileListsState>(
              asyncValue: listsAsync,
              mockData: ProfileListsState.mock(),
              keepValueWhileLoading: true,
              builder: (state) => RefreshIndicator(
                color: Theme.of(context).colorScheme.onSurface,
                onRefresh: () async =>
                    ref.invalidate(profileListsProvider(widget.handle)),
                child: state.lists.isEmpty
                    ? (_query.isNotEmpty
                          ? WandererNoResults(query: _query)
                          : _EmptyState(
                              isOwn: isOwn,
                              username: usernameFromHandle(widget.handle),
                            ))
                    : ListView.builder(
                        // Not on the skeleton: it is on screen alongside the
                        // real list during the cross-fade.
                        controller: listsAsync.hasValue
                            ? _scrollController
                            : null,
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
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final bool isOwn;
  final String username;

  const _EmptyState({required this.isOwn, required this.username});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return WandererEmptyState(
      icon: FontAwesomeIcons.layerGroup,
      title: isOwn
          ? l10n.profile_lists_empty_own_title
          : l10n.profile_lists_empty_other_title,
      body: isOwn
          ? l10n.profile_lists_empty_own_body
          : l10n.profile_lists_empty_other_body(username),
    );
  }
}
