import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/models/category_preference.dart';
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/models/subcategory_preference.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/components/category/category_icon.dart';
import 'package:wanderer/util/category/preference_sort.dart';
import 'package:wanderer/actions/guard_online.dart';
import 'package:wanderer/util/category/own_trail_count.dart';

/// The leaf screen reached by tapping a
/// category row. It shows a parent category's subcategories with priority sort,
/// per-row visibility toggle (auto-save, error-toast-only), drag-handle reorder
/// scoped to the parent category, an empty state for categories with no
/// subcategories, and the own-trail confirm-before-disable dialog whose "View
/// trails" action opens the user's own profile trail list pre-filtered to the
/// subcategory.
///
/// Mirrors [SettingsCategoriesScreen] but takes a required parent
/// [Category] (passed via go_router `extra` in Plan 04) and scopes its list +
/// reorder to that category. A `ConsumerStatefulWidget` because it holds the
/// local `_orderedIds` drag working copy (never rendered from the re-sorting
/// provider mid-drag) and needs `mounted` guards after async saves.
///
/// Parent-category visibility cascades into every row (quick-260702-ere,
/// mirroring web `settings/categories/+page.svelte`): when the parent category
/// is disabled, each subcategory Switch renders OFF and non-interactive, the
/// row dims to 0.6 opacity, and drag-reordering is disabled. This is
/// presentation-only — the parent's state never mutates a subcategory's stored
/// `visible` value, so re-enabling the parent restores each saved preference.
class SettingsSubcategoriesScreen extends ConsumerStatefulWidget {
  const SettingsSubcategoriesScreen({super.key, required this.category});

  /// The parent category whose subcategories this screen configures. Its
  /// locale-resolved name is the AppBar title.
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

  /// True for the entire drag→`reorder()` round-trip: set in `onReorderStart`
  /// and cleared only after the async `reorder()` network call resolves
  /// (success or caught failure) inside `_onReorder`. While true, `build()`
  /// must not reseed `_orderedIds` from the provider-derived sort. This covers
  /// two cases:
  ///   1. The non-optimistic snap-back — clearing on drag-end (as before) let
  ///      the next `build()` reseed `_orderedIds` from still-stale provider
  ///      data before the server responded, snapping the dropped row back to
  ///      its original slot until the provider re-emitted.
  ///   2. Mid-drag protection — an unrelated `ref.watch` dependency
  ///      change mid-drag (e.g. a background provider refresh) must not
  ///      silently reset the optimistic working copy.
  bool _reordering = false;

  /// Persists [op] and surfaces only an error toast on failure (no
  /// success toast; the watched provider drives optimistic UI via
  /// `invalidateSelf`).
  Future<void> _save(Future<void> Function() op) async {
    final l10n = AppLocalizations.of(context)!;
    if (!guardOnline(ref, l10n)) return;
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

  /// Re-probes connectivity and clears the cached failures on the keepAlive
  /// preference providers — see the sibling category screen for why the
  /// invalidate is required.
  Future<void> _retryOnline() async {
    final online = await ref.read(onlineStatusProvider.notifier).refresh();
    if (!online) return;
    ref.invalidate(subcategoryPreferenceProvider);
    ref.invalidate(categoryPreferenceProvider);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context);

    // Offline the preference providers have no local cache to serve, so show
    // the offline state instead of a skeleton that resolves into an error.
    // Returns before those providers are watched so no doomed fetch starts.
    if (!ref.watch(onlineStatusProvider)) {
      return Scaffold(
        appBar: AppBar(
          leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            onPressed: () => context.pop(),
          ),
          title: Text(widget.category.displayName(locale)),
        ),
        body: WandererOfflineState(
          title: l10n.offline_title,
          body: l10n.offline_categories_body,
          retryLabel: l10n.offline_try_again,
          onRetry: _retryOnline,
        ),
      );
    }

    // subcategoryProvider is a synchronous List; only the preference providers
    // are async, so AsyncLoader wraps just the prefs loads.
    final subcategories = ref.watch(subcategoryProvider);
    final prefsAsync = ref.watch(subcategoryPreferenceProvider);
    // Parent-category preferences drive the visibility cascade (read-only here).
    final categoryPrefsAsync = ref.watch(categoryPreferenceProvider);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    // Filter to this parent category's subcategories.
    final filtered = subcategories
        .where((s) => s.category == widget.category.id)
        .toList();

