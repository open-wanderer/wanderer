# Phase 12: Settings Categories Screen - Research

**Researched:** 2026-07-01
**Domain:** Flutter mobile UI (Material 3 + Riverpod) — reorderable settings lists, optimistic auto-save, guarded-disable confirm dialog
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Reordering**
- **D-01:** Category rows use `ReorderableListView.builder` with a dedicated drag-handle icon (`Icons.drag_handle`) — not whole-row drag. Ruled out up/down arrow buttons.
- **D-02:** Drag reorder posts to `POST /user-category-preference/reorder` with the full ordered list of category IDs.
- **D-03:** Subcategories are reorderable too (scope amendment) — but only inside `SettingsSubcategoriesScreen`, via the same drag-handle `ReorderableListView` pattern, posting to `POST /user-subcategory-preference/reorder` (endpoint already exists server-side, confirmed in `web/src/routes/api/v1/user-subcategory-preference/reorder/+server.ts`).
- **D-04:** On reorder failure (either endpoint): revert the list to its pre-drag order and show the same error-toast pattern used elsewhere, rather than leaving the list in an unconfirmed state.

**Navigation / Row Layout (redesigned from ExpansionTile)**
- **D-05:** No inline `ExpansionTile`. Category rows are: drag handle, icon, locale-resolved name, visibility `Switch`. Tapping the row body (not the switch, not the drag handle) navigates to `SettingsSubcategoriesScreen`, passing the category.
- **D-06:** `SettingsSubcategoriesScreen` reuses the same list pattern as the category screen: AppBar title = parent category's locale-resolved name; body = `ReorderableListView` of subcategories, each with drag handle + visibility `Switch` (no further nesting).
- **D-07:** Categories with zero subcategories are still tappable and navigate to `SettingsSubcategoriesScreen`, which will render an empty state (no dedicated "disable navigation" case was requested).

**Save / Error Feedback**
- **D-08:** Visibility toggles (category and subcategory) reuse the existing settings pattern: try/catch around the provider save call, error toast via `toastProvider`/`ToastMessage(type: error)` on failure (mirrors `settings_notifications_screen.dart`), no success toast. Optimistic UI via the watched provider.
- **D-09:** Reorder failures revert + toast per D-04.

