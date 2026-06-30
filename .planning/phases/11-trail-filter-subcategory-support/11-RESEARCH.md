# Phase 11: Trail Filter Subcategory Support - Research

**Researched:** 2026-06-29
**Domain:** Flutter (Riverpod) filter UI + Meilisearch filter-string construction, mirroring a shipped web feature
**Confidence:** HIGH (all findings verified against the live codebase; no external dependencies)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**TrailFilter Model**
- **D-01:** `TrailFilter` freezed model gains `List<Subcategory> subcategory` field alongside the existing `List<Category> category` field (FILTER-01).
- **D-02:** `TrailFilter.toFilterString()` adds a `subcategory_id IN [...]` clause using subcategory IDs (not names). Field name is `subcategory_id` — mirrors the web's filter builder in `trail_store.ts` and the PocketBase column name confirmed in `web/src/lib/models/trail.ts`.
  - *Research correction:* the Flutter method is named **`toFilterText()`**, not `toFilterString()`. See **Pitfall 1** for a critical correction on the category clause.

**Subcategory Chips in TrailFilterScreen**
- **D-03:** New titled **"Subcategories"** section below the Category section. Conditionally rendered when `filter.category.isNotEmpty`. Matches the existing labelled-block layout pattern.
- **D-04:** Section animates in/out with `AnimatedSize` (standard Flutter widget, zero extra dependencies).
- **D-05:** Subcategory chips display **only subcategories belonging to the currently selected categories** (filter: `subcategory.category == selectedCategory.id`).
- **D-06:** Hidden categories (`visible: false` in CategoryPreference) and hidden subcategories (`visible: false` in SubcategoryPreference) are omitted from all chip options. Missing preference record (null) = visible (default shown).

**Chip Icon Rendering**
- **D-07:** `WandererFilterChip<T>` is extended with an optional `Widget? Function(T item)? avatarBuilder`. Existing call sites pass `null` (no change). New category/subcategory call sites pass a builder that returns an icon widget.
- **D-08:** Category chips show a FontAwesome icon via `avatarBuilder`. Icon name resolved from `category.icon` using `fontAwesomeIconsMap` in `icon_util.dart`. Strip `fa-` prefix before lookup. Fall back to `Icons.category` (Material) if icon name not found in map.
- **D-09:** Subcategory chips show primary FA icon + `badge_icon` overlay. `Stack` in `avatarBuilder` — primary `FaIcon` (16px), badge `FaIcon` (10px) positioned `Alignment.bottomRight`. Primary icon falls back to parent category's icon if `subcategory.icon` is empty. Badge icon omitted if `subcategory.badgeIcon` is null/empty.
- **D-10:** Locale-resolved names via `CategoryDisplay.displayName(Localizations.localeOf(context))` for category chips and `SubcategoryDisplay.displayName(...)` for subcategory chips. Uses Phase 10 helpers.

**Quick Filter Bar**
- **D-11:** The existing **Category chip opens a single bottom sheet** — no second chip added. The sheet is extended to include a Subcategories section below the categories.
- **D-12:** Subcategory section inside the bottom sheet follows the **same AnimatedSize pattern** as TrailFilterScreen (D-04).
- **D-13:** The Category chip's active state uses `filter.category.isNotEmpty || filter.subcategory.isNotEmpty`. (Update `_isCategoryActive()`.)
- **D-14:** Bottom sheet initial size stays at `initialChildSize: 0.5`. Existing `maxChildSize: 0.9` + scroll controller handle overflow.