    // Combine the two async values into a single record so AsyncLoader renders
    // one skeleton for both the subcategory prefs and the parent category prefs
    // (error-first, then loading, then data), mirroring the sibling category
    // screen's `combined` record. Prefer whichever data each sub-provider
    // still carries (Riverpod keeps `.value` populated through a seamless
    // refresh after `invalidateSelf()`) over collapsing to a bare
    // `.loading()` — a bare loading state has no previous value, which
    // forced AsyncLoader's fallback to empty mockData and produced a
    // visible flash on every toggle/reorder save.
    final combined = prefsAsync.hasError
        ? AsyncValue<
            ({
              List<SubcategoryPreference> prefs,
              List<CategoryPreference> categoryPrefs,
            })
          >.error(prefsAsync.error!, prefsAsync.stackTrace!)
        : categoryPrefsAsync.hasError
        ? AsyncValue<
            ({
              List<SubcategoryPreference> prefs,
              List<CategoryPreference> categoryPrefs,
            })
          >.error(categoryPrefsAsync.error!, categoryPrefsAsync.stackTrace!)
        : prefsAsync.hasValue && categoryPrefsAsync.hasValue
        ? AsyncValue<
            ({
              List<SubcategoryPreference> prefs,
              List<CategoryPreference> categoryPrefs,
            })
          >.data((
            prefs: prefsAsync.value ?? const [],
            categoryPrefs: categoryPrefsAsync.value ?? const [],
          ))
        : const AsyncValue<
            ({
              List<SubcategoryPreference> prefs,
              List<CategoryPreference> categoryPrefs,
            })
          >.loading();

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        // Parent category's localized name.
        title: Text(widget.category.displayName(locale)),
      ),
      body: Column(
        children: [
          Expanded(
            child:
                AsyncLoader<
                  ({
                    List<SubcategoryPreference> prefs,
                    List<CategoryPreference> categoryPrefs,
                  })
                >(
                  asyncValue: combined,
                  mockData: const (prefs: [], categoryPrefs: []),
                  builder: (data) {
                    // Empty state — the screen stays reachable even with no
                    // subcategories.
                    if (filtered.isEmpty) {
                      return _buildEmptyState(l10n);
                    }

                    // Parent visibility, computed once per build from the
                    // read-only category prefs. Named `catVisible` to avoid
                    // shadowing the `categoryVisible` helper. Drives the row
                    // cascade below.
                    final catVisible = categoryVisible(
                      widget.category.id,
                      data.categoryPrefs,
                    );

                    final sorted = sortedSubcategoriesByPreference(
                      filtered,
                      data.prefs,
                      locale,
                    );
                    // Seed the reorder working copy from the current sorted
                    // order. On the idle path this simply mirrors `sorted`; a
                    // drag mutates it optimistically. Skip reseeding for the
                    // whole reorder round-trip (`_reordering`): while a drag
                    // is in progress OR the async `reorder()` is still in
                    // flight, a rebuild must not clobber the optimistic
                    // working copy — reseeding from the not-yet-updated
                    // provider would snap the dropped row back to its old
                    // slot.
                    if (!_reordering) {
                      _orderedIds = sorted.map((s) => s.id).toList();
                    }

                    return _buildList(
                      sorted,
                      data.prefs,
                      locale,
                      activeColor,
                      catVisible,
                    );
                  },
                ),
          ),
        ],
      ),
    );
  }

  /// Empty-state copy for a category with no subcategories. Reuses the
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
  /// mid-drag). `buildDefaultDragHandles: true` (the default) wraps the whole
  /// row in Flutter's built-in `ReorderableDelayedDragStartListener` on
  /// iOS/Android — no explicit handle icon is shown; a long-press anywhere on
  /// the row starts the drag, while quick taps still reach the row's `Switch`.
  /// Reordering stays available even when [categoryOn] is false — only the
  /// per-row visibility toggle cascades on the parent's state.
  ///
  /// [categoryOn] is the parent-category visibility: when false every row's
  /// Switch dims/disables (quick-260702-ere cascade), but drag-reordering is
  /// unaffected.
  Widget _buildList(
    List<Subcategory> sorted,
    List<SubcategoryPreference> prefs,
    Locale locale,
    Color activeColor,
    bool categoryOn,
  ) {
    final byId = {for (final s in sorted) s.id: s};
    return ReorderableListView.builder(
      itemCount: _orderedIds.length,
      itemBuilder: (context, index) {
        final sub = byId[_orderedIds[index]];
        if (sub == null) return const SizedBox.shrink();
        return _buildRow(sub, index, prefs, locale, activeColor, categoryOn);
      },
      // The index-shift `if (newIndex > oldIndex) newIndex -= 1` inside
      // _onReorder is the canonical onReorder contract; onReorderItem is not
      // used so that shift stays explicit and testable (Pitfall 1).
      // ignore: deprecated_member_use
      onReorder: (oldIndex, newIndex) => _onReorder(oldIndex, newIndex),
      // Guard the working copy for the full drag→reorder() round-trip. It is
      // cleared inside `_onReorder` once the network call resolves — NOT on
      // drag-end, which fires synchronously before `await reorder()` returns
      // and would let the next build reseed stale data (the snap-back bug).
      onReorderStart: (_) => setState(() => _reordering = true),
    );
  }

  /// Optimistically reorders `_orderedIds`, POSTs the parent category id + the
  /// full ordered subcategory id list, and reverts to the
  /// pre-drag snapshot + error toast on failure.
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
    if (!guardOnline(ref, l10n)) {
      // Mirror the catch path below: nothing was persisted, so revert the
      // optimistic reorder and clear the in-flight guard — otherwise the
      // list would stay stuck in its reordering state (never reseeding from
      // the provider again) while displaying an order the server never saw.
      setState(() {
        _orderedIds = snapshot;
        _reordering = false;
      });
      return;
    }
    try {
      // Parent category id FIRST (payload
      // {category, subcategories}).
      await ref
          .read(subcategoryPreferenceProvider.notifier)
          .reorder(widget.category.id, _orderedIds);
      // reorder() calls invalidateSelf() internally but does not wait for the
      // resulting refetch — clearing the guard here would let a rebuild land
      // with _reordering already false while prefsAsync.value still carries
      // the pre-reorder order (Riverpod keeps the previous value through the
      // loading state), reseeding _orderedIds back to the stale order for one
      // frame before the refetch resolves and snaps it forward again (the
      // reported flash). Await the refetch itself so the guard only drops
      // once the provider's data actually reflects the new order.
      if (!mounted) return;
      await ref.read(subcategoryPreferenceProvider.future);
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

  /// A single subcategory row: leading avatar + name + trailing visibility
  /// Switch, styled identically to the sibling category screen's row —
  /// same padding/spacing/bold-name treatment, minus the subcategory chips
  /// (this screen has no nested children to show). Reordering starts on a
  /// long-press anywhere on the row (see `_buildList`'s
  /// `buildDefaultDragHandles`), so there is no explicit drag-handle widget
  /// here. Rows are LEAF — no body-tap navigation.
  ///
  /// [categoryOn] is the parent-category visibility (quick-260702-ere
  /// cascade, mirroring web `disabled={... || !categoryVisible}` /
  /// `class:opacity-60={!categoryVisible}`): when false, the Switch shows OFF
  /// and non-interactive (onChanged null) and the whole row dims to 0.6.
  /// Drag-reordering is NOT gated on [categoryOn] — it stays available
  /// regardless of the parent's visibility.
  Widget _buildRow(
    Subcategory sub,
    int index,
    List<SubcategoryPreference> prefs,
    Locale locale,
    Color activeColor,
    bool categoryOn,
  ) {
    // Presentation-only: derive display state from read data. When the parent
    // is off, the switch reads OFF regardless of the subcategory's own stored
    // `visible` — but that stored value is never written, so re-enabling the
    // parent restores it for free.
    final effectiveVisible = categoryOn && subcategoryVisible(sub.id, prefs);
    return Opacity(
      key: ValueKey(sub.id),
      opacity: categoryOn ? 1.0 : 0.6,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0, vertical: 8),
        child: Row(
          children: [
            subcategoryFilterAvatar(sub, widget.category, locale),
            const SizedBox(width: 24),
            Expanded(
              child: Text(
                sub.displayName(locale),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Switch(
              value: effectiveVisible,
              activeThumbColor: activeColor,
              // null renders the Switch disabled/non-interactive when the
              // parent category is off (Flutter's standard mechanism).
              onChanged: categoryOn ? (v) => _onToggle(sub, v) : null,
            ),
          ],
        ),
      ),
    );
  }

  /// ON saves immediately with no check; OFF routes to the own-trail
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

  /// OFF path — own-trail confirm-before-disable. Lazily counts the user's
  /// own trails in [sub] (never preloaded); if none, saves directly, otherwise shows a confirm dialog whose
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
        title: Text(l10n.settings_categories_confirm_disable_subcategory_title),
        content: Text(l10n.settings_categories_confirm_disable_body(count)),
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
  Future<void> _viewOwnTrails(
    BuildContext dialogContext,
    Subcategory sub,
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
    // with the same `@`-prefixed handle; TrailFilter.subcategory is a
    // List<Subcategory> — pass the current sub. Await the provider's future
    // first — updateFilter is a no-op while the provider has no value yet
    // (e.g. first visit this session), which would otherwise silently drop
    // the pre-filter.
    final filterNotifier = ref.read(
      trailFilterProvider('profile_trail_$handle').notifier,
    );
    await ref.read(trailFilterProvider('profile_trail_$handle').future);
    if (!mounted) return;
    filterNotifier.updateFilter(
      (f) => f.copyWith(category: const [], subcategory: [sub]),
    );

    if (!dialogContext.mounted) return;
    Navigator.of(dialogContext).pop(false);
    if (!context.mounted) return;
    context.push('/profile/$handle/trails');
  }
}