**Own-Trail Disable Confirmation (new — mirrors web)**
- **D-10:** Turning a visibility switch **OFF** (category or subcategory) first checks whether the user has their own trails using that category/subcategory. If count > 0, show a confirm dialog: "{count} of your trails use this category/subcategory" + a link/action to view those trails, "disable anyway" / "cancel". Cancelling reverts the switch to on (no-op). Confirming proceeds with the normal save (D-08).
- **D-11:** Turning a switch **ON** never triggers this check — only OFF transitions.
- **D-12:** The own-trail count is **not preloaded**; it's fetched at the moment of a toggle-off attempt by querying the trail list API filtered by `author = self AND category/subcategory = id` (reuse `TrailFilter` + the existing trail search endpoint already used by `profile_trails_provider.dart` / Phase 11's `TrailFilter.subcategory`). There is no dedicated count endpoint — web computes this server-side in `+page.server.ts`; mobile must call the trail list API directly.
- **D-13:** Plugin-mapping warnings (web's `confirm-disable-*-active-plugin-mappings` messaging) are explicitly **not ported** — Integrations/plugin system is out of scope for mobile settings v1 (per PROJECT.md).

**Loading State**
- **D-14:** Use the existing `AsyncLoader` component (`app/lib/components/async_loader.dart`) to wrap the category/preference-loading state on both screens — not a bespoke `CircularProgressIndicator` or skeleton.

**Empty state (all categories hidden)**
- **D-15:** No special warning if a user ends up with zero visible categories — allowed silently, consistent with no cross-field validation elsewhere in the settings suite.

### Claude's Discretion
- Exact confirm-dialog widget (`AlertDialog` vs a reusable app-wide confirm component, if one exists) — pick whatever matches existing app conventions found during research/planning.
- Exact drag-handle icon size/spacing and row padding — follow existing settings screen visual conventions (16/16/16/8 section padding, etc. from `settings_notifications_screen.dart`).

### Deferred Ideas (OUT OF SCOPE)
None — the scope changes discussed (subcategory screen + reorder, own-trail confirm dialog) were folded into this phase's requirements rather than deferred.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETCAT-01 | SettingsScreen gains a "Categories" list tile navigating to `/settings/categories` | Add a `ListTile` mirroring the existing tiles in `settings_screen.dart` (FaIcon size 18 + chevron + `context.push`). `l10n.categories` exists (`app_en.arb:55`). See Pattern 6. |
| SETCAT-02 | go_router registers `/settings/categories` route → `SettingsCategoriesScreen` | Add a child `GoRoute` under the existing `/settings` route in `router_provider.dart` (see Pattern 6). SettingsSubcategoriesScreen pushed via `context.push` with `extra:` (see Pattern 2). |
| SETCAT-06 | List categories sorted by priority asc, alphabetical (locale name) for ties; each row shows icon + locale-resolved name | Port `sortedCategoriesByPreference()` (Pattern 4). Providers `categoryProvider` + `categoryPreferenceProvider` already exist. `CategoryDisplay.displayName(locale)` + `category_icon_util.dart` resolve name/icon. |
| SETCAT-07 | Each row has a visibility switch; toggling PUTs `/user-category-preference` and auto-saves | `CategoryPreferenceNotifier.upsert(id, visible)` already exists and invalidates self (optimistic via `invalidateSelf`). Wrap in try/catch + toast (Pattern 3). |
| SETCAT-08 | Tapping a row navigates to `SettingsSubcategoriesScreen` with per-subcategory switches | New screen; body tap uses `onTap` on the row content, guarded from switch/handle (Pattern 2). `SubcategoryPreferenceNotifier.upsert` exists. |
| SETCAT-09 | Category screen uses `ReorderableListView` + drag handle; drop POSTs `/user-category-preference/reorder` with ordered category IDs | Payload shape `{categories: [id,...]}` (verified in schema). Needs a new `reorder` method on `CategoryPreferenceNotifier`. Pattern 1 + Pitfall 1. |
| SETCAT-10 | Subcategory screen uses same pattern; drop POSTs `/user-subcategory-preference/reorder` with ordered subcategory IDs scoped to parent | Payload shape `{category: id, subcategories: [id,...]}` (verified in schema). Needs a new `reorder` method on `SubcategoryPreferenceNotifier`. |
| SETCAT-11 | Turning OFF a switch when user has own trails in it shows confirm dialog (count + view link); confirm saves, cancel reverts. ON never checks. No plugin warnings. | Own-trail count via `POST /profile/{ownHandle}/trails` filtered by category/subcategory, read `totalHits` (Pattern 5). `showDialog<bool>` + `AlertDialog` pattern exists in `settings_account_screen.dart`. |
</phase_requirements>

## Summary

This is a **UI-only Flutter phase** built entirely on first-party Material widgets and existing Phase 10 providers. Nothing new is installed. The single genuinely new pattern is `ReorderableListView.builder` with a **custom drag handle** (`ReorderableDragStartListener` + `buildDefaultDragHandles: false`) — no prior art exists in `app/lib`, so it must be built from Flutter's standard widget. Everything else (optimistic toggle + toast, `AsyncLoader`, `AlertDialog` confirm, go_router sub-route wiring, locale-resolved names, category icon resolution) has direct in-repo precedent that should be copied rather than reinvented.

Two provider gaps must be filled: the Phase 10 `CategoryPreferenceNotifier` and `SubcategoryPreferenceNotifier` currently expose only `build()` (GET) and `upsert(id, visible)` (PUT). This phase must **add a `reorder(List<String> ids)` method to each** that POSTs to the `/reorder` endpoints and invalidates self. The reorder payload shapes are verified from the server Zod schemas: category = `{categories: [...ids]}`, subcategory = `{category: parentId, subcategories: [...ids]}`.

The own-trail count check (SETCAT-11) has no dedicated count endpoint. Web computes it server-side by listing the author's full trail set. Mobile must call the existing trail-search API — the cleanest path is `POST /profile/{ownHandle}/trails` (author is server-enforced, so no client-side author filter is needed) with a `filter` scoped to the single category or subcategory ID, reading Meilisearch's `totalHits` from the response. This is fetched lazily only on an OFF-toggle attempt (D-12), never preloaded.

**Primary recommendation:** Build two `ConsumerStatefulWidget` screens. Use `ReorderableListView.builder(buildDefaultDragHandles: false, ...)` with a per-row `ReorderableDragStartListener(index: i, child: Icon(Icons.drag_handle))`. Apply the `onReorder` index-shift fix (`if (newIndex > oldIndex) newIndex -= 1`). Reorder optimistically on a local ordered-ID list, POST, and on failure revert that local list + toast. Toggle via the existing `upsert` + toast pattern; intercept OFF-transitions to run the own-trail count check before saving.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Category/subcategory list rendering, drag reorder gesture, switch state | Flutter UI (`app/lib/routes/*.dart`) | — | Pure presentation; consumes providers |
| Sorting by priority + alphabetical tie-break | Flutter UI (util fn) | — | Client-side sort mirroring web `sortedCategoriesByPreference` |
| Preference persistence (visibility upsert, reorder) | Riverpod provider (`app/lib/provider/*_preference_provider.dart`) | API/Backend | Providers own the HTTP call + cache invalidation; server persists |
| Own-trail count lookup | Riverpod / direct API call | API/Backend (Meilisearch) | No count endpoint; reuse trail-search, server enforces author scope |
| Route registration | go_router (`router_provider.dart`) | — | Client-side navigation only |
| Locale name / icon resolution | Flutter model extensions + `category_icon_util.dart` | — | Already implemented in Phase 10/11 |
| Server-side ordering/persistence, auth injection of `user` | API/Backend (PocketBase) | — | Endpoints already exist; client never sends `user` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Flutter Material (SDK) | 3.44.2 stable / Dart SDK ^3.11.5 | `ReorderableListView`, `SwitchListTile`/`Switch`, `ListTile`, `AlertDialog`, `AppBar`, `Scaffold` | First-party; `ReorderableListView` is the canonical Flutter reorder widget [VERIFIED: flutter --version = 3.44.2, api.flutter.dev] |
| flutter_riverpod | 3.3.1 | State management, provider watch/read/invalidate | Established app-wide; Phase 10 providers are Riverpod [CITED: CLAUDE.md] |
| riverpod_annotation | (codegen) | `@Riverpod` notifier classes, `.g.dart` generation | Existing preference providers use `@Riverpod(keepAlive: true)` [VERIFIED: category_preference_provider.dart] |
| go_router | 17.2.1 | Route registration + `context.push`/`context.pop` navigation | App's router; `/settings/*` sub-routes already registered here [CITED: CLAUDE.md, router_provider.dart] |
| font_awesome_flutter | ^11.0.0 | `FaIcon` category/tile glyphs | Already a dependency; used by every settings tile [VERIFIED: pubspec.yaml, settings_screen.dart] |
| dio | 5.9.2 | HTTP client (via `apiProvider`) | App's HTTP layer; preference providers already call `ref.read(apiProvider).put/get` [VERIFIED: category_preference_provider.dart] |
| skeletonizer (via AsyncLoader) | (transitive) | Loading skeleton — wrapped by `AsyncLoader` | D-14 mandates `AsyncLoader`; it internally uses `Skeletonizer` [VERIFIED: async_loader.dart] |
| intl / flutter_localizations | (SDK) | ARB-based l10n, `AppLocalizations` | 14 ARB files; `flutter gen-l10n` via `generate: true` [VERIFIED: pubspec.yaml, l10n.yaml] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| freezed_annotation | 3.1.0 | Immutable models (`CategoryPreference`, `Subcategory`, `TrailFilter`) | Models already generated in Phase 10/11; no new models required unless the planner adds a reorder-response model (not needed) [VERIFIED: category_preference.dart] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `ReorderableListView.builder` (custom handle) | `flutter_reorderable_list` / `reorderables` pub packages | Rejected — D-01 mandates the stock widget; adding a package violates the "no new packages" UI-SPEC constraint and Package Legitimacy risk. Stock widget fully supports custom handles via `ReorderableDragStartListener`. |
| `POST /profile/{handle}/trails` for own-trail count | `POST /search/trails` with explicit `author = {actorId}` filter | Both work. `/profile/{handle}/trails` is cleaner (server enforces author, no client author-filter string needed), and matches `profile_trails_provider.dart`. `/search/trails` requires building the author clause via `TrailFilter.toFilterText(actor:)`. Recommend `/profile/{handle}/trails`. |
| `SwitchListTile` for rows | `ListTile` with trailing `Switch` | `ReorderableListView` rows need a leading drag handle AND a tappable body AND a trailing switch. A `SwitchListTile`'s whole tile toggles — that conflicts with "tap body navigates". Use a `ListTile`/`Row` with an explicit trailing `Switch` so body-tap navigation and switch-toggle are separate hit targets (see Pattern 2). |

**Installation:** None. No packages added this phase.

## Package Legitimacy Audit

> Not applicable — this phase installs **zero** external packages. All widgets are first-party Flutter Material SDK; `flutter_riverpod`, `go_router`, `font_awesome_flutter`, `dio`, `freezed` are pre-existing project dependencies (`app/pubspec.yaml`) already vetted in prior phases. slopcheck not run because no install occurs.

## Architecture Patterns

### System Architecture Diagram

```
SettingsScreen ──tap "Categories" tile──▶ context.push('/settings/categories')
                                                    │
                                                    ▼
                                    SettingsCategoriesScreen (ConsumerStatefulWidget)
                                                    │
                    ┌───────────────────────────────┼───────────────────────────────┐
                    ▼                                ▼                                ▼
      watch categoryProvider (List)   watch categoryPreferenceProvider   local _orderedIds (List<String>)
                    │                                │                     (drag working copy)
                    └──────────┬─────────────────────┘                                │
                               ▼                                                       │
                   sortedCategoriesByPreference(cats, prefs, locale)  ◀───(seed)──────┘
                               │
                               ▼
              AsyncLoader ▶ ReorderableListView.builder(buildDefaultDragHandles:false)
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼ (drag handle)        ▼ (row body tap)        ▼ (trailing Switch)
ReorderableDragStartListener   context.push(           onChanged(value):
  → onReorder(old,new)         '/settings/categories/  ├─ value==true → upsert(id,true)  (D-11: no check)
  → fix newIndex               subcategories', extra:  └─ value==false →
  → optimistic reorder _ids       category)               fetchOwnTrailCount(catId)
  → POST /reorder {categories}                            ├─ count==0 → upsert(id,false)
  → on error: revert + toast                              └─ count>0 → showDialog(AlertDialog)
                                                              ├─ confirm → upsert(id,false)
                                                              └─ cancel  → revert switch (no-op)

Own-trail count:  POST /profile/{ownHandle}/trails  { q:'', options:{ filter:"category_id IN ['<id>']", hitsPerPage:1, page:1 } }
                    └─▶ response.data['totalHits']  (Meilisearch SearchResponse)

SettingsSubcategoriesScreen — same structure, filtered to subcategories where subcategory.category == parent.id,
  reorder POSTs {category: parentId, subcategories:[...]}, no body-tap navigation (leaf), empty state if none.
```

### Recommended Project Structure
```
app/lib/
├── routes/
│   ├── settings_screen.dart                  # MODIFY: add "Categories" ListTile (SETCAT-01)
│   ├── settings_categories_screen.dart       # NEW: category list + reorder + toggle (SETCAT-06/07/09/11)
│   └── settings_subcategories_screen.dart    # NEW: subcategory list + reorder + toggle (SETCAT-08/10/11)
├── provider/
│   ├── router_provider.dart                  # MODIFY: register /settings/categories (+ subcategories) (SETCAT-02)
│   ├── category_preference_provider.dart     # MODIFY: add reorder(List<String>) method (SETCAT-09)
│   └── subcategory_preference_provider.dart  # MODIFY: add reorder(String categoryId, List<String>) (SETCAT-10)
├── util/
│   └── category_preference_sort.dart         # NEW (optional): port sortedCategoriesByPreference/sortedSubcategoriesByPreference
└── i18n/*.arb (×14)                           # MODIFY: add confirm-dialog + empty-state keys, regen AppLocalizations
```

### Pattern 1: ReorderableListView with a dedicated drag handle
**What:** Reorderable list where drag starts ONLY from a handle icon, not the whole row.
**When to use:** Both category and subcategory screens (D-01, D-03).
**Example:**
```dart
// Source: https://api.flutter.dev/flutter/widgets/ReorderableDragStartListener-class.html
//         https://api.flutter.dev/flutter/material/ReorderableListView-class.html  [VERIFIED]
ReorderableListView.builder(
  buildDefaultDragHandles: false, // CRITICAL: disables whole-row / long-press drag (D-01)
  itemCount: items.length,
  itemBuilder: (context, index) {
    final item = items[index];
    return Padding(
      key: ValueKey(item.id), // REQUIRED: every child needs a stable unique key
      padding: EdgeInsets.zero,
      child: Row(
        children: [
          ReorderableDragStartListener(
            index: index,                       // drag affordance only here
            child: const Padding(
              padding: EdgeInsets.symmetric(horizontal: 8),
              child: Icon(Icons.drag_handle),   // 24px, ≥48px touch target (UI-SPEC)
            ),
          ),
          // ... icon, name (Expanded + onTap for body nav), trailing Switch ...
        ],
      ),
    );
  },
  onReorder: (oldIndex, newIndex) {
    // Index-shift fix — see Pitfall 1. Without this, moving an item DOWN lands one slot too low.
    if (newIndex > oldIndex) newIndex -= 1;
    _onReorder(oldIndex, newIndex); // optimistic local reorder + POST + revert-on-error
  },
);
```
Notes: `ReorderableDragStartListener` constructor = `{ Key? key, required Widget child, required int index, bool enabled = true }` [VERIFIED: api.flutter.dev]. Use it (not `ReorderableDelayedDragStartListener`) so the drag starts immediately on pointer-down rather than after a long press.

### Pattern 2: Separate hit targets — body tap navigates, switch toggles, handle drags
**What:** One row, three independent gesture zones (D-05).
**Why:** A `SwitchListTile` toggles on whole-tile tap and cannot host a separate "tap body → navigate" action. Compose explicitly.
**Example:**
```dart
Row(
  children: [
    ReorderableDragStartListener(index: index, child: const Icon(Icons.drag_handle)),
    Expanded(
      child: InkWell(
        onTap: () => context.push(
          '/settings/categories/subcategories', // or push the widget directly (planner's call)
          extra: category,
        ),
        child: Row(children: [
          categoryFilterAvatar(category), // from category_icon_util.dart (Phase 11)
          const SizedBox(width: 12),
          Expanded(child: Text(category.displayName(locale), overflow: TextOverflow.ellipsis)),
        ]),
      ),
    ),
    Switch(
      value: isVisible,
      activeThumbColor: activeColor, // dark? onSurface : primary — copy from settings_notifications_screen.dart
      onChanged: (v) => _onToggle(category, v), // NOT wrapped in the InkWell → no nav on toggle
    ),
  ],
)
```
Navigation to the subcategory screen: prefer `context.push('/settings/categories/subcategories', extra: category)` with a registered child route, OR `Navigator.push(MaterialPageRoute(...))`. Passing the `Category` object via `extra:` avoids a fetch-by-id round-trip (mirrors how `router_provider.dart` passes `extra` for `/trail/:id/navigate`).

### Pattern 3: Optimistic toggle + error toast (copy from settings_notifications_screen.dart)
**What:** Auto-save on switch change; provider invalidation drives optimistic UI; toast only on error.
**Example:**
```dart
// Source: app/lib/routes/settings_notifications_screen.dart  [VERIFIED in-repo]
Future<void> _save(WidgetRef ref, AppLocalizations l10n, Future<void> Function() op) async {
  try {
    await op(); // e.g. ref.read(categoryPreferenceProvider.notifier).upsert(id, visible)
  } catch (_) {
    ref.read(toastProvider.notifier).add(ToastMessage(
      type: ToastType.error,
      icon: FontAwesomeIcons.circleExclamation,
      text: l10n.error_saving_settings, // key already exists (app_en.arb:154)
    ));
  }
}
```
`CategoryPreferenceNotifier.upsert(categoryId, visible)` already calls `invalidateSelf()`, so the watched provider re-fetches and the switch reflects saved state (optimistic-via-refetch). No success toast (D-08).

### Pattern 4: Sort by priority ascending, alphabetical (locale) for ties
**What:** Port `sortedCategoriesByPreference()` / `sortedSubcategoriesByPreference()` to Dart.
**Example:**
```dart
// Source: web/src/lib/util/category_util.ts  [CITED — mirror logic exactly]
List<Category> sortedCategoriesByPreference(
  List<Category> categories, List<CategoryPreference> prefs, Locale? locale) {
  int? priorityOf(String id) =>
      prefs.where((p) => p.category == id).map((p) => p.priority).firstOrNull;
  final list = [...categories];
  list.sort((a, b) {
    final ap = priorityOf(a.id); final bp = priorityOf(b.id);
    final aPrioritized = ap != null && ap > 0;
    final bPrioritized = bp != null && bp > 0;
    if (aPrioritized && bPrioritized) return ap!.compareTo(bp!);
    if (aPrioritized) return -1;
    if (bPrioritized) return 1;
    // Tie-break: case-insensitive locale-aware name compare
    return a.displayName(locale).toLowerCase().compareTo(b.displayName(locale).toLowerCase());
  });
  return list;
}
```
Note: web uses `localeCompare(..., { sensitivity: "base" })`. Dart has no built-in locale collator in the SDK; `String.compareTo` on lowercased strings is an acceptable approximation for this settings list (matches the pragmatism already in `TrailFilter`). Do NOT add an `intl`-collation package for this.

### Pattern 5: Own-trail count via profile-trails search (no count endpoint)
**What:** Lazily fetch the count of the current user's trails using a given category/subcategory.
**Why this endpoint:** `POST /profile/{handle}/trails` enforces `author = <actor>` server-side (verified in `web/.../profile/[handle]/trails/+server.ts`), returns a raw Meilisearch `SearchResponse` including `totalHits`. No client author filter needed.
**Example:**
```dart
// Source: app/lib/provider/profile/profile_trails_provider.dart (call shape) [VERIFIED]
//         web/src/routes/api/v1/profile/[handle]/trails/+server.ts (author enforced) [CITED]
Future<int> _ownTrailCount(WidgetRef ref, {required bool isSubcategory, required String id}) async {
  final user = ref.read(authProvider).value;
  if (user == null) return 0;
  final field = isSubcategory ? 'subcategory_id' : 'category_id';
  final response = await ref.read(apiProvider).post(
    '/profile/${user.preferredUsername}/trails',
    data: {
      'q': '',
      'options': {
        'filter': "$field IN ['$id']",
        'hitsPerPage': 1, // we only need the count, not the hits
        'page': 1,
      },
    },
  );
  final data = response.data as Map<String, dynamic>;
  // Meilisearch returns totalHits (finite) or estimatedTotalHits depending on config; prefer totalHits.
  return (data['totalHits'] ?? data['estimatedTotalHits'] ?? 0) as int;
}
```
**Verify at plan time:** confirm the field names are `category_id` / `subcategory_id` (they are — same names used in `TrailFilter.toFilterText`, `trail.dart:298,303`) and that the profile-trails response surfaces `totalHits`. If the Meilisearch config only returns `estimatedTotalHits`, the fallback above handles it. The "view trails" link (SETCAT-11) should navigate to the user's own filtered trail list — reuse `/profile/{handle}/trails` UI or the trail filter; exact target is planner's discretion.

**Caveat (document but low-risk):** `/profile/{handle}/trails` applies `withTrailPreferenceMeiliFilter` (the user's own visibility preferences) server-side. Because the count is fetched BEFORE the disable is saved (D-10 fires on the OFF *attempt*), the category is still visible at query time, so its trails are counted correctly. If a planner ever moves the count fetch to *after* save, the count would wrongly drop to 0.

### Pattern 6: Settings tile + go_router sub-route (copy exact existing shape)
```dart
// Tile in settings_screen.dart (Source: settings_screen.dart lines 40-51) [VERIFIED]
ListTile(
  leading: const FaIcon(FontAwesomeIcons.layerGroup, size: 18), // or FontAwesomeIcons.tag per SETCAT-01
  title: Text(l10n.categories),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/categories'),
),

// Route in router_provider.dart, as a child of the existing '/settings' GoRoute (Source: lines 169-194):
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

### Anti-Patterns to Avoid
- **Using `SwitchListTile` for reorderable rows:** its whole-tile tap conflicts with body-tap navigation. Use explicit `Row` + `Switch` (Pattern 2).
- **Forgetting the `onReorder` index shift:** dropping items lands one slot off. Always `if (newIndex > oldIndex) newIndex -= 1` (Pitfall 1).
- **Missing `Key` on `ReorderableListView` children:** throws / breaks animations. Every child needs a stable `ValueKey(item.id)`.
- **Reordering by rebuilding from the provider mid-drag:** the provider re-sorts by priority; you must reorder a **local working list of IDs** optimistically and only re-seed it from the provider after a successful POST + invalidate. Otherwise the item snaps back before the server confirms.
- **Sending `user` in preference payloads:** server injects it from session (Security V4 / T-10-05). The existing `upsert` correctly omits it — the new `reorder` methods must too.
- **Adding a reorder pub package:** violates UI-SPEC "no new packages"; stock widget suffices.
- **Preloading own-trail counts for every row:** D-12 says fetch lazily on OFF-attempt only. Preloading is wasted network + wrong pattern.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-to-reorder list | Custom pan-gesture + AnimatedList | `ReorderableListView.builder` + `ReorderableDragStartListener` | Stock widget handles drop targets, auto-scroll, drag proxy elevation, a11y semantics |
| Loading/skeleton state | Bespoke `CircularProgressIndicator` | `AsyncLoader` (D-14) | Consistent skeleton UX; already app-standard |
| Confirm dialog | Custom overlay/modal | `showDialog<bool>` + `AlertDialog` (as in `settings_account_screen.dart`) | Correct focus trap, barrier, back-button handling, returns a bool |
| Locale-resolved names | Ad-hoc translation lookups | `CategoryDisplay.displayName(locale)` / `SubcategoryDisplay.displayName(locale)` | Phase 10 fallback chain (locale → en → raw) already implemented |
| Category/subcategory icons | Manual FA-name mapping | `category_icon_util.dart` (`categoryFilterAvatar`, `subcategoryFilterAvatar`, `trailCategoryIcon`) | Phase 11 shared util handles `fa-` prefix stripping + badge overlay + fallback |
| Error feedback | New snackbar system | `toastProvider` + `ToastMessage(type: error)` | App-wide toast, `error_saving_settings` key exists |
| Own-trail count | New count API/query builder | `POST /profile/{handle}/trails` + read `totalHits` | Server enforces author; no new endpoint; reuses proven call shape |

**Key insight:** Every capability except the reorder gesture already has a copy-paste source in `app/lib`. The phase's risk is concentrated in the one new widget (reorder + custom handle) and the two new provider methods — everything else is assembly of proven parts.

## Common Pitfalls

### Pitfall 1: `onReorder` newIndex is off-by-one when dragging downward
**What goes wrong:** Item dropped one position lower than intended.
**Why it happens:** `ReorderableListView` reports `newIndex` as if the dragged item were still in the list; after removal the target index shifts.
**How to avoid:** `if (newIndex > oldIndex) newIndex -= 1;` before mutating your list. This is the documented, universal fix. [VERIFIED: api.flutter.dev notes automatic adjustment for `onReorderItem`, but classic `onReorder` requires the manual shift]
**Warning signs:** Reorder "works" moving up but overshoots moving down.

### Pitfall 2: List snaps back to priority order mid/after drag
**What goes wrong:** After a drag the row jumps to a different position than dropped.
**Why it happens:** The list is derived from `sortedCategoriesByPreference(provider)`. On any rebuild it re-sorts by the *old* priorities until the server returns new ones.
**How to avoid:** Keep a local `List<String> _orderedIds` seeded from the sorted provider list. Mutate it optimistically in `_onReorder`, render from it, POST, and only re-seed from the provider after a successful invalidate. On failure, revert `_orderedIds` to the pre-drag snapshot + toast (D-04).
**Warning signs:** Visible flicker/snap immediately after dropping.

### Pitfall 3: Missing/duplicate keys on reorderable children
**What goes wrong:** `Every item of ReorderableListView must have a key` assertion, or wrong item animates.
**How to avoid:** `key: ValueKey(item.id)` on the top-level child widget returned by `itemBuilder`.

### Pitfall 4: Switch value fights the provider (visible defaults)
**What goes wrong:** A category with no preference row shows OFF instead of ON.
**Why it happens:** Web treats "no preference" and `visible !== false` as visible (`subcategoryVisible` / `preference?.visible !== false`). A naive `pref?.visible ?? false` inverts this.
**How to avoid:** Compute `isVisible = preferenceFor(id)?.visible != false` (i.e. visible unless explicitly false). Mirror `subcategoryVisible()` / `categoryVisible` from `category_util.ts`.

### Pitfall 5: async gap after `showDialog` / `await` without mounted guard
**What goes wrong:** Using `context`/`ref` after an await when the screen was popped.
**How to avoid:** These are `ConsumerStatefulWidget`s — guard with `if (!mounted) return;` (State) or `if (!context.mounted) return;` after each await, matching the established v1.2 convention noted in STATE.md and `settings_account_screen.dart`.

### Pitfall 6: Reduced-motion not honored on drag animation
**What goes wrong:** Reorder animation plays for users who requested reduced motion (WCAG 2.1 AA + PRODUCT.md).
**How to avoid:** `ReorderableListView`'s drag proxy animation is largely intrinsic; check `MediaQuery.of(context).disableAnimations` (or `.accessibleNavigation`) and avoid adding *extra* transitions. Do not add gratuitous `AnimatedSize`/`AnimatedOpacity` around rows. The UI-SPEC only requires not *adding* motion beyond the widget's built-in affordance.

## Runtime State Inventory

> Not a rename/refactor/migration phase — this is greenfield UI. Section included only to state explicitly that no stored-state migration is involved.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — preferences already persist server-side via existing endpoints; no new collections or keys | None |
| Live service config | None | None |
| OS-registered state | None | None |
| Secrets/env vars | None | None |
| Build artifacts | `*.g.dart` for the two preference providers will regenerate after adding `reorder` methods; `app_localizations.dart` regenerates after ARB edits | Run `dart run build_runner build` (providers) + `flutter gen-l10n` (or it runs on build via `generate: true`) |

## Code Examples

### Reorder method to ADD to CategoryPreferenceNotifier
```dart
// ADD to app/lib/provider/category_preference_provider.dart
// Payload verified: web/src/lib/models/api/category_preference_schema.ts → { categories: string[] }
Future<void> reorder(List<String> orderedCategoryIds) async {
  await ref.read(apiProvider).post(
    '/user-category-preference/reorder',
    data: {'categories': orderedCategoryIds},
  );
  ref.invalidateSelf();
}
```

### Reorder method to ADD to SubcategoryPreferenceNotifier
```dart
// ADD to app/lib/provider/subcategory_preference_provider.dart
// Payload verified: web/src/lib/models/api/subcategory_preference_schema.ts
//   → { category: string, subcategories: string[] }
Future<void> reorder(String categoryId, List<String> orderedSubcategoryIds) async {
  await ref.read(apiProvider).post(
    '/user-subcategory-preference/reorder',
    data: {'category': categoryId, 'subcategories': orderedSubcategoryIds},
  );
  ref.invalidateSelf();
}
```

### Confirm dialog (returns bool) — mirror settings_account_screen.dart
```dart
// Source: app/lib/routes/settings_account_screen.dart:74-93 [VERIFIED in-repo]
final confirmed = await showDialog<bool>(
  context: context,
  builder: (ctx) => AlertDialog(
    title: Text(l10n.settings_categories_confirm_disable_title), // new key
    content: Column(mainAxisSize: MainAxisSize.min, children: [
      Text(l10n.settings_categories_confirm_disable_body(count, name)), // ICU placeholders
      TextButton(
        onPressed: () { /* navigate to filtered trail list */ },
        child: Text(l10n.settings_categories_confirm_view_trails),
      ),
    ]),
    actions: [
      TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: Text(l10n.cancel)),
      TextButton(onPressed: () => Navigator.of(ctx).pop(true),
                 child: Text(l10n.settings_categories_confirm_disable_confirm)), // "Disable anyway"
    ],
  ),
);
if (confirmed == true) {
  await _save(...); // proceed with upsert(id, false)
}
// else: switch reverts automatically because provider state was never changed (D-10)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Inline `ExpansionTile` for subcategories (original SETCAT-08) | Dedicated `SettingsSubcategoriesScreen` with reorder | Phase 12 discuss (2026-07-01) | Requirements amended; ExpansionTile NOT used |
| Whole-row drag / long-press reorder | Dedicated drag-handle (`ReorderableDragStartListener` + `buildDefaultDragHandles:false`) | D-01 | Explicit handle = clearer affordance, doesn't block body tap |
| `activeColor` on Switch (deprecated in newer Material) | `activeThumbColor` | Flutter 3.x | Existing code already uses `activeThumbColor` (settings_notifications_screen.dart) — copy it |

**Deprecated/outdated:**
- `Switch.activeColor` — the repo already uses `activeThumbColor`; follow suit.
- The web's plugin-mapping confirm messaging — intentionally NOT ported (D-13).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `/profile/{handle}/trails` response exposes `totalHits` (or `estimatedTotalHits`) usable as the own-trail count | Pattern 5 | Count shows 0 → confirm dialog never fires; fallback to `data['hits'].length` with a large `hitsPerPage`, or count via a paginated sum. Verify response shape at plan/execute time. |
| A2 | Meilisearch filter field names are `category_id` / `subcategory_id` for the profile-trails index | Pattern 5 | Filter matches nothing. Low risk — same names verified in `TrailFilter.toFilterText` (trail.dart) which already queries this index successfully. |
| A3 | Passing the `Category` object via go_router `extra:` is acceptable (vs. re-fetching by ID) | Pattern 2/6 | On deep-link/process-restart `extra` is null (as router already handles for `/trail/:id/navigate`). Since this screen is only reached by in-app push, acceptable; planner may add an id-param fallback. |
| A4 | `String.compareTo` on lowercased names is an adequate stand-in for `localeCompare(sensitivity:"base")` | Pattern 4 | Minor mis-ordering for accented/locale-specific names. Acceptable per existing app pragmatism; no collation package added. |
| A5 | `SettingsSubcategoriesScreen` is a leaf (no body-tap navigation), only toggle + reorder | D-06 | If a deeper level were expected, structure changes. CONTEXT D-06 explicitly says "no further nesting" — low risk. |
| A6 | The "view trails" action target is the user's filtered own-trail list (planner's discretion on exact route) | SETCAT-11 | Cosmetic; any reasonable trail-list destination satisfies the requirement. |