### Claude's Discretion
None explicitly designated — CONTEXT.md locked all implementation choices. Research-identified discretion areas: the exact l10n key name for the new "Subcategories" string, and the precise badge `Stack` offset values.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. (Broader deferred items: trail-form category picker → v1.4; bulk-edit modal → web-only; subcategory reordering within a category → future milestone.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FILTER-01 | `TrailFilter` gains a `subcategory` list alongside `category` | `app/lib/models/trail.dart:160-189` — add `required List<Subcategory> subcategory` field; `Subcategory` model exists (`app/lib/models/subcategory.dart`). Default value `[]` must be added to the default-filter constructor in `trail_filter_provider.dart:26-48`. |
| FILTER-02 | TrailFilterScreen shows a subcategory chip section appearing when ≥1 category selected, listing subcategories of selected categories | `subcategoryProvider` returns `List<Subcategory>` synchronously (cache-first). Filter by `sub.category == selectedCat.id`. `AnimatedSize` wraps the section. See Pattern 2. |
| FILTER-03 | Selecting/deselecting subcategory chips updates `TrailFilter.subcategory` and the API filter payload | `updateFilter((f) => f.copyWith(subcategory: ...))`. Filter payload built in `toFilterText()` → `subcategory_id IN [...]`. **Verified filterable** in Meilisearch (see Standard Stack / Pitfall 1). |
| FILTER-04 | Category chips display locale-resolved names (CAT-01 fallback chain) | `CategoryDisplay.displayName(Locale?)` exists (`category.dart:39-44`). Current code uses raw `c.name` (`trail_filter_screen.dart:79`) — must switch to `displayName`. |
| FILTER-05 | Quick filter bar Category chip bottom sheet supports both categories and subcategories | `_showCategorySheet` (`trail_quick_filter_bar.dart:230-298`) — add Subcategories section + AnimatedSize inside the existing sheet. |
| FILTER-06 | Both surfaces omit categories marked `visible: false` from selectable chips | `categoryPreferenceProvider` (AsyncValue) supplies prefs. Mirror `categoryVisibleInDesign` / `subcategoryVisible` from `category_util.ts`. |
| FILTER-07 | Subcategory chips omit subcategories marked `visible: false` | `subcategoryPreferenceProvider` (AsyncValue). Same visibility rule. |
</phase_requirements>

## Summary

Phase 11 is a **pure Flutter UI + filter-string change** with **no new packages, no new screens, no new API endpoints**. It mirrors a shipped web feature (PR #1059), whose icon/visibility/filter logic lives in `web/src/lib/util/category_util.ts` and `web/src/lib/stores/trail_store.ts`. All Phase 10 building blocks (Category/Subcategory models with locale display extensions, four cache-first/AsyncValue providers, and the FontAwesome icon map) already exist and were verified. The work is: (1) add a `subcategory` field to `TrailFilter`, (2) emit a `subcategory_id IN [...]` clause from `toFilterText()`, (3) render two new chip sections (full filter screen + quick-filter bottom sheet) with locale names, FA icons, badge overlays, `AnimatedSize` expansion, and preference-based visibility filtering.

The single most important finding is a **latent bug the planner must address, not just an addition**: the redesigned Meilisearch index (`db/main.go:335-352`, applied on every server boot via `UpdateSettings`) declares only `category_id` and `subcategory_id` as filterable category attributes — the plain `category` (name) attribute was dropped. But Flutter's current `toFilterText()` still emits `category IN ['<name>']` (`trail.dart:275-278`). That clause now targets a non-filterable attribute and will be rejected/ignored by Meilisearch. To make subcategory filtering work **and** keep category filtering working, the category clause must migrate from name-based to **`category_id IN ['<id>']`**, exactly as the web does (`trail_store.ts:887`). CONTEXT.md's D-02 covers the subcategory clause but does not flag this required category-clause migration.

Second-order finding: `TrailFilterValues` is only the server-provided *bounds* object (max distance/elevation), not a persisted filter-selection schema. The active category/subcategory selection lives in-memory in `TrailFilterNotifier`. So D-87's open question ("check if subcategory needs adding to persisted filter values") resolves to **no persistence change** — only the default-filter constructor needs `subcategory: []`.

**Primary recommendation:** Implement the locked CONTEXT.md decisions verbatim, but in the `toFilterText()` task also migrate the existing category clause from `category IN ['name']` to `category_id IN ['id']` (mirror `trail_store.ts`), and emit the subcategory clause as a combined OR group so subcategory + category selections compose correctly.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Subcategory chip selection UI | Flutter Client (widgets) | — | Pure presentation + local filter state |
| Filter-string construction (`subcategory_id IN [...]`) | Flutter Client (model method) | API/Backend (Meilisearch interprets it) | `toFilterText()` builds the string; Meilisearch evaluates it server-side |
| Category/subcategory data | Flutter Client (Riverpod providers, Phase 10) | API/Backend (`/category`, `/subcategory`) | Cache-first ObjectBox; refreshed from API |
| Visibility (hidden) filtering | Flutter Client (provider read) | API/Backend (`/user-(sub)category-preference`) | Prefs fetched per-user; UI applies the filter |
| Trail result filtering by visibility | API/Backend (`withTrailPreferenceFilter`) | — | Already enforced server-side; Flutter receives pre-filtered trails |
| Locale name resolution | Flutter Client (model extension) | — | `displayName(Locale?)` resolves translations client-side |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| flutter_riverpod | ^3.3.1 | Filter + provider state | [VERIFIED: app/pubspec.yaml:27] Already the app's state-management standard |
| riverpod_annotation / riverpod_generator | ^4.0.2 / ^4.0.3 | Code-gen providers | [VERIFIED: app/pubspec.yaml:45,75] Phase 10 providers use this |
| freezed_annotation | (3.x) | `TrailFilter` immutable model + `copyWith` | [VERIFIED: trail.dart uses @freezed] `subcategory` field is added here |
| font_awesome_flutter | ^11.0.0 | FA icon rendering (`FaIcon`, `FaIconData`) | [VERIFIED: app/pubspec.yaml:29] `fontAwesomeIconsMap` already maps name→`FaIconData` |

**No packages to install.** All capabilities use Flutter SDK widgets (`AnimatedSize`, `Stack`, `FilterChip`, `DraggableScrollableSheet`) and already-present dependencies. **The Package Legitimacy Audit section is intentionally omitted — this phase installs nothing.**

### Supporting (Flutter SDK widgets — already available)
| Widget | Purpose | When to Use |
|--------|---------|-------------|
| `AnimatedSize` | Smooth expand/collapse of the Subcategories section | D-04 / D-12 — wrap the conditional section |
| `Stack` + `Positioned`/`Align` | Subcategory badge-icon overlay | D-09 — primary 16px FA + 10px badge bottom-right |
| `FilterChip` | Chip rendering (inside `WandererFilterChip`) | Existing widget; add `avatarBuilder` slot |
| `DraggableScrollableSheet` | Quick-filter bottom sheet (already scrollable) | D-14 — no size change needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `AnimatedSize` | `AnimatedCrossFade` / `ExpansionTile` | CONTEXT.md locks `AnimatedSize` (zero deps, matches existing pattern). Do not substitute. |
| `avatarBuilder` on `WandererFilterChip` | New dedicated chip widget | CONTEXT.md D-07 locks extending the existing generic chip; keeps all call sites consistent. |

**Version verification:**
```bash
# Verified from app/pubspec.yaml (no registry lookup needed — all deps already resolved in pubspec.lock)
flutter_riverpod: ^3.3.1   font_awesome_flutter: ^11.0.0   riverpod_annotation: ^4.0.2
```

## Architecture Patterns

### System Architecture Diagram

```
User taps category chip (TrailFilterScreen OR quick-filter bottom sheet)
        │
        ▼
WandererFilterChip.onChanged ──► trailFilterProvider(filterId).notifier.updateFilter(
        │                              (f) => f.copyWith(category: [...]) )
        ▼
TrailFilter state updated (in-memory, AsyncData)
        │
        ├──► UI rebuilds ──► Subcategories section reveals (AnimatedSize) when category.isNotEmpty
        │                         │
        │                         ▼
        │                    subcategoryProvider (List<Subcategory>, cache-first)
        │                    ⨯ filtered by sub.category ∈ selectedCategoryIds
        │                    ⨯ filtered by subcategoryPreferenceProvider (visible != false)
        │                         │
        │                         ▼
        │                    user toggles subcategory chip ──► updateFilter(subcategory: [...])
        ▼
trail search trigger (map_trail_search_provider / trail_search_provider)
        │
        ▼
TrailFilter.toFilterText(actor: ...) builds Meilisearch filter string:
   "... AND (category_id IN ['c1'] OR subcategory_id IN ['s1','s2'])"
        │
        ▼
POST to Meilisearch via /trail search endpoint ──► filtered trail results
```

### Recommended File Touch Map (no new files except optional test + l10n)
```
app/lib/models/trail.dart                              # add subcategory field + clause (FILTER-01, -03)
app/lib/provider/trail/trail_filter_provider.dart      # add subcategory: [] to defaultFilter
app/lib/components/base/wanderer_filter_chip.dart       # add avatarBuilder param (D-07)
app/lib/routes/trail_filter_screen.dart                # Subcategories section + locale names + icons (FILTER-02,-04,-06,-07)
app/lib/components/trail/trail_quick_filter_bar.dart    # extend category sheet + _isCategoryActive (FILTER-05, D-13)
app/lib/util/icon_util.dart                            # (read-only) fontAwesomeIconsMap lookup
app/lib/i18n/app_en.arb (+ other locales)              # new "subcategories" string (none exists yet)
```

### Pattern 1: Migrate category clause to ID + add subcategory clause (combined OR group)
**What:** Mirror the web's combined category/subcategory OR group so selections compose.
**When to use:** Inside `TrailFilter.toFilterText()`, replacing the current name-based category block.
**Example:**
```dart
// Source: mirror of web/src/lib/stores/trail_store.ts:862-908 (VERIFIED in repo)
// Replaces the existing `category IN ['<name>']` block in trail.dart:275-278
if (category.isNotEmpty || subcategory.isNotEmpty) {
  final List<String> categoryParts = [];

  if (category.isNotEmpty) {
    final ids = category.map((c) => "'${c.id}'").join(", ");
    categoryParts.add('category_id IN [$ids]');
  }
  if (subcategory.isNotEmpty) {
    final ids = subcategory.map((s) => "'${s.id}'").join(", ");
    categoryParts.add('subcategory_id IN [$ids]');
  }
  if (categoryParts.isNotEmpty) {
    parts.add('(${categoryParts.join(" OR ")})');
  }
}
```
> Note: the web has extra handling for "no-subcategory synthetic categories"; the Flutter filter does not model those, so the simple OR group above is sufficient for this phase.

### Pattern 2: Conditional Subcategories section with AnimatedSize + visibility filtering
**What:** Reveal subcategory chips for selected categories, omitting hidden ones.
**Example:**
```dart
// f = current TrailFilter; subcategories from subcategoryProvider (List<Subcategory>)
final selectedCategoryIds = f.category.map((c) => c.id).toSet();
final subPrefs = ref.watch(subcategoryPreferenceProvider).value ?? [];
final visibleSubs = subcategories
    .where((s) => selectedCategoryIds.contains(s.category))           // D-05
    .where((s) => subPrefs                                            // D-06 / FILTER-07
        .firstWhereOrNull((p) => p.subcategory == s.id)?.visible != false)
    .toList();

AnimatedSize(
  duration: const Duration(milliseconds: 200),
  alignment: Alignment.topCenter,
  child: f.category.isEmpty || visibleSubs.isEmpty
      ? const SizedBox.shrink()
      : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(l10n.subcategories, style: TextTheme.of(context).labelLarge),
          const SizedBox(height: 8),
          WandererFilterChip<Subcategory>(
            options: visibleSubs,
            selectedValues: f.subcategory,
            multiple: true,
            labelBuilder: (s) => s.displayName(Localizations.localeOf(context)),
            avatarBuilder: (s) => _subcategoryAvatar(s, parentOf(s)),  // D-09
            onChanged: (sel) => ref.read(trailFilterProvider(widget.filterId).notifier)
                .updateFilter((flt) => flt.copyWith(subcategory: sel)),
          ),
        ]),
)
```
> `firstWhereOrNull` requires `package:collection` (already a transitive dep of Flutter; import `package:collection/collection.dart`). Alternatively use a manual loop to avoid the import.

### Pattern 3: Category visibility filter (mirror categoryVisibleInDesign)
```dart
// Source: mirror of category_util.ts:389-411 (VERIFIED)
final catPrefs = ref.watch(categoryPreferenceProvider).value ?? [];
final visibleCategories = (categoryProvider value).where((c) =>
    catPrefs.firstWhereOrNull((p) => p.category == c.id)?.visible != false).toList();
// null preference == visible (D-06)
```

### Anti-Patterns to Avoid
- **Filtering category by name (`category IN ['<name>']`):** the redesigned Meilisearch index no longer makes `category` filterable. Use `category_id`. (See Pitfall 1.)
- **Using `.valueOrNull` on AsyncValue:** project memory note — use `.value` for nullable access (preference/category providers are `AsyncValue`). [CITED: user MEMORY.md — feedback-riverpod-value-or-null]
- **Reading `subcategoryProvider` as an AsyncValue:** it is a **synchronous** notifier returning `List<Subcategory>` directly — `ref.watch(subcategoryProvider)` gives a `List`, not an `AsyncValue`. Do not call `.value` on it. (Contrast with category/preference providers which ARE AsyncValue.)
- **Raw `c.name` for chip labels:** FILTER-04 requires `displayName(locale)`. The current `trail_filter_screen.dart:79` and `trail_quick_filter_bar.dart:279` use `c.name` — both must change.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Locale name resolution | Custom translation lookup | `CategoryDisplay.displayName` / `SubcategoryDisplay.displayName` (Phase 10) | Already implements CAT-01 fallback chain (locale→en→raw) |
| FA icon name → widget | Switch statement on icon names | `fontAwesomeIconsMap[name]` after stripping `fa-` | 2000+ entry map already exists in `icon_util.dart` |
| Subcategory data fetch/cache | New provider | `subcategoryProvider` (cache-first, keepAlive) | Phase 10 already built this |
| Visibility preference fetch | New provider | `categoryPreferenceProvider` / `subcategoryPreferenceProvider` | Phase 10; AsyncValue, anonymous-safe |
| Animated expand/collapse | Custom AnimationController | `AnimatedSize` | Zero deps; locked by D-04 |
| Filter-string composition | New filter builder | Extend existing `toFilterText()` | All other filters already flow through it |

**Key insight:** Phase 10 deliberately front-loaded every data/cache/locale primitive this phase needs. Phase 11 is almost entirely view-layer wiring plus one filter-string edit — resist rebuilding any data plumbing.

## Common Pitfalls

### Pitfall 1: Category filter silently broken — name vs. id mismatch (HIGH confidence, CRITICAL)
**What goes wrong:** Adding only the `subcategory_id` clause (per D-02) leaves the existing `category IN ['<name>']` clause emitting a filter against an attribute Meilisearch no longer indexes as filterable. Category filtering may already be broken on the redesigned index, and subcategory work won't fix it.
**Why it happens:** `db/main.go:335-352` (`initMeilisearchConfig`, run on every boot via `UpdateSettings`) declares filterable category attributes as **only** `category_id` and `subcategory_id`. The legacy `category` (name) filterable attribute from migration `1742167033_init_meilisearch.go:32` is overwritten/dropped. The indexed trail document (`db/util/meilisearch.go:81-84`) still contains a `category` *name* field, but it is not filterable. Flutter's `toFilterText()` (`trail.dart:275-278`) still filters by `category` name.
**How to avoid:** In the `toFilterText()` task, migrate the category clause to `category_id IN ['<id>']` (the `Category` objects in `filter.category` carry `.id`). Combine with the subcategory clause in one OR group (Pattern 1).
**Warning signs:** Selecting a category returns all trails (filter ignored) or zero trails (filter error swallowed); web filters correctly while Flutter does not.
[VERIFIED: db/main.go:335-352, db/util/meilisearch.go:81-84, web/src/lib/stores/trail_store.ts:862-908]

### Pitfall 2: `subcategoryProvider` is synchronous, not AsyncValue (MEDIUM)
**What goes wrong:** Treating `ref.watch(subcategoryProvider)` as `AsyncValue<List<Subcategory>>` and calling `.value`/`.when` on a plain `List`.
**Why:** Its `build()` returns `List<Subcategory>` directly (cache-first, `app/lib/provider/trail/subcategory_provider.dart:13`), unlike `categoryProvider` (FutureOr → AsyncValue) and the preference providers (Future → AsyncValue).
**How to avoid:** Use the list directly. Mixed provider shapes in the same widget — be deliberate per provider.
[VERIFIED: subcategory_provider.dart vs category_provider.dart]

### Pitfall 3: `TrailFilter` required field breaks all constructors (MEDIUM)
**What goes wrong:** Adding `required List<Subcategory> subcategory` forces every `TrailFilter(...)` literal to pass it; the freezed model has no default for required fields.
**Why:** `TrailFilter` is constructed in `trail_filter_provider.dart:26-48` (defaultFilter) and possibly in tests/fixtures.
**How to avoid:** Either use `@Default(<Subcategory>[]) List<Subcategory> subcategory` (freezed default, mirrors how `tags`/`category` patterns work elsewhere) OR add `subcategory: []` to every constructor. Grep for `TrailFilter(` across `app/lib` and `app/test` first. Note the existing `category` field is `required` (no default), so to stay consistent the planner may prefer `required` + updating the one constructor — a compiler-driven sweep (same approach Phase 10 used for `Settings.category` removal).
[VERIFIED: trail.dart:160-189, trail_filter_provider.dart:26-48]

### Pitfall 4: Missing l10n key for "Subcategories" (LOW)
**What goes wrong:** Referencing `l10n.subcategories` when no such ARB key exists → build/codegen failure.
**Why:** `grep` of `app/lib/i18n/app_en.arb` finds `categories` and `filter_categories` but **no** subcategory key.
**How to avoid:** Add a `"subcategories"` (and optionally `"filter_subcategories"`) entry to `app_en.arb` and the other locale ARBs, then run l10n codegen. ~13 locale files exist (`app_*.arb`).
[VERIFIED: grep app/lib/i18n/app_en.arb]

### Pitfall 5: Badge overlay clipping inside FilterChip avatar (LOW)
**What goes wrong:** The 10px badge positioned bottom-right of a 16px primary icon can be clipped by the chip's avatar bounds.
**Why:** Material `FilterChip` constrains the avatar to a circular/clipped region.
**How to avoid:** Use a `Stack` with `clipBehavior: Clip.none` and size the avatar container to fit both icons; test visually on-device (UI safety gate is enabled in config).
[ASSUMED — based on Flutter avatar layout behavior; verify during implementation]

## Code Examples

### avatarBuilder extension on WandererFilterChip (D-07)
```dart
// app/lib/components/base/wanderer_filter_chip.dart
class WandererFilterChip<T> extends StatelessWidget {
  // ...existing fields...
  final Widget? Function(T item)? avatarBuilder;   // ADD — optional, defaults null

  const WandererFilterChip({
    // ...existing...
    this.avatarBuilder,
  });

  // inside build(), on FilterChip:
  //   avatar: avatarBuilder?.call(option),
}
```

### Category avatar via fontAwesomeIconsMap (D-08)
```dart
// strip leading 'fa-' then look up; fall back to Material Icons.category
Widget _categoryAvatar(Category c) {
  final raw = (c.icon ?? '').trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  final faData = fontAwesomeIconsMap[key];
  return faData != null
      ? FaIcon(faData, size: 16)
      : const Icon(Icons.category, size: 16);
}
```

### Subcategory avatar with badge overlay (D-09)
```dart
Widget _subcategoryAvatar(Subcategory s, Category? parent) {
  // primary icon: subcategory.icon, else parent category icon (mirror displaySubcategoryIcon)
  final primaryRaw = ((s.icon?.trim().isNotEmpty ?? false) ? s.icon! : (parent?.icon ?? '')).trim();
  final primaryKey = primaryRaw.startsWith('fa-') ? primaryRaw.substring(3) : primaryRaw;
  final primary = fontAwesomeIconsMap[primaryKey];

  final badgeRaw = (s.badgeIcon ?? '').trim();
  final badgeKey = badgeRaw.startsWith('fa-') ? badgeRaw.substring(3) : badgeRaw;
  final badge = badgeKey.isEmpty ? null : fontAwesomeIconsMap[badgeKey];

  return Stack(clipBehavior: Clip.none, children: [
    primary != null ? FaIcon(primary, size: 16) : const Icon(Icons.category, size: 16),
    if (badge != null)
      Positioned(right: -2, bottom: -2, child: FaIcon(badge, size: 10)),
  ]);
}
// Source: mirrors displaySubcategoryIcon/displaySubcategoryBadgeIcon (category_util.ts:221-233)
```

## Runtime State Inventory

> This is a refactor-adjacent phase (adds a field + migrates a filter clause). No stored runtime state is renamed. Categories below verified explicitly.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `subcategory` field is additive to an in-memory filter model; no DB/ObjectBox key renamed. Subcategory cache (ObjectBox) already exists from Phase 10. | None |
| Live service config | **Meilisearch index `trails`** — filterable attributes already include `subcategory_id` (`db/main.go:340`). No index reconfiguration needed in this (Flutter-only) phase. The category-name→id filter migration is a **client-side string change**, not an index change. | None (index already supports `subcategory_id`) |
| OS-registered state | None — no tasks/services embed any renamed string. | None |
| Secrets/env vars | None referenced. | None |
| Build artifacts | freezed/riverpod codegen for `trail.dart` and `trail_filter_provider.dart` must be regenerated after adding the `subcategory` field (`dart run build_runner build`). l10n codegen after ARB edits. | Run build_runner + l10n gen |

**Canonical question — after files are updated, what runtime state still has the old behavior?** Only the Meilisearch index, which already exposes `subcategory_id`; the stale behavior is in the *client* filter string, fixed by the `toFilterText()` edit. Nothing server-side or stored needs migration.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Meilisearch `trails` filters by `category` (name) | Filters by `category_id` + `subcategory_id` | Category redesign (`db/main.go:335`, migration `1781000000`) | Flutter `toFilterText()` category clause is now stale — must migrate to `category_id` |
| `Settings.category` single category | Per-user category/subcategory *preferences* (visibility + priority) | Phase 10 (CAT-05) | Visibility filtering in this phase reads preference providers, not Settings |

**Deprecated/outdated:**
- `category IN ['<name>']` filter clause (Flutter `trail.dart:275-278`): stale against the redesigned index; replace with `category_id IN ['<id>']`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The legacy `category` (name) attribute is genuinely non-filterable post-redesign, so the current Flutter category filter is broken | Pitfall 1 | If Meilisearch silently tolerates filtering a non-filterable attribute as a no-op, category filtering returns all trails rather than erroring — still wrong, still fixed by the same migration. Low risk to the recommendation. |
| A2 | Badge overlay may clip inside FilterChip avatar bounds | Pitfall 5 | Cosmetic only; resolved during on-device UI verification |
| A3 | `package:collection` (`firstWhereOrNull`) is available transitively | Pattern 2 | If not, use a manual loop — trivial alternative, no blocker |
| A4 | No additional `TrailFilter(...)` constructors exist beyond `trail_filter_provider.dart` and tests | Pitfall 3 | Planner should grep `TrailFilter(` to confirm; missing one = compile error caught immediately |

## Open Questions

1. **Should the category clause migration (name→id) be an explicit task or folded into the subcategory task?**
   - What we know: Both edits are in the same `toFilterText()` block; doing them together (Pattern 1) is cleanest.
   - What's unclear: Whether the team wants the bug-fix called out separately for traceability.
   - Recommendation: One task that rewrites the combined category/subcategory clause, with a commit message noting the category-name→id fix. Add a test asserting the emitted filter string contains `category_id IN` and `subcategory_id IN`.

2. **`required` vs `@Default([])` for the new `subcategory` field.**
   - What we know: Existing `category` is `required` (no default); Phase 10 used compiler-driven sweeps successfully.
   - Recommendation: Use `@Default(<Subcategory>[]) List<Subcategory> subcategory` to avoid touching test fixtures, OR `required` + update the single defaultFilter constructor. Planner's choice; flag the constructor-sweep either way.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | Build/run app | ✓ (assumed dev env) | ^3.11.5 (pubspec) | — |
| build_runner (freezed/riverpod codegen) | Regenerate `*.g.dart` / `*.freezed.dart` | ✓ (dev dep) | — | — |
| font_awesome_flutter | Icon rendering | ✓ | ^11.0.0 | Material `Icons.category` fallback (already in design) |
| Meilisearch `trails` index | Filter evaluation | ✓ (server-side, runtime) | — | None — but no change needed; `subcategory_id` already filterable |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None (FA icon misses fall back to `Icons.category` by design).

## Sources

### Primary (HIGH confidence)
- `db/main.go:335-352` — Meilisearch filterable attributes (`category_id`, `subcategory_id`); config applied on every boot
- `db/util/meilisearch.go:18-103` — trail document fields (`category` name, `category_id`, `subcategory_id`)
- `web/src/lib/stores/trail_store.ts:862-908` — canonical category/subcategory OR-group filter builder
- `web/src/lib/util/category_util.ts` — `displayCategoryIcon`, `displaySubcategoryIcon`, `displaySubcategoryBadgeIcon`, `subcategoryVisible`, `categoryVisibleInDesign`
- `web/src/lib/models/trail.ts:216-219` — `category_id`, `subcategory_id` indexed fields
- `app/lib/models/trail.dart` — `TrailFilter` + `toFilterText()` (named `toFilterText`, not `toFilterString`)
- `app/lib/routes/trail_filter_screen.dart`, `app/lib/components/trail/trail_quick_filter_bar.dart` — UI to modify
- `app/lib/components/base/wanderer_filter_chip.dart` — chip to extend
- `app/lib/models/{category,subcategory,subcategory_preference}.dart` — Phase 10 models + display extensions
- `app/lib/provider/trail/{category,subcategory}_provider.dart`, `app/lib/provider/{category,subcategory}_preference_provider.dart` — provider shapes (sync vs AsyncValue)
- `app/lib/provider/trail/*.g.dart` — generated provider names (`categoryProvider`, `subcategoryProvider`, `categoryPreferenceProvider`, `subcategoryPreferenceProvider`)
- `app/lib/util/icon_util.dart:1018` — `fontAwesomeIconsMap` (FA name → `FaIconData`, no `fa-` prefix in keys)
- `app/pubspec.yaml` — dependency versions

### Secondary (MEDIUM confidence)
- `db/migrations/1742167033_init_meilisearch.go:32` — legacy `category` filterable attribute (now superseded)
- `web/src/lib/server/category_preference_filter.ts` — server-side visibility filtering (context only; Flutter receives pre-filtered trails)

### Tertiary (LOW confidence)
- Flutter `FilterChip` avatar clipping behavior (Pitfall 5) — to confirm on-device

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all deps verified in pubspec; nothing to install
- Architecture: HIGH — both filter surfaces and all providers read directly from source
- Pitfalls: HIGH for #1–#4 (verified in repo), LOW for #5 (visual, needs device)

**Research date:** 2026-06-29
**Valid until:** 2026-07-29 (stable; local codebase, no fast-moving external deps). Re-verify only if the Meilisearch index config in `db/main.go` changes.
