import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/category_preference_sort.dart';

/// SETCAT-08/10/11 (subcategory half): the leaf screen reached by tapping a
/// category row. It shows a parent category's subcategories with priority sort,
/// per-row visibility toggle (auto-save, error-toast-only), drag-handle reorder
/// scoped to the parent category, an empty state for categories with no
/// subcategories, and the own-trail confirm-before-disable dialog whose "View
/// trails" action opens the user's own profile trail list pre-filtered to the
/// subcategory.
///
/// Mirrors [SettingsCategoriesScreen] (D-06) but takes a required parent
/// [Category] (passed via go_router `extra` in Plan 04) and scopes its list +
/// reorder to that category. A `ConsumerStatefulWidget` because it holds the
/// local `_orderedIds` drag working copy (never rendered from the re-sorting
/// provider mid-drag) and needs `mounted` guards after async saves.
class SettingsSubcategoriesScreen extends ConsumerStatefulWidget {
  const SettingsSubcategoriesScreen({super.key, required this.category});

  /// The parent category whose subcategories this screen configures. Its
  /// locale-resolved name is the AppBar title (D-06).
  final Category category;

  @override
  ConsumerState<SettingsSubcategoriesScreen> createState() =>
      _SettingsSubcategoriesScreenState();
}

class _SettingsSubcategoriesScreenState
    extends ConsumerState<SettingsSubcategoriesScreen> {
  /// Working copy of the current sorted ordering, keyed by subcategory id.
  /// Seeded from the provider-derived sort on every build and mutated
  /// optimistically during a drag (Pitfall 2 — never render reorder from the
  /// re-sorting provider mid-drag).
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
    // subcategoryProvider is a synchronous List; only the preference provider is
    // async, so AsyncLoader (D-14) wraps just the prefs load.
    final subcategories = ref.watch(subcategoryProvider);
    final prefsAsync = ref.watch(subcategoryPreferenceProvider);
    final locale = Localizations.localeOf(context);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    // Filter to this parent category's subcategories (SETCAT-08 scoping).
    final filtered = subcategories
        .where((s) => s.category == widget.category.id)
        .toList();

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        // Parent category's localized name (D-06).
        title: Text(widget.category.displayName(locale)),
      ),
      body: AsyncLoader<List<SubcategoryPreference>>(
        asyncValue: prefsAsync,
        mockData: const [],
        builder: (prefs) {
          // Empty state — the screen stays reachable even with no subcategories
          // (D-07).
          if (filtered.isEmpty) {
            return _buildEmptyState(l10n);
          }

          final sorted = sortedSubcategoriesByPreference(
            filtered,
            prefs,
            locale,
          );
          // Seed the reorder working copy from the current sorted order. On the
          // non-drag path this simply mirrors `sorted`; a drag mutates it
          // optimistically (Task 2).
          _orderedIds = sorted.map((s) => s.id).toList();

          return _buildList(sorted, prefs, locale, activeColor);
        },
      ),
    );
  }

  /// Empty-state copy for a category with no subcategories (D-07). Reuses the
  /// shared `settings_categories_empty_*` keys.
  Widget _buildEmptyState(AppLocalizations l10n) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              l10n.settings_categories_empty_title,
              style: Theme.of(context).textTheme.titleMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.settings_categories_empty_body,
              style: Theme.of(context).textTheme.bodyMedium,
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  /// Renders the reorderable subcategory list from the local `_orderedIds`
  /// working copy (wired in Task 2).
  Widget _buildList(
    List<Subcategory> sorted,
    List<SubcategoryPreference> prefs,
    Locale locale,
    Color activeColor,
  ) {
    final byId = {for (final s in sorted) s.id: s};
    return ListView.builder(
      itemCount: _orderedIds.length,
      itemBuilder: (context, index) {
        final sub = byId[_orderedIds[index]];
        if (sub == null) return const SizedBox.shrink();
        return _buildRow(sub, index, prefs, locale, activeColor);
      },
    );
  }

  /// A single subcategory row: drag handle (Task 2) + leading avatar + name +
  /// trailing visibility Switch. An explicit Row, not a switch-tile widget
  /// (Pattern 2 — mirrors the sibling category screen, D-06). Rows are LEAF —
  /// no body-tap navigation (D-06).
  Widget _buildRow(
    Subcategory sub,
    int index,
    List<SubcategoryPreference> prefs,
    Locale locale,
    Color activeColor,
  ) {
    final isVisible = subcategoryVisible(sub.id, prefs);
    return Padding(
      key: ValueKey(sub.id),
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          ReorderableDragStartListener(
            index: index,
            child: const Padding(
              padding: EdgeInsets.symmetric(horizontal: 8),
              child: Icon(Icons.drag_handle),
            ),
          ),
          subcategoryFilterAvatar(context, sub, widget.category, locale),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              sub.displayName(locale),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          Switch(
            value: isVisible,
            activeThumbColor: activeColor,
            onChanged: (v) => _onToggle(sub, v),
          ),
          const SizedBox(width: 8),
        ],
      ),
    );
  }

  /// ON (D-11) saves immediately with no check; OFF routes to the own-trail
  /// guarded flow (implemented in Task 2).
  void _onToggle(Subcategory sub, bool value) {
    if (value) {
      _save(
        () => ref
            .read(subcategoryPreferenceProvider.notifier)
            .upsert(sub.id, true),
      );
    } else {
      _onToggleOff(sub);
    }
  }

  /// OFF path — own-trail confirm-before-disable (implemented in Task 2).
  Future<void> _onToggleOff(Subcategory sub) async {
    await _save(
      () => ref
          .read(subcategoryPreferenceProvider.notifier)
          .upsert(sub.id, false),
    );
  }
}