## Open Questions

1. **Own-trail count response field (`totalHits` vs `estimatedTotalHits`)**
   - What we know: endpoint returns a Meilisearch `SearchResponse`; `profile_trails_provider.dart` reads `totalPages` from it, confirming it's the raw MS shape.
   - What's unclear: whether the index returns exact `totalHits` or only `estimatedTotalHits`.
   - Recommendation: read `totalHits ?? estimatedTotalHits ?? 0`. Verify with a live call during execution; a Wave-0 spike is optional given nyquist_validation is off.

2. **"View trails" link destination on mobile**
   - What we know: web links to `/trails?author=<actor>&category=<id>`.
   - What's unclear: the exact mobile route (no `/trails` list-by-filter deep route enumerated).
   - Recommendation: navigate to `/profile/{handle}/trails` (own profile trail list) with the filter applied, or the trail-filter screen pre-seeded. Planner's discretion (D-Discretion + A6).

3. **Where to place the sort helper**
   - Recommendation: a new `app/lib/util/category_preference_sort.dart` (or inline private fns per screen). Keep it out of the provider to preserve the read-only-provider boundary from Phase 10.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | entire phase | ✓ | 3.44.2 stable | — |
| Dart SDK | codegen, build | ✓ | ^3.11.5 (pubspec) | — |
| build_runner (freezed/riverpod codegen) | regenerate `*.g.dart` after adding reorder methods | ✓ (dev dep, used in prior phases) | per pubspec | — |
| flutter gen-l10n | regenerate `AppLocalizations` after ARB edits | ✓ (`generate: true`, l10n.yaml present) | SDK | runs on build |
| Running backend (PocketBase + Meilisearch) | manual verification of reorder + own-trail count | Not probed (dev machine may not run it) | — | Human-verify at end of phase (config: human_verify_mode = end-of-phase); logic is code-verifiable |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** Live backend for runtime verification — deferred to end-of-phase human verification (matches config).

