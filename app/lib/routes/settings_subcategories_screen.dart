import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/category_preference_sort.dart';
import 'package:wanderer/util/own_trail_count.dart';

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
  /// working copy (Pitfall 2 — never render reorder from the re-sorting provider
  /// mid-drag). `buildDefaultDragHandles: false` disables whole-row/long-press
  /// drag (D-01); the only drag affordance is the explicit
  /// `ReorderableDragStartListener` handle built in `_buildRow`.
  Widget _buildList(
    List<Subcategory> sorted,
    List<SubcategoryPreference> prefs,
    Locale locale,
    Color activeColor,
  ) {
    final byId = {for (final s in sorted) s.id: s};
    return ReorderableListView.builder(
      buildDefaultDragHandles: false,
      itemCount: _orderedIds.length,
      itemBuilder: (context, index) {
        final sub = byId[_orderedIds[index]];
        if (sub == null) return const SizedBox.shrink();
        return _buildRow(sub, index, prefs, locale, activeColor);
      },
      // The index-shift `if (newIndex > oldIndex) newIndex -= 1` inside
      // _onReorder is the canonical onReorder contract; onReorderItem is not
      // used so that shift stays explicit and testable (Pitfall 1).
      // ignore: deprecated_member_use
      onReorder: (oldIndex, newIndex) => _onReorder(oldIndex, newIndex),
    );
  }

  /// Optimistically reorders `_orderedIds`, POSTs the parent category id + the
  /// full ordered subcategory id list (D-02/SETCAT-10), and reverts to the
  /// pre-drag snapshot + error toast on failure (D-04/D-09).
  Future<void> _onReorder(int oldIndex, int newIndex) async {
    // Flutter's ReorderableListView passes a newIndex that assumes the dragged
    // item is still present; adjust for the removal (Pitfall 1).
    if (newIndex > oldIndex) newIndex -= 1;

    final snapshot = List<String>.from(_orderedIds);
    final reordered = List<String>.from(_orderedIds);
    final moved = reordered.removeAt(oldIndex);
    reordered.insert(newIndex, moved);
    setState(() => _orderedIds = reordered);

    final l10n = AppLocalizations.of(context)!;
    try {
      // Parent category id FIRST (SETCAT-10 scoping — payload
      // {category, subcategories}).
      await ref
          .read(subcategoryPreferenceProvider.notifier)
          .reorder(widget.category.id, _orderedIds);
    } catch (_) {
      if (!mounted) return;
      setState(() => _orderedIds = snapshot);
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

  /// OFF path — own-trail confirm-before-disable (SETCAT-11 subcategory half,
  /// D-10). Lazily counts the user's own trails in [sub] (D-12, never
  /// preloaded); if none, saves directly, otherwise shows a confirm dialog whose
  /// "View trails" action navigates to the user's own profile trail list
  /// pre-filtered to the subcategory. Confirm saves; cancel is a no-op (the
  /// switch reverts because provider state was never changed).
  Future<void> _onToggleOff(Subcategory sub) async {
    final int count;
    try {
      count = await ownTrailCount(ref, isSubcategory: true, id: sub.id);
    } catch (_) {
      if (!mounted) return;
      final l10n = AppLocalizations.of(context)!;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
      return;
    }
    if (!mounted) return;

    if (count == 0) {
      await _save(
        () => ref
            .read(subcategoryPreferenceProvider.notifier)
            .upsert(sub.id, false),
      );
      return;
    }

    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(
          l10n.settings_categories_confirm_disable_subcategory_title,
        ),
        content: Text(
          l10n.settings_categories_confirm_disable_body(count),
        ),
        actions: [
          TextButton(
            onPressed: () => _viewOwnTrails(dialogContext, sub),
            child: Text(l10n.settings_categories_confirm_view_trails),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(l10n.settings_categories_confirm_disable_confirm),
          ),
        ],
      ),
    );

    if (confirmed != true) return;
    if (!mounted) return;

    await _save(
      () => ref
          .read(subcategoryPreferenceProvider.notifier)
          .upsert(sub.id, false),
    );
  }

  /// Dismisses the confirm dialog and pushes the user's OWN profile trail list
  /// pre-filtered to [sub]. Resolves the own handle WITH the `@` sigil exactly
  /// as `profile_screen.dart:38` and Plan 01's `own_trail_count.dart` do — every
  /// self `/profile/{handle}/...` call site uses the `@`-prefixed handle.
  /// Navigating here does NOT save or disable; the user returns to an unchanged
  /// switch.
  void _viewOwnTrails(BuildContext dialogContext, Subcategory sub) {
    final l10n = AppLocalizations.of(context)!;
    final username = ref.read(authProvider).value?.preferredUsername;
    if (username == null) {
      // Not signed in — should not happen on this screen. Never build a bogus
      // handle from a null username; surface an error toast and abort (no
      // silent no-op).
      Navigator.of(dialogContext).pop(false);
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
      return;
    }
    // `@`-prefixed handle, exactly as profile_screen.dart:38 and
    // own_trail_count.dart resolve it from authProvider's preferredUsername.
    final handle = '@${ref.read(authProvider).value!.preferredUsername}';

    // Seed the destination screen's filter so the pushed list is pre-filtered.
    // profile_trail_screen keys its filter by exactly 'profile_trail_$handle'
    // with the same `@`-prefixed handle; TrailFilter.subcategory is a
    // List<Subcategory> — pass the current sub.
    // ignore: lines_longer_than_80_chars
    ref.read(trailFilterProvider('profile_trail_$handle').notifier).updateFilter((f) => f.copyWith(category: const [], subcategory: [sub]));

    Navigator.of(dialogContext).pop(false);
    if (!context.mounted) return;
    context.push('/profile/$handle/trails');
  }
}
