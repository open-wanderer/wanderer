import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/search_history_item.dart';
import 'package:wanderer/provider/search/search_history_provider.dart';

/// "Recent searches": [items] under a heading whose trashcan clears them
/// all, each swipeable to remove just that one.
///
/// Clears and removes exactly the [items] it is given, so a screen showing
/// a filtered history (location search shows only places) never wipes
/// entries it did not display.
class SearchHistoryList extends ConsumerWidget {
  final List<SearchHistoryItem> items;
  final Widget Function(SearchHistoryItem item) tileBuilder;

  const SearchHistoryList({
    super.key,
    required this.items,
    required this.tileBuilder,
  });

  Future<void> _confirmClear(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.clear_search_history),
        content: Text(l10n.clear_search_history_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(l10n.delete, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    ref.read(searchHistoryProvider.notifier).remove(items);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 4, 0),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  l10n.recent_searches,
                  style: theme.textTheme.titleSmall,
                ),
              ),
              IconButton(
                icon: const FaIcon(FontAwesomeIcons.trashCan, size: 16),
                tooltip: l10n.clear_search_history,
                onPressed: () => _confirmClear(context, ref),
              ),
            ],
          ),
        ),
        Expanded(
          child: ListView(
            children: [
              for (final item in items)
                Dismissible(
                  key: ValueKey(item.key),
                  background: const _DismissBackground(
                    alignment: Alignment.centerLeft,
                  ),
                  secondaryBackground: const _DismissBackground(
                    alignment: Alignment.centerRight,
                  ),
                  onDismissed: (_) =>
                      ref.read(searchHistoryProvider.notifier).remove([item]),
                  child: tileBuilder(item),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

class _DismissBackground extends StatelessWidget {
  final Alignment alignment;

  const _DismissBackground({required this.alignment});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Colors.red,
      alignment: alignment,
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: const FaIcon(
        FontAwesomeIcons.trashCan,
        size: 18,
        color: Colors.white,
      ),
    );
  }
}