## Security Domain

> `security_enforcement: true`, ASVS level 1. This phase is client-side UI calling already-secured endpoints.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Session handled by existing `authProvider`; no auth logic added |
| V3 Session Management | no | Reuses existing dio session/cookie handling via `apiProvider` |
| V4 Access Control | yes | Server injects `user` from session on all preference writes; client MUST NOT send `user` (existing `upsert` omits it — new `reorder` methods must too). Server enforces `author` on `/profile/{handle}/trails`. [T-10-05 precedent] |
| V5 Input Validation | yes (server-side) | Reorder payloads validated by server Zod schemas (`z.string().length(15)` per ID). Client sends IDs from trusted provider data, not free text. |
| V6 Cryptography | no | None |

### Known Threat Patterns for Flutter + PocketBase/Meilisearch client

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Client tampering with `user` field on preference write | Tampering / Elevation | Never send `user`; server derives from session (V4). Enforced already server-side. |
| Meilisearch filter injection via category/subcategory IDs | Tampering | IDs come from server-fetched provider lists (15-char PB IDs), not user input; interpolated into `IN ['<id>']` matching existing `TrailFilter` usage. Low risk — no free-text reaches the filter. |
| Leaking another user's trail counts | Info disclosure | `/profile/{handle}/trails` is author-scoped server-side; querying own `preferredUsername` returns only own trails. |

