# Phase 12: Settings Categories Screen - Pattern Map

**Mapped:** 2026-07-01
**Files analyzed:** 6 (2 new screens, 1 new/optional util, 3 modified)
**Analogs found:** 6 / 6 (every capability except the reorder gesture has an in-repo analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `app/lib/routes/settings_categories_screen.dart` (NEW) | route/screen | request-response (toggle) + event-driven (drag reorder) | `app/lib/routes/settings_notifications_screen.dart` + `settings_account_screen.dart` | role-match (no reorder analog exists) |
| `app/lib/routes/settings_subcategories_screen.dart` (NEW) | route/screen | request-response + event-driven | `settings_categories_screen.dart` (sibling) / `settings_notifications_screen.dart` | role-match |
| `app/lib/util/category_preference_sort.dart` (NEW, optional) | utility | transform (pure sort) | `web/src/lib/util/category_util.ts` (cross-lang port) | port-only (no Dart analog) |
| `app/lib/routes/settings_screen.dart` (MODIFY) | route/screen | request-response (navigation) | itself — existing tiles | exact |
| `app/lib/provider/router_provider.dart` (MODIFY) | provider/route-config | n/a | itself — existing `/settings/*` sub-routes | exact |
| `app/lib/provider/category_preference_provider.dart` (MODIFY) | provider/store | CRUD (add reorder POST) | itself — existing `upsert` + `subcategory_preference_provider.dart` | exact |
| `app/lib/provider/subcategory_preference_provider.dart` (MODIFY) | provider/store | CRUD (add reorder POST) | itself — existing `upsert` | exact |

**Note:** No `ReorderableListView` exists anywhere in `app/lib`. The reorder gesture is the one genuinely new pattern — see "No Analog Found" and rely on RESEARCH.md Pattern 1 for it.

## Pattern Assignments

### `app/lib/routes/settings_categories_screen.dart` (screen; toggle + reorder)

Must be a `ConsumerStatefulWidget` (needs local `_orderedIds` drag working copy + `mounted` guards after awaits). The notifications screen is `ConsumerWidget` — upgrade to Stateful, keeping its save/toast/section-header patterns.

**Analog: `app/lib/routes/settings_notifications_screen.dart`**

Imports pattern (lines 1-8):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/toast_provider.dart';
```

Optimistic save + error-toast pattern — copy verbatim (lines 16-34). No success toast (D-08):
```dart
Future<void> _save(WidgetRef ref, AppLocalizations l10n, /* op */) async {
  try {
    await ref.read(settingsProvider.notifier).saveToServer(updated);
  } catch (_) {
    ref.read(toastProvider.notifier).add(ToastMessage(
      type: ToastType.error,
      icon: FontAwesomeIcons.circleExclamation,
      text: l10n.error_saving_settings,
    ));
  }
}
```

`activeThumbColor` derivation for the `Switch` — copy exactly (lines 90-93). Note: `activeColor` is deprecated; this repo uses `activeThumbColor`:
```dart
final colorScheme = Theme.of(context).colorScheme;
final activeColor = Theme.of(context).brightness == Brightness.dark
    ? colorScheme.onSurface
    : colorScheme.primary;
```

Scaffold + AppBar with back button — copy (lines 154-162). Reuse for both screens (subcategory AppBar title = parent category `displayName(locale)` per D-06):
```dart
Scaffold(
  appBar: AppBar(
    leading: IconButton(icon: const Icon(Icons.arrow_back), onPressed: () => context.pop()),
    title: Text(l10n.categories),
  ),
  body: /* AsyncLoader-wrapped ReorderableListView */,
);
```

**Analog for the confirm dialog: `app/lib/routes/settings_account_screen.dart`** — `showDialog<bool>` + `AlertDialog` returning a bool, with `mounted` guards after every await (lines 74-97). This is the SETCAT-11 own-trail confirm pattern:
```dart
final confirmed = await showDialog<bool>(
  context: context,
  builder: (ctx) => AlertDialog(
    title: Text(/* new key */),
    content: Text(/* "{count} of your trails use this…" + view-trails action */),
    actions: [
      TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: Text(l10n.cancel)),
      TextButton(onPressed: () => Navigator.of(ctx).pop(true), child: Text(/* Disable anyway */)),
    ],
  ),
);
if (confirmed != true) return; // cancel → switch reverts (provider state never changed, D-10)
if (!context.mounted) return;
```
Mounted-guard convention (account screen lines 46, 97, 102): `if (!context.mounted) return;` after each await. In a `State`, use `if (!mounted) return;` (Pitfall 5).

**Loading wrapper (D-14): `app/lib/components/async_loader.dart`** — `AsyncLoader<T>` requires `asyncValue`, `mockData`, and `builder`. Provide a mock list for the skeleton:
```dart
AsyncLoader<List<Category>>(
  asyncValue: /* combined AsyncValue */,
  mockData: const [],
  builder: (categories) => ReorderableListView.builder(/* ... */),
)
```

**Reorder gesture:** No in-repo analog. Use RESEARCH.md Pattern 1 (`ReorderableListView.builder(buildDefaultDragHandles: false)` + `ReorderableDragStartListener` + `if (newIndex > oldIndex) newIndex -= 1`). Keep a local `List<String> _orderedIds` snapshot; revert + toast on POST failure (D-04, Pitfall 2).

---

### `app/lib/routes/settings_subcategories_screen.dart` (screen; toggle + reorder)

**Analog:** the sibling `settings_categories_screen.dart` (same structure per D-06). Differences: AppBar title = parent category name; rows are leaf (no body-tap navigation); filter subcategories to `subcategory.category == parent.id`; empty state when none (D-07); reorder POST payload includes the parent `category` id (SETCAT-10). Receives the `Category` via go_router `extra:`.

---

### `app/lib/provider/category_preference_provider.dart` (provider; add `reorder`)

**Analog:** its own existing `upsert` method (lines 43-49) — same `apiProvider` call + `invalidateSelf()` shape. Never send `user` (server injects from session, Security V4):
```dart
Future<void> upsert(String categoryId, bool visible) async {
  await ref.read(apiProvider).put(
    '/user-category-preference',
    data: {'category': categoryId, 'visible': visible},
  );
  ref.invalidateSelf();
}
```
Add (payload `{categories: [...]}` verified from server Zod schema):
```dart
Future<void> reorder(List<String> orderedCategoryIds) async {
  await ref.read(apiProvider).post(
    '/user-category-preference/reorder',
    data: {'categories': orderedCategoryIds},
  );
  ref.invalidateSelf();
}
```
Regenerate `category_preference_provider.g.dart` via `dart run build_runner build`.

---

### `app/lib/provider/subcategory_preference_provider.dart` (provider; add `reorder`)

**Analog:** its own `upsert` (lines 43-49). Add (payload `{category, subcategories: [...]}` verified):
```dart
Future<void> reorder(String categoryId, List<String> orderedSubcategoryIds) async {
  await ref.read(apiProvider).post(
    '/user-subcategory-preference/reorder',
    data: {'category': categoryId, 'subcategories': orderedSubcategoryIds},
  );
  ref.invalidateSelf();
}
```

---

### `app/lib/routes/settings_screen.dart` (add "Categories" tile — SETCAT-01)

**Analog:** the existing tiles in this same file (lines 28-51). Copy one and swap icon/label/route:
```dart
ListTile(
  leading: const FaIcon(FontAwesomeIcons.layerGroup, size: 18), // or FontAwesomeIcons.tag
  title: Text(l10n.categories),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/categories'),
),
```

---

### `app/lib/provider/router_provider.dart` (register routes — SETCAT-02)

**Analog:** the existing `/settings` GoRoute + children (lines 169-193). Add a `categories` child with a nested `subcategories` route receiving `state.extra as Category`:
```dart
GoRoute(
  path: 'categories',
  builder: (context, state) => const SettingsCategoriesScreen(),
  routes: [
    GoRoute(
      path: 'subcategories',
      builder: (context, state) =>
          SettingsSubcategoriesScreen(category: state.extra as Category),
    ),
  ],
),
```

---

### `app/lib/util/category_preference_sort.dart` (NEW, optional — port)

No Dart analog. Port `sortedCategoriesByPreference` / `sortedSubcategoriesByPreference` from `web/src/lib/util/category_util.ts` per RESEARCH.md Pattern 4. Keep it out of the provider to preserve the Phase-10 read-only-provider boundary. Use `displayName(locale)` (from `category.dart:40` / `subcategory.dart:32`) for the alphabetical tie-break, and `preferenceFor(id)?.visible != false` for visibility (Pitfall 4).

## Shared Patterns

### Error feedback (toast)
**Source:** `app/lib/routes/settings_notifications_screen.dart` lines 16-34 (`ToastMessage(type: ToastType.error, icon: FontAwesomeIcons.circleExclamation, text: l10n.error_saving_settings)`)
**Apply to:** both new screens — visibility toggle failures (D-08) and reorder failures after revert (D-04).

### Access control on preference writes
**Source:** `category_preference_provider.dart` / `subcategory_preference_provider.dart` `upsert` — client never sends `user`; server injects from session.
**Apply to:** both new `reorder` methods (must also omit `user`).

### Own-trail count (SETCAT-11)
**Source:** `app/lib/provider/profile/profile_trails_provider.dart` lines 94-116 — `POST /profile/{handle}/trails` call shape; author enforced server-side. Read count from response (`totalHits ?? estimatedTotalHits ?? 0`; note this file reads `totalPages`/`hits`, so confirm the count field at execution — Assumption A1).
```dart
final response = await api.post('/profile/$handle/trails', data: {
  'q': '',
  'options': {'filter': "category_id IN ['$id']", 'hitsPerPage': 1, 'page': 1},
});
```
**Apply to:** OFF-toggle attempts on both screens; fetched lazily, never preloaded (D-12). Field names `category_id` / `subcategory_id` per `trail.dart` `TrailFilter.toFilterText` (A2).

### Mounted guards
**Source:** `settings_account_screen.dart` lines 46, 97, 102 — `if (!context.mounted) return;` after each await.
**Apply to:** both screens (they are `ConsumerStatefulWidget`; use `if (!mounted) return;` in State methods) around dialog/save awaits (Pitfall 5).

### Category/subcategory name + icon
**Source:** `app/lib/util/category_icon_util.dart` — `categoryFilterAvatar(Category)` (line 72), `subcategoryFilterAvatar(...)` (line 82), `trailCategoryIcon(...)` (line 31); `displayName(Locale?)` on `category.dart` / `subcategory.dart`.
**Apply to:** row leading icon + label on both screens.

### Loading state
**Source:** `app/lib/components/async_loader.dart` — `AsyncLoader<T>(asyncValue:, mockData:, builder:)`.
**Apply to:** both screens (D-14).

## No Analog Found

| File / Capability | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Reorder gesture in both screens | screen | event-driven | No `ReorderableListView` exists anywhere in `app/lib`. Build from Flutter SDK per RESEARCH.md Pattern 1 (custom drag handle, `onReorder` index-shift fix, local `_orderedIds` optimistic copy). |
| `category_preference_sort.dart` | utility | transform | No Dart sort helper exists; cross-language port from `web/src/lib/util/category_util.ts`. |

Planner should use RESEARCH.md (Pattern 1, Pattern 4, Pitfalls 1-3) as the pattern source for these two, since no in-repo Dart precedent exists.

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/provider/`, `app/lib/components/`, `app/lib/util/`, `app/lib/models/`
**Files read for extraction:** `settings_notifications_screen.dart`, `settings_screen.dart`, `settings_account_screen.dart`, `category_preference_provider.dart`, `subcategory_preference_provider.dart`, `profile/profile_trails_provider.dart`, `router_provider.dart`, `async_loader.dart`, `category_icon_util.dart` (grep), `category.dart` / `subcategory.dart` (grep)
**Pattern extraction date:** 2026-07-01
