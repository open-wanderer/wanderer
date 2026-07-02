import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/category_preference.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/category_preference_sort.dart';
import 'package:wanderer/util/own_trail_count.dart';

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

  /// True for the entire drag→`reorder()` round-trip: set in `onReorderStart`
  /// and cleared only after the async `reorder()` network call resolves
  /// (success or caught failure) inside `_onReorder`. While true, `build()`
  /// must not reseed `_orderedIds` from the provider-derived sort. This covers
  /// two cases:
  ///   1. The non-optimistic snap-back — clearing on drag-end (as before) let
  ///      the next `build()` reseed `_orderedIds` from still-stale provider
  ///      data before the server responded, snapping the dropped row back to
  ///      its original slot until the provider re-emitted.
  ///   2. WR-02 mid-drag protection — an unrelated `ref.watch` dependency
  ///      change mid-drag (e.g. a background provider refresh) must not
  ///      silently reset the optimistic working copy.
  bool _reordering = false;

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
    // subcategoryProvider is a synchronous List (like categoryProvider); it does
    // NOT participate in the async error/loading combine. subcategoryPrefsAsync
    // IS async and MUST join the combine.
    final subcategories = ref.watch(subcategoryProvider);
    final subcategoryPrefsAsync = ref.watch(subcategoryPreferenceProvider);
    final locale = Localizations.localeOf(context);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    // Combine the two async values into a single record so AsyncLoader (D-14)
    // renders one skeleton over the list rather than a bespoke spinner.
    // Prefer whichever data each sub-provider still carries (Riverpod keeps
    // `.value` populated through a seamless refresh after `invalidateSelf()`)
    // over collapsing to a bare `.loading()` — a bare loading state has no
    // previous value, which forced AsyncLoader's fallback to empty mockData
    // and produced a visible flash on every toggle/reorder save.
    final combined = categoriesAsync.hasError
        ? AsyncValue<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >.error(categoriesAsync.error!, categoriesAsync.stackTrace!)
        : prefsAsync.hasError
        ? AsyncValue<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >.error(prefsAsync.error!, prefsAsync.stackTrace!)
        : subcategoryPrefsAsync.hasError
        ? AsyncValue<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >.error(
            subcategoryPrefsAsync.error!,
            subcategoryPrefsAsync.stackTrace!,
          )
        : categoriesAsync.hasValue &&
              prefsAsync.hasValue &&
              subcategoryPrefsAsync.hasValue
        ? AsyncValue<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >.data((
            categories: categoriesAsync.value ?? const [],
            prefs: prefsAsync.value ?? const [],
            subcategoryPrefs: subcategoryPrefsAsync.value ?? const [],
          ))
        : const AsyncValue<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >.loading();

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.categories),
      ),
      body:
          AsyncLoader<
            ({
              List<Category> categories,
              List<CategoryPreference> prefs,
              List<SubcategoryPreference> subcategoryPrefs,
            })
          >(
            asyncValue: combined,
            mockData: const (categories: [], prefs: [], subcategoryPrefs: []),
            builder: (data) {
              final sorted = sortedCategoriesByPreference(
                data.categories,
                data.prefs,
                locale,
              );
              // Seed the reorder working copy from the current sorted order. On
              // the idle path this simply mirrors `sorted`; a drag mutates it
              // optimistically. Skip reseeding for the whole reorder round-trip
              // (`_reordering`): while a drag is in progress OR the async
              // `reorder()` is still in flight, a rebuild must not clobber the
              // optimistic working copy — reseeding from the not-yet-updated
              // provider would snap the dropped row back to its old slot.
              if (!_reordering) {
                _orderedIds = sorted.map((c) => c.id).toList();
              }

              return _buildList(
                sorted,
                data.prefs,
                locale,
                activeColor,
                subcategories,
                data.subcategoryPrefs,
              );
            },
          ),
    );
  }

  /// Renders the reorderable category list from the local `_orderedIds` working
  /// copy (Pitfall 2 — never render reorder from the re-sorting provider
  /// mid-drag). `buildDefaultDragHandles: false` disables whole-row/long-press
  /// drag (D-01); the only drag affordance is the explicit
  /// `ReorderableDragStartListener` handle built in `_buildRow`.
  Widget _buildList(
    List<Category> sorted,
    List<CategoryPreference> prefs,
    Locale locale,
    Color activeColor,
    List<Subcategory> subcategories,
    List<SubcategoryPreference> subcategoryPrefs,
  ) {
    final byId = {for (final c in sorted) c.id: c};
    return ReorderableListView.builder(
      buildDefaultDragHandles: false,
      itemCount: _orderedIds.length,
      itemBuilder: (context, index) {
        final category = byId[_orderedIds[index]];
        if (category == null) return const SizedBox.shrink();
        return _buildRow(
          category,
          index,
          prefs,
          locale,
          activeColor,
          subcategories,
          subcategoryPrefs,
        );
      },
      // The index-shift `if (newIndex > oldIndex) newIndex -= 1` inside
      // _onReorder is the canonical onReorder contract; onReorderItem is not
      // used so that shift stays explicit and testable (Pitfall 1).
      // ignore: deprecated_member_use
      onReorder: (oldIndex, newIndex) =>
          _onReorder(oldIndex, newIndex, prefs, locale),
      // Guard the working copy for the full drag→reorder() round-trip. It is
      // cleared inside `_onReorder` once the network call resolves — NOT on
      // drag-end, which fires synchronously before `await reorder()` returns
      // and would let the next build reseed stale data (the snap-back bug).
      onReorderStart: (_) => setState(() => _reordering = true),
    );
  }

  /// Optimistically reorders `_orderedIds`, POSTs the full ordered id list
  /// (D-02/SETCAT-09), and reverts to the pre-drag snapshot + error toast on
  /// failure (D-04/D-09).
  Future<void> _onReorder(
    int oldIndex,
    int newIndex,
    List<CategoryPreference> prefs,
    Locale locale,
  ) async {
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
      await ref.read(categoryPreferenceProvider.notifier).reorder(_orderedIds);
      // reorder() calls invalidateSelf() internally but does not wait for the
      // resulting refetch — clearing the guard here would let a rebuild land
      // with _reordering already false while prefsAsync.value still carries
      // the pre-reorder order (Riverpod keeps the previous value through the
      // loading state), reseeding _orderedIds back to the stale order for one
      // frame before the refetch resolves and snaps it forward again (the
      // reported flash). Await the refetch itself so the guard only drops
      // once the provider's data actually reflects the new order.
      if (!mounted) return;
      await ref.read(categoryPreferenceProvider.future);
      // Success: the provider has re-emitted the persisted order, so the guard
      // can drop — the next build's reseed will now mirror the settled order.
      if (!mounted) return;
      setState(() => _reordering = false);
    } catch (_) {
      if (!mounted) return;
      // Failure: revert to the pre-drag snapshot, surface the error toast, and
      // clear the guard so the reverted order is what future builds reseed.
      setState(() {
        _orderedIds = snapshot;
        _reordering = false;
      });
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

  /// A single category row: drag handle (Task 2) + tappable body (navigate,
  /// Task 2) + trailing visibility Switch. An explicit Row, not a switch-tile
  /// (Pattern 2 — the body must navigate independently of the switch).
  Widget _buildRow(
    Category category,
    int index,
    List<CategoryPreference> prefs,
    Locale locale,
    Color activeColor,
    List<Subcategory> subcategories,
    List<SubcategoryPreference> subcategoryPrefs,
  ) {
    final isVisible = categoryVisible(category.id, prefs);
    // This category's subcategories (same filter the sibling subcategory screen
    // uses). Read-only chips beneath the row surface them at a glance without
    // opening the leaf screen.
    final subs = subcategories
        .where((s) => s.category == category.id)
        .toList();
    return Padding(
      key: ValueKey(category.id),
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              ReorderableDragStartListener(
                index: index,
                child: const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 8),
                  child: Icon(Icons.drag_handle),
                ),
              ),
              Expanded(
                child: InkWell(
                  // ignore: lines_longer_than_80_chars
                  onTap: () => context.push(
                    '/settings/categories/subcategories',
                    extra: category,
                  ),
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
          // Read-only subcategory chip row (mirrors web
          // `{#if childSubcategories.length > 0}`). Each chip dims solely on the
          // subcategory's OWN visibility — deliberately NOT compounded with the
          // parent category's visibility (user's explicit divergence from web).
          if (subs.isNotEmpty)
            Padding(
              // Left-align the chips under the category label, roughly matching
              // the drag-handle + avatar + gap lead-in.
              padding: const EdgeInsets.only(
                left: 52,
                right: 8,
                bottom: 8,
              ),
              child: Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  for (final s in subs)
                    Opacity(
                      // Dim ONLY on the subcategory's own visibility toggle.
                      opacity: subcategoryVisible(s.id, subcategoryPrefs)
                          ? 1.0
                          : 0.5,
                      // A BARE Chip is non-interactive in both states: no
                      // onTap/onPressed/onDeleted and not an
                      // Action/Choice/Input/Filter chip, so it has no tap
                      // affordance.
                      child: Chip(
                        avatar: subcategoryFilterAvatar(
                          context,
                          s,
                          category,
                          locale,
                        ),
                        label: Text(
                          s.displayName(locale),
                          overflow: TextOverflow.ellipsis,
                        ),
                        labelStyle: Theme.of(context).textTheme.bodySmall,
                        visualDensity: VisualDensity.compact,
                        materialTapTargetSize:
                            MaterialTapTargetSize.shrinkWrap,
                      ),
                    ),
                ],
              ),
            ),
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

  /// OFF path — own-trail confirm-before-disable (SETCAT-11, D-10). Lazily
  /// counts the user's own trails in [category] (D-12, never preloaded); if
  /// none, saves directly, otherwise shows a confirm dialog whose "View trails"
  /// action navigates to the user's own profile trail list pre-filtered to the
  /// category. Confirm saves; cancel is a no-op (the switch reverts because
  /// provider state was never changed).
  Future<void> _onToggleOff(Category category) async {
    final int count;
    try {
      count = await ownTrailCount(ref, isSubcategory: false, id: category.id);
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
            .read(categoryPreferenceProvider.notifier)
            .upsert(category.id, false),
      );
      return;
    }

    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.settings_categories_confirm_disable_title),
        content: Text(l10n.settings_categories_confirm_disable_body(count)),
        actions: [
          TextButton(
            onPressed: () => _viewOwnTrails(dialogContext, category),
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
          .read(categoryPreferenceProvider.notifier)
          .upsert(category.id, false),
    );
  }

  /// Dismisses the confirm dialog and pushes the user's OWN profile trail list
  /// pre-filtered to [category]. Resolves the own handle WITH the `@` sigil
  /// exactly as `profile_screen.dart:38` and Plan 01's `own_trail_count.dart`
  /// do — every self `/profile/{handle}/...` call site uses the `@`-prefixed
  /// handle. Navigating here does NOT save or disable; the user returns to an
  /// unchanged switch.
  Future<void> _viewOwnTrails(
    BuildContext dialogContext,
    Category category,
  ) async {
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
    // with the same `@`-prefixed handle. Await the provider's future first —
    // updateFilter is a no-op while the provider has no value yet (e.g. first
    // visit this session), which would otherwise silently drop the pre-filter.
    final filterNotifier = ref.read(
      trailFilterProvider('profile_trail_$handle').notifier,
    );
    await ref.read(trailFilterProvider('profile_trail_$handle').future);
    if (!mounted) return;
    filterNotifier.updateFilter(
      (f) => f.copyWith(category: [category], subcategory: const []),
    );

    Navigator.of(dialogContext).pop(false);
    if (!context.mounted) return;
    context.push('/profile/$handle/trails');
  }
}