## Sources

### Primary (HIGH confidence)
- `app/lib/routes/settings_notifications_screen.dart` — toggle/save/toast + `activeThumbColor` + section-header pattern
- `app/lib/routes/settings_account_screen.dart` — `showDialog<bool>` + `AlertDialog` confirm pattern, mounted guards
- `app/lib/routes/settings_screen.dart` — settings tile shape (`FaIcon` size 18 + chevron + `context.push`)
- `app/lib/provider/router_provider.dart` — `/settings/*` child-route registration + `extra:` passing
- `app/lib/provider/category_preference_provider.dart`, `subcategory_preference_provider.dart` — existing `build`/`upsert`; where `reorder` must be added
- `app/lib/provider/profile/profile_trails_provider.dart` — profile-trails POST call shape (own-trail count)
- `app/lib/models/category.dart`, `subcategory.dart`, `category_preference.dart`, `subcategory_preference.dart`, `trail.dart` (`TrailFilter`) — models + `displayName`
- `app/lib/util/.../category_icon_util.dart` — icon/avatar resolution (Phase 11)
- `app/lib/components/async_loader.dart` — loading wrapper (D-14)
- `web/src/lib/util/category_util.ts` — sort/visibility logic to mirror
- `web/src/routes/settings/categories/+page.svelte` + `+page.server.ts` — feature-parity reference (reorder, confirm, own-trail count)
- `web/src/lib/models/api/category_preference_schema.ts` + `subcategory_preference_schema.ts` — **verified reorder payload shapes**
- `web/src/routes/api/v1/profile/[handle]/trails/+server.ts` — confirms server-side author enforcement + Meilisearch `SearchResponse`
- api.flutter.dev `ReorderableListView-class` + `ReorderableDragStartListener-class` — custom drag-handle API + `buildDefaultDragHandles`
- `flutter --version` → 3.44.2 stable; `app/pubspec.yaml`, `app/l10n.yaml` — SDK, deps, l10n config

### Secondary (MEDIUM confidence)
- Flutter `onReorder` index-shift convention (`if (newIndex > oldIndex) newIndex -= 1`) — universally documented; api.flutter.dev notes auto-adjustment only for the newer `onReorderItem` callback

### Tertiary (LOW confidence)
- Exact Meilisearch total-count field name for the profile-trails index (`totalHits` vs `estimatedTotalHits`) — inferred from raw `SearchResponse` type; flagged A1.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all first-party/existing deps verified against `pubspec.yaml` + live `flutter --version`
- Architecture: HIGH — every pattern except reorder has a verified in-repo source; reorder verified against official Flutter API docs
- Pitfalls: HIGH — drawn from Flutter API docs + the web reference implementation's own handling
- Own-trail count field name: MEDIUM (A1) — endpoint + call shape verified, exact count field to confirm at execution

**Research date:** 2026-07-01
**Valid until:** 2026-07-31 (stable Flutter APIs; in-repo patterns unlikely to shift before Phase 12 executes)
