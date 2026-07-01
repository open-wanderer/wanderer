import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/category_preference.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/category_preference_sort.dart';

/// SETCAT-06/07/09/11 (category half): the category list with priority sort,
/// per-row visibility toggle (auto-save, error-toast-only), drag-handle
/// reorder, and the own-trail confirm-before-disable dialog whose "View trails"
/// action opens the user's own profile trail list pre-filtered to the category.
///
/// A `ConsumerStatefulWidget` because it holds the local `_orderedIds` drag
/// working copy (never rendered from the re-sorting provider mid-drag) and needs
/// `mounted` guards after async saves.
class SettingsCategoriesScreen extends ConsumerStatefulWidget {
  const SettingsCategoriesScreen({super.key});

  @override
  ConsumerState<SettingsCategoriesScreen> createState() =>
      _SettingsCategoriesScreenState();
}

class _SettingsCategoriesScreenState
    extends ConsumerState<SettingsCategoriesScreen> {
  /// Working copy of the current sorted ordering, keyed by category id. Seeded
  /// from the provider-derived sort on every build and mutated optimistically
  /// during a drag (Pitfall 2 — never render reorder from the re-sorting
  /// provider mid-drag).
  List<String> _orderedIds = const [];

  /// Persists [op] and surfaces only an error toast on failure (D-08 — no
  /// success toast; the watched provider drives optimistic UI via
  /// `invalidateSelf`).
  Future<void> _save(Future<void> Function() op) async {
    final l10n = AppLocalizations.of(context)!;
    try {
      await op();
    } catch (_) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final categoriesAsync = ref.watch(categoryProvider);
    final prefsAsync = ref.watch(categoryPreferenceProvider);
    final locale = Localizations.localeOf(context);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    // Combine the two async values into a single record so AsyncLoader (D-14)
    // renders one skeleton over the list rather than a bespoke spinner.
    final combined =
        categoriesAsync.hasError
        ? AsyncValue<
            ({List<Category> categories, List<CategoryPreference> prefs})
          >.error(categoriesAsync.error!, categoriesAsync.stackTrace!)
        : prefsAsync.hasError
        ? AsyncValue<
            ({List<Category> categories, List<CategoryPreference> prefs})
          >.error(prefsAsync.error!, prefsAsync.stackTrace!)
        : (categoriesAsync.isLoading || prefsAsync.isLoading)
        ? const AsyncValue<
            ({List<Category> categories, List<CategoryPreference> prefs})
          >.loading()
        : AsyncValue<
            ({List<Category> categories, List<CategoryPreference> prefs})
          >.data((
            categories: categoriesAsync.value ?? const [],
            prefs: prefsAsync.value ?? const [],
          ));

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.categories),
      ),
      body: AsyncLoader<
        ({List<Category> categories, List<CategoryPreference> prefs})
      >(
        asyncValue: combined,
        mockData: const (categories: [], prefs: []),
        builder: (data) {
          final sorted = sortedCategoriesByPreference(
            data.categories,
            data.prefs,
            locale,
          );
          // Seed the reorder working copy from the current sorted order. On the
          // non-drag path this simply mirrors `sorted`; a drag mutates it
          // optimistically (Task 2).
          _orderedIds = sorted.map((c) => c.id).toList();

          return _buildList(sorted, data.prefs, locale, activeColor);
        },
      ),
    );
  }

  /// Renders the sorted category rows. Extended in Task 2 to a
  /// `ReorderableListView.builder`; Task 1 renders the row layout + toggle.
  Widget _buildList(
    List<Category> sorted,
    List<CategoryPreference> prefs,
    Locale locale,
    Color activeColor,
  ) {
    return ListView.builder(
      itemCount: sorted.length,
      itemBuilder: (context, index) =>
          _buildRow(sorted[index], index, prefs, locale, activeColor),
    );
  }

  /// A single category row: drag handle (Task 2) + tappable body (navigate,
  /// Task 2) + trailing visibility Switch. An explicit Row, not a switch-tile
  /// (Pattern 2 — the body must navigate independently of the switch).
  Widget _buildRow(
    Category category,
    int index,
    List<CategoryPreference> prefs,
    Locale locale,
    Color activeColor,
  ) {
    final isVisible = categoryVisible(category.id, prefs);
    return Padding(
      key: ValueKey(category.id),
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 8),
            child: Icon(Icons.drag_handle),
          ),
          Expanded(
            child: InkWell(
              onTap: () {},
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Row(
                  children: [
                    categoryFilterAvatar(category),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        category.displayName(locale),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
          Switch(
            value: isVisible,
            activeThumbColor: activeColor,
            onChanged: (v) => _onToggle(category, v),
          ),
          const SizedBox(width: 8),
        ],
      ),
    );
  }

  /// ON (D-11) saves immediately with no check; OFF routes to the own-trail
  /// guarded flow (implemented in Task 2).
  void _onToggle(Category category, bool value) {
    if (value) {
      _save(
        () => ref
            .read(categoryPreferenceProvider.notifier)
            .upsert(category.id, true),
      );
    } else {
      _onToggleOff(category);
    }
  }

  /// OFF path — own-trail confirm-before-disable (implemented in Task 2).
  void _onToggleOff(Category category) {
    _save(
      () => ref
          .read(categoryPreferenceProvider.notifier)
          .upsert(category.id, false),
    );
  }
}
