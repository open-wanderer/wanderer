# Phase 10: Category & Subcategory Data Layer - Research

**Researched:** 2026-06-29
**Domain:** Flutter/Dart data layer — freezed models, ObjectBox entities, Riverpod providers mirroring web PR #1059
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**CategoryEntity Update**
- **D-01:** Extend `CategoryEntity` with `icon` (String?), `short_name` (String?), and `translationsJson` (String? — JSON-encoded map) to match the updated Category model. Same approach as `SettingsEntity.notificationsJson` for the JSON blob.
- **D-02:** `CategoryNotifier` writes to ObjectBox on **every** successful API fetch (overwrite all CategoryEntity rows). No staleness tracking — always fresh after a successful call.

**SubcategoryNotifier Startup Behavior**
- **D-03:** `SubcategoryNotifier.build()` reads from ObjectBox first (returns cached subcategories immediately), then triggers a background refresh from `/subcategory` and writes fresh results back to ObjectBox. Subcategories are available at app start without waiting for an API call — satisfies "surviving app restarts" (CAT-03).
- **D-04:** `keepAlive: true` — subcategories are a shared reference dataset (used by filter screens, settings screen, and future UI). Consistent with categoryProvider.

**Preference Provider Shape**
- **D-05:** `CategoryPreferenceNotifier` returns `List<CategoryPreference>`. `SubcategoryPreferenceNotifier` returns `List<SubcategoryPreference>`. List is the natural shape — consumers sort by priority as needed.
- **D-06:** Both providers are **parameterless** (always current user, like `SettingsNotifier`). The API endpoints `/user-category-preference` and `/user-subcategory-preference` scope to the authenticated user implicitly.
- **D-07:** Both providers return **empty list** when the user is not logged in (no API call made). Consumers treat an empty preference list as "all visible" — graceful degradation with no AsyncError boilerplate.
- **D-08:** Both providers are `keepAlive: true`.

**Settings.category Removal**
- **D-09:** Remove `String? category` from `Settings` freezed model, `SettingsEntity`, and `SettingsEntity.fromModel()`/`toModel()`. Regenerate `objectbox-model.json` — ObjectBox drops removed properties gracefully on next app open (no migration script needed).
- **D-10:** A compiler-sweep catches all remaining call sites. Known files: `trail_filter_provider.dart`, `global_search_provider.dart`, `global_search_models.dart`, `trail_quick_filter_bar.dart`, `trail_filter_screen.dart`, `settings_entity.dart`, `trail_entity.dart`.

> ⚠️ **Research correction to D-10 — see Common Pitfall 1.** Verification of the codebase shows that of these files, **only `settings_entity.dart` actually references the `Settings.category` field.** The other files reference `TrailFilter.category`, `GlobalSearchState.category`, or `trail.expand.category` — entirely separate fields that MUST NOT be modified. The "compiler-sweep" approach (D-10) remains correct and is the safe way to find the true call sites, but the planner must not pre-emptively edit the listed files; let the compiler identify them.

### Claude's Discretion

None explicitly delegated; standard Flutter idioms apply within the locked decisions.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. (Filter UI is Phase 11; Settings Categories screen is Phase 12.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CAT-01 | Category model gains `icon`, `short_name`, `translations`; display name resolved by active locale with fallback to English then raw `name` | Web `category.ts` field shape confirmed; `localeProvider` (returns `Locale?`) in `local_settings_provider.dart` supplies active locale; CategoryTranslation nested type pattern via `navigate_response.dart` example |
| CAT-02 | Subcategory freezed model with `id`, `category` (parent ID), `name`, `short_name`, `icon`, `badge_icon`, `translations` | Web `subcategory.ts` confirmed exact field shape |
| CAT-03 | SubcategoryEntity in ObjectBox with indexed `id` and `category` | `CategoryEntity` `@Index @Unique` pattern + JSON-blob pattern from `SettingsEntity.notificationsJson` |
| CAT-04 | SubcategoryNotifier fetches all from `/subcategory` | `/subcategory` returns `ListResult` (paginated `.items`); mirror `CategoryNotifier`; read-from-cache-then-refresh per D-03 |
| CAT-05 | `Settings.category` removed from model and all call sites | Only real call site is `settings_entity.dart` (verified); compiler-sweep finds any others |
| SETCAT-03 | `CategoryPreference` & `SubcategoryPreference` freezed models with `id?`, `user`, `category`/`subcategory`, `visible?`, `priority?` | Web `category_preference.ts` / `subcategory_preference.ts` confirmed exact shape |
| SETCAT-04 | CategoryPreferenceNotifier fetches `GET /user-category-preference`, upserts `PUT /user-category-preference` | GET returns bare JSON **array** (not ListResult); `[]` for anonymous; PUT body `{category, visible}`, returns saved single record |
| SETCAT-05 | SubcategoryPreferenceNotifier fetches `GET /user-subcategory-preference`, upserts `PUT /user-subcategory-preference` | Same array shape; PUT body `{subcategory, visible}` |
</phase_requirements>

## Summary

This is a code-only Flutter/Dart data-layer phase that mirrors a shipped web PR (#1059). There is no new external dependency, no server-side work, and no UI — the entire scope is freezed models, ObjectBox entities, Riverpod (code-gen) providers, and a `Settings.category` field removal. All required API endpoints already exist server-side and were inspected during this research. The established patterns this phase needs (freezed 3.x with `@JsonSerializable(explicitToJson: true)`, ObjectBox `@Index/@Unique` entities, JSON-blob storage via `notificationsJson`, keepAlive providers that read ObjectBox in `build()`) are all already present in the codebase and can be copied directly.

The single most important research finding is a **correction to the field-removal sweep (D-10)**: the codebase contains many `*.category` references, but the only one that touches the `Settings` model's `category` field is `settings_entity.dart`. Every other listed file references a *different* `category` field (`TrailFilter.category`, `GlobalSearchState.category`, `trail.expand.category`) that is unrelated to the favourite-sport field and must be left untouched. The correct procedure is the compiler-driven sweep already chosen in D-10 — but the planner must instruct executors to let `flutter analyze` / `build_runner` reveal the real breakages rather than blindly editing the listed files.

Two API response-shape nuances drive provider implementation: `/category` and `/subcategory` return a paginated `ListResult` (use `.items`), whereas `/user-category-preference` and `/user-subcategory-preference` GET return a **bare JSON array** and an **empty array for anonymous users**. This difference dictates two different parsing strategies in the providers.

**Primary recommendation:** Copy the existing freezed/ObjectBox/Riverpod patterns verbatim. Create `CategoryTranslation` as a nested freezed type, add `@JsonSerializable(explicitToJson: true)` to every model with a nested object or `Map<String, CategoryTranslation>`, store translations as a JSON-encoded string in ObjectBox, and run `dart run build_runner build --delete-conflicting-outputs` after every model/entity change. Drive the `Settings.category` removal with the compiler, not with the D-10 file list.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Category/Subcategory data model | Mobile App (Flutter — models) | — | Pure client-side domain types mirroring server JSON |
| Local persistence / cache | Mobile App (Flutter — ObjectBox entities) | — | ObjectBox is the on-device store; survives restarts |
| Fetch + cache orchestration | Mobile App (Flutter — Riverpod providers) | API/Backend (read-only) | Providers call existing read endpoints; no backend changes |
| Locale-aware name resolution | Mobile App (Flutter — model extension/helper) | — | Resolution uses `localeProvider`; pure client logic |
| Preference fetch/upsert | Mobile App (Flutter — Riverpod providers) | API/Backend (existing endpoints) | `/user-(sub)category-preference` already implemented server-side |
| API endpoints (`/subcategory`, `/user-*-preference`) | API/Backend (SvelteKit + PocketBase) | — | Already shipped in web PR #1059; **no work this phase** |

## Standard Stack

This phase uses only dependencies already present in `app/pubspec.yaml`. No new packages are installed.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| freezed / freezed_annotation | 3.2.5 / 3.1.0 | Immutable data models with codegen | Existing model convention across the app |
| json_serializable / json_annotation | 6.13.0 / 4.11.0 | JSON (de)serialization codegen | Pairs with freezed for `fromJson`/`toJson` |
| objectbox / objectbox_generator | 5.3.1 | On-device persistence | Existing local DB for entities (CategoryEntity, SettingsEntity) |
| flutter_riverpod / riverpod_annotation / riverpod_generator | 3.3.1 / 4.0.2 / 4.0.3 | State management + codegen providers | Existing provider convention (`@riverpod`, `@Riverpod(keepAlive: true)`) |
| build_runner | 2.13.1 | Runs all codegen | Required after any model/entity/provider change |
| dio | (via apiProvider) | HTTP client | `apiProvider` wraps Dio; all API calls go through it |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| dart:convert (`jsonEncode`/`jsonDecode`) | SDK | JSON blob (de)serialization for ObjectBox | Encoding `translations` map into `translationsJson` and back |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| JSON-blob `translationsJson` string | ObjectBox `@Property(type: PropertyType.flex)` or a related entity | Locked by D-01 to mirror `notificationsJson`; a relation would over-engineer a read-only reference map |

**Installation:** None — no new dependencies. After any code change:
```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

**Version verification:** All versions read directly from `app/pubspec.yaml` on 2026-06-29 [VERIFIED: app/pubspec.yaml]. No registry lookup needed since nothing is being installed.

## Package Legitimacy Audit

**No external packages are installed in this phase.** All libraries used (freezed, objectbox, riverpod, dio, json_serializable, build_runner) are pre-existing project dependencies already present in `app/pubspec.yaml` and in active use. The Package Legitimacy Gate is **not applicable** — no new install occurs.

## Architecture Patterns

### System Architecture Diagram

```
App start
   │
   ├─► CategoryNotifier.build() ───► GET /category (ListResult.items)
   │        │                              │
   │        └─► write all CategoryEntity rows to ObjectBox (D-02)
   │                                       │
   │                                       └─► return List<Category>
   │
   ├─► SubcategoryNotifier.build()
   │        │
   │        ├─(1)─► read ObjectBox (SubcategoryEntity.getAll) ──► return cached List<Subcategory> immediately (D-03)
   │        │
   │        └─(2 async)─► GET /subcategory (ListResult.items)
   │                          │
   │                          └─► overwrite SubcategoryEntity rows ──► update state
   │
   ├─► CategoryPreferenceNotifier.build()
   │        │
   │        ├─ user == null ──► return [] (D-07, no API call)
   │        └─ user != null ──► GET /user-category-preference (bare JSON array) ──► List<CategoryPreference>
   │                              upsert: PUT /user-category-preference {category, visible} ──► saved record
   │
   └─► SubcategoryPreferenceNotifier.build()  (same shape, /user-subcategory-preference, {subcategory, visible})

Display-name resolution (CAT-01, pure client):
   Category + localeProvider(Locale?) ──► translations[locale.languageCode]?.name
                                          ?? translations['en']?.name
                                          ?? name
```

### Recommended Project Structure
```
app/lib/
├── models/
│   ├── category.dart              # extend: add short_name, icon, translations + CategoryTranslation nested type
│   ├── subcategory.dart           # NEW: Subcategory freezed model
│   ├── category_preference.dart   # NEW: CategoryPreference freezed model
│   ├── subcategory_preference.dart# NEW: SubcategoryPreference freezed model
│   └── settings.dart              # remove String? category (D-09)
├── entities/
│   ├── category_entity.dart       # extend: icon, short_name, translationsJson (D-01)
│   ├── subcategory_entity.dart    # NEW: @Index id + @Index category (CAT-03)
│   └── settings_entity.dart       # remove category field + fromModel/toModel lines (D-09)
├── provider/
│   ├── trail/
│   │   ├── category_provider.dart       # update: write to ObjectBox on fetch (D-02)
│   │   └── subcategory_provider.dart    # NEW: read-cache-then-refresh (D-03/D-04)
│   └── category_preference_provider.dart    # NEW (SETCAT-04)
│   └── subcategory_preference_provider.dart # NEW (SETCAT-05)
└── objectbox-model.json           # regenerated by objectbox_generator
```

### Pattern 1: Freezed model with nested object / map (CRITICAL)
**What:** Any freezed model containing a nested freezed type or a `Map<String, NestedType>` MUST place `@JsonSerializable(explicitToJson: true)` on the **factory constructor line** (not above `@freezed`).
**When to use:** `Category` (has `Map<String, CategoryTranslation>`), `Subcategory` (same), `CategoryTranslation` itself if nested.
**Example:**
```dart
// Source: app/lib/models/navigate_response.dart:24-29 [VERIFIED: codebase]
@freezed
abstract class NavigateResponse with _$NavigateResponse {
  @JsonSerializable(explicitToJson: true)   // ON the factory line
  const factory NavigateResponse({
    required List<NavigateManeuver> maneuvers,
    required List<List<double>> shape,
  }) = _NavigateResponse;

  factory NavigateResponse.fromJson(Map<String, dynamic> json) =>
      _$NavigateResponseFromJson(json);
}
```
Apply the same shape to `Category`:
```dart
// Mirrors web category.ts [CITED: web/src/lib/models/category.ts]
@freezed
abstract class CategoryTranslation with _$CategoryTranslation {
  const factory CategoryTranslation({String? name, String? short_name}) = _CategoryTranslation;
  factory CategoryTranslation.fromJson(Map<String, dynamic> json) =>
      _$CategoryTranslationFromJson(json);
}

@freezed
abstract class Category with _$Category {
  @JsonSerializable(explicitToJson: true)
  const factory Category({
    required String id,
    required String name,
    String? short_name,
    String? icon,
    Map<String, CategoryTranslation>? translations,
  }) = _Category;
  factory Category.fromJson(Map<String, dynamic> json) => _$CategoryFromJson(json);
}
```
> Note: current `Category` has an `img` field (`app/lib/models/category.dart`) that is NOT in the web model. Verify whether `img` is still consumed (e.g. category icons in UI) before removing it — keep it if any call site reads `category.img`. Web PR uses `icon` (a string name), not `img`.

### Pattern 2: ObjectBox entity with JSON-blob field
**What:** Store a map as a JSON-encoded `String?` column; encode in `fromModel`, decode in `toModel`.
**When to use:** `CategoryEntity.translationsJson`, `SubcategoryEntity.translationsJson`.
**Example:**
```dart
// Source: app/lib/entities/settings_entity.dart:50-91 [VERIFIED: codebase]
// write:
notificationsJson: settings.notifications != null
    ? jsonEncode(settings.notifications!.map((k, v) => MapEntry(k, v.toJson())))
    : null,
// read:
final decoded = jsonDecode(notificationsJson!) as Map<String, dynamic>;
notifs = decoded.map((k, v) =>
    MapEntry(k, NotificationPreference.fromJson(v as Map<String, dynamic>)));
```

### Pattern 3: ObjectBox entity with indexed unique id
**What:** `@Id() int obxId`, plus `@Index() @Unique(onConflict: ConflictStrategy.replace) String id`. CAT-03 also requires an index on `category`.
**Example:**
```dart
// Source: app/lib/entities/category_entity.dart [VERIFIED: codebase]
@Entity()
class SubcategoryEntity {
  @Id() int obxId = 0;
  @Index() @Unique(onConflict: ConflictStrategy.replace) String id;
  @Index() String category;          // CAT-03 requires index on parent id
  String name;
  String? shortName;
  String? icon;
  String? badgeIcon;
  String? translationsJson;
  // ... constructor, fromModel, toModel extension
}
```

### Pattern 4: keepAlive provider that reads ObjectBox in build()
**What:** `@Riverpod(keepAlive: true)` notifier whose `build()` reads from a Box; mutations call `ref.invalidateSelf()`.
**Example:** `app/lib/provider/settings_provider.dart` (`SettingsNotifier`) is the canonical model. For `SubcategoryNotifier`, `build()` returns cached list synchronously-ish, then an async fetch overwrites the box and updates state (D-03).
```dart
// Source: app/lib/provider/settings_provider.dart:16-19 [VERIFIED: codebase]
@override
Settings? build() {
  final box = ref.watch(objectBoxProvider).box<SettingsEntity>();
  return box.getAll().firstOrNull?.toModel();
}
```

### Pattern 5: Preference provider — empty-list-for-anonymous
**What:** `build()` checks current user; if not logged in, return `[]` without an API call (D-07). The GET endpoint also returns `[]` for anonymous, so this is doubly safe.
**Example:**
```dart
// GET shape [VERIFIED: web/src/routes/api/v1/user-category-preference/+server.ts]
// returns json(preferences) — a BARE ARRAY, not a ListResult
final response = await api.get('/user-category-preference');
final list = (response.data as List)
    .map((e) => CategoryPreference.fromJson(e as Map<String, dynamic>))
    .toList();
// upsert:
await api.put('/user-category-preference', data: {'category': id, 'visible': v});
```
> Determine the "current user" gate consistently with how other providers detect auth (check `auth_provider` / `currentUser`). The web endpoint keys off `event.locals.user`; on mobile, mirror whatever existing auth signal the app uses.

### Anti-Patterns to Avoid
- **Parsing preferences as `ListResult`:** `/user-*-preference` returns a bare array, NOT `{items, page, ...}`. Using `ListResult.fromJson` will throw.
- **Editing every `*.category` reference during CAT-05:** Most are unrelated fields. See Pitfall 1.
- **Putting `@JsonSerializable` above `@freezed`:** It must be on the factory constructor line, or nested-object serialization silently breaks. [VERIFIED: STATE.md decision log + navigate_response.dart]
- **Hand-writing an ObjectBox migration for the dropped `Settings.category`:** Unnecessary — ObjectBox drops removed properties on next open (D-09).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON (de)serialization | Manual `fromJson`/`toJson` | freezed + json_serializable codegen | Existing convention; nested map handling is error-prone by hand |
| List API response parsing | Custom pagination parser | `ListResult.fromJson` (`app/lib/models/list_result.dart`) | Already generic over `T`; used by CategoryNotifier |
| HTTP client/cookies | Raw `HttpClient` | `apiProvider` (Dio + cookie jar) | Centralized base URL + auth cookies |
| ObjectBox schema IDs | Manually editing `objectbox-model.json` | `objectbox_generator` via build_runner | Generator manages crucial property IDs; manual edits corrupt the model |
| Locale plumbing | New locale store | `localeProvider` (`local_settings_provider.dart`) | Returns `Locale?` from `settings.language`; already wired (Phase 6) |

**Key insight:** Every primitive this phase needs already exists in the codebase as a copyable pattern. The work is mechanical replication plus a compiler-driven field removal — not novel engineering.

## Runtime State Inventory

> This is a refactor phase (removing `Settings.category`). Runtime-state audit below.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `SettingsEntity.category` column persisted in ObjectBox on device. New `CategoryEntity`/`SubcategoryEntity` translation/icon columns added. | Code edit only. ObjectBox drops the removed `category` property on next app open (D-09) — no migration script. New columns auto-created. |
| Live service config | None — no external service stores this string. | None. |
| OS-registered state | None — no OS-level registration involves the category field. | None. |
| Secrets/env vars | None — no secret or env var references `Settings.category`. | None. |
| Build artifacts | Generated files: `settings.freezed.dart`, `settings.g.dart`, `category.freezed.dart`, `category.g.dart`, `*_entity` references in `objectbox.g.dart`, `objectbox-model.json`. All stale after model/entity edits. | Run `dart run build_runner build --delete-conflicting-outputs` to regenerate ALL of them. Commit regenerated `objectbox-model.json`. |

**The canonical question — after every repo file is updated, what runtime systems still hold the old string?** Only the on-device ObjectBox database holds a `Settings.category` value, and ObjectBox discards it gracefully on the next open. No server, secret, or OS state is involved. **Verified by grep across `app/lib` and inspection of `settings_entity.dart`.**

## Common Pitfalls

### Pitfall 1: Over-broad `Settings.category` sweep (HIGH IMPACT)
**What goes wrong:** D-10 lists `trail_filter_provider.dart`, `global_search_provider.dart`, `global_search_models.dart`, `trail_quick_filter_bar.dart`, `trail_filter_screen.dart`, `trail_entity.dart` as `Settings.category` call sites. Editing their `category` references will break unrelated, correct functionality.
**Why it happens:** These files DO contain `category`, but it is `TrailFilter.category` (a `List` of category IDs, defaulted to `[]` in `trail_filter_provider.dart:28`), `GlobalSearchState.category` (a selected-category string), or `trail.expand.category` (the trail's own category relation). **None of them read `Settings.category`.** Verified: the only file that references the `Settings` model's `category` field is `app/lib/entities/settings_entity.dart` (lines 30 and 45).
**How to avoid:** Do the removal compiler-first (which D-10's "compiler-sweep" already specifies). Remove `category` from `settings.dart` and `settings_entity.dart`, then run `dart run build_runner build` and `flutter analyze`. Fix ONLY the files the compiler flags as broken. Do NOT pre-edit the D-10 file list.
**Warning signs:** A diff that touches `TrailFilter`, `GlobalSearchState`, or `trail.expand.category` logic — those are out of scope and indicate the wrong `category` was edited.

### Pitfall 2: Wrong API response parsing for preferences
**What goes wrong:** Parsing `/user-category-preference` GET as a `ListResult` throws, because it returns a bare JSON array (`json(preferences)`), not `{items, ...}`.
**Why it happens:** `/category` and `/subcategory` DO return `ListResult`; the preference endpoints do not. Easy to copy the wrong parser.
**How to avoid:** Parse preferences as `(response.data as List).map(...)`. Parse categories/subcategories with `ListResult.fromJson(...).items`. [VERIFIED: web `+server.ts` files]
**Warning signs:** `type 'List<dynamic>' is not a subtype of 'Map<String, dynamic>'` at runtime.

### Pitfall 3: `@JsonSerializable(explicitToJson: true)` misplacement
**What goes wrong:** Nested `CategoryTranslation` map serializes as `[Instance of 'CategoryTranslation']` or fails round-trip.
**Why it happens:** freezed 3.x requires the annotation on the factory constructor line, not above `@freezed`. [VERIFIED: STATE.md decision; navigate_response.dart]
**How to avoid:** Copy the exact placement from `navigate_response.dart`.
**Warning signs:** `toJson()` output contains non-JSON instance strings, or `fromJson` of a re-encoded blob fails.

### Pitfall 4: `priority` is nullable int, `visible` is nullable bool
**What goes wrong:** Modeling `priority` as required or non-nullable breaks parsing when the server omits it.
**Why it happens:** Web schema declares `visible?: boolean` and `priority?: number | null`. [VERIFIED: web preference models]
**How to avoid:** `bool? visible`, `int? priority`, `String? id` in the freezed models. `user` and `category`/`subcategory` are required.

### Pitfall 5: Forgetting to regenerate / commit `objectbox-model.json`
**What goes wrong:** Adding/removing entity properties without regenerating corrupts the ObjectBox schema or fails to build.
**How to avoid:** Always run build_runner after entity changes; commit the regenerated `objectbox-model.json` (it tracks crucial property IDs). [CITED: objectbox-model.json `_note1`]

## Code Examples

### Subcategory model (CAT-02)
```dart
// Mirrors web/src/lib/models/subcategory.ts [CITED: web subcategory.ts]
@freezed
abstract class Subcategory with _$Subcategory {
  @JsonSerializable(explicitToJson: true)
  const factory Subcategory({
    required String id,
    required String category,        // parent category id
    required String name,
    String? short_name,
    String? icon,
    String? badge_icon,
    Map<String, CategoryTranslation>? translations,
  }) = _Subcategory;
  factory Subcategory.fromJson(Map<String, dynamic> json) =>
      _$SubcategoryFromJson(json);
}
```
> Field naming: web uses snake_case (`short_name`, `badge_icon`). Keep snake_case in Dart factory params OR use `@JsonKey(name: 'short_name')` with camelCase Dart names — match whatever the existing models do (current `Category` uses bare `img`; `NavigateManeuver` uses `@JsonKey(name: 'begin_shape_index')`). Decide one convention and apply consistently.

### Preference models (SETCAT-03)
```dart
// [CITED: web category_preference.ts / subcategory_preference.ts]
@freezed
abstract class CategoryPreference with _$CategoryPreference {
  const factory CategoryPreference({
    String? id,
    required String user,
    required String category,
    bool? visible,
    int? priority,
  }) = _CategoryPreference;
  factory CategoryPreference.fromJson(Map<String, dynamic> json) =>
      _$CategoryPreferenceFromJson(json);
}
// SubcategoryPreference: identical but `required String subcategory` instead of `category`.
```

### Locale-aware display name (CAT-01)
```dart
// Resolution chain: active locale -> 'en' -> raw name [CITED: CONTEXT specifics]
extension CategoryDisplay on Category {
  String displayName(Locale? locale) {
    final code = locale?.languageCode;
    return translations?[code]?.name
        ?? translations?['en']?.name
        ?? name;
  }
}
// Consumer: final name = category.displayName(ref.watch(localeProvider));
```
> `localeProvider` returns `Locale?` (null when no language set). [VERIFIED: app/lib/provider/local_settings_provider.dart:45-50]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `Settings.category` (single favourite-sport string) | Per-user category/subcategory preferences with `visible`/`priority` | Web PR #1059 | This phase removes the old field and adds the new preference models/providers |
| `Category` with only `id`, `name`, `img` | `Category` with `short_name`, `icon`, locale `translations` | Web PR #1059 | Model extension in this phase |

**Deprecated/outdated:**
- `Settings.category` field — replaced by category preferences (CAT-05).
- Possibly `Category.img` — verify against current consumers before removing; web model uses `icon` instead.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The "current user" gate for preference providers can reuse the app's existing auth signal (e.g. `currentUser`/`auth_provider`) | Pattern 5 | If no clean client-side auth signal exists, provider must rely on the server's `[]`-for-anonymous response (still safe per D-07) |
| A2 | `Category.img` may still be consumed by UI and should be retained unless verified unused | Pattern 1 / State of the Art | Removing it could break category icon rendering; planner should add a verification step |
| A3 | Snake_case vs camelCase field naming convention should follow existing models (mixed in codebase) | Code Examples | Inconsistent naming causes `@JsonKey` mismatches and failed parsing; pick one convention explicitly during planning |
| A4 | The `/subcategory` endpoint returns translations/icon/badge_icon inline (no `expand` needed) | CAT-04 | If fields require `?expand=`, the provider must add a query param; web `Subcategory` shows them as direct fields, so low risk |

**If A1–A4 are confirmed during planning, no further user input is needed — all field shapes and API contracts are verified against the shipped web PR.**

## Open Questions (RESOLVED)

1. **Field naming convention (snake_case vs camelCase)**
   - What we know: Web uses snake_case; existing Dart models are mixed (`Category.img` bare, `NavigateManeuver` uses `@JsonKey`).
   - What's unclear: Which to standardize on for the new models.
   - RESOLVED: Use camelCase Dart field names with `@JsonKey(name: 'snake_case')` annotations for any field whose JSON key differs from its Dart name (e.g. `@JsonKey(name: 'short_name') final String shortName`). Applied consistently to all four new models in Plan 01 Task 1.

2. **`Category.img` retention**
   - What we know: Current model has `img`; web model has `icon` instead.
   - What's unclear: Whether any call site reads `category.img`.
   - RESOLVED: `Category.img` has zero call sites (verified by grep during planning). Remove `img` and add `icon` as the canonical field. Implemented in Plan 01 Task 1.

3. **Mobile auth signal for preference providers**
   - What we know: Server returns `[]` for anonymous; D-07 wants no API call when logged out.
   - What's unclear: The exact client-side "is logged in" provider to watch.
   - RESOLVED: Use `ref.read(authProvider).value == null` as the logged-out gate (pattern from `list_search_provider.dart:61`). When null, return `[]` immediately without making an API call. Implemented in Plan 03 Task 2.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter/Dart SDK | All Dart codegen + build | ✓ (project requirement) | 3.11.5 / 3.11.5 | — |
| build_runner (dev dep) | freezed/json/objectbox/riverpod codegen | ✓ | 2.13.1 | — |
| `/subcategory` endpoint | CAT-04 | ✓ (web PR #1059, verified server-side route) | — | — |
| `/user-category-preference` GET/PUT | SETCAT-04 | ✓ (verified `+server.ts`) | — | — |
| `/user-subcategory-preference` GET/PUT | SETCAT-05 | ✓ (verified `+server.ts`) | — | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None. All endpoints exist; all tooling is in `pubspec.yaml`.

## Security Domain

> `security_enforcement: true` in config. This phase adds no auth/crypto/session logic — it is a read-mostly data layer over existing authenticated endpoints. The endpoints already enforce auth server-side (`event.locals.user`).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth logic added; relies on existing Dio cookie session via `apiProvider` |
| V3 Session Management | no | Session handled by existing cookie jar; unchanged |
| V4 Access Control | yes (indirect) | Server scopes preferences to `event.locals.user`; client must NOT send `user` in PUT body (server injects it). Client returns `[]` when logged out (D-07) |
| V5 Input Validation | yes | Server validates with zod (`UserCategoryPreferenceUpsertSchema`: `{category: string(15), visible: boolean}`). Client must send only `{category, visible}` / `{subcategory, visible}` |
| V6 Cryptography | no | No crypto in scope |

### Known Threat Patterns for Flutter data layer over PocketBase

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Client sends arbitrary `user` id in preference PUT | Elevation of Privilege | Server ignores client `user` and injects `event.locals.user.id` (verified). Client must not include `user` in upsert payload. |
| Cached subcategory data tampered on device | Tampering | Low impact — reference data only, refreshed from server on next launch (D-03). No trust decisions made from cached values. |
| Parsing malformed API JSON | Tampering/DoS | freezed/json_serializable parsing with nullable fields; wrap fetch in try/catch like existing `CategoryNotifier` |

## Sources

### Primary (HIGH confidence)
- `web/src/lib/models/category.ts`, `subcategory.ts`, `category_preference.ts`, `subcategory_preference.ts` — exact field shapes [VERIFIED: codebase]
- `web/src/lib/models/api/category_preference_schema.ts`, `subcategory_preference_schema.ts` — upsert/reorder zod schemas [VERIFIED: codebase]
- `web/src/routes/api/v1/subcategory/+server.ts` — returns `ListResult` [VERIFIED: codebase]
- `web/src/routes/api/v1/user-category-preference/+server.ts` — GET bare array / `[]` anonymous, PUT `{category, visible}` returns saved record [VERIFIED: codebase]
- `app/lib/models/category.dart`, `settings.dart`, `navigate_response.dart`, `list_result.dart` — existing freezed patterns [VERIFIED: codebase]
- `app/lib/entities/category_entity.dart`, `settings_entity.dart` — ObjectBox entity + JSON-blob patterns [VERIFIED: codebase]
- `app/lib/provider/trail/category_provider.dart`, `settings_provider.dart`, `local_settings_provider.dart` — provider patterns + `localeProvider` [VERIFIED: codebase]
- `app/pubspec.yaml` — dependency versions [VERIFIED: codebase]
- grep across `app/lib` for `Settings.category` call sites [VERIFIED: codebase]

### Secondary (MEDIUM confidence)
- None — all claims verified against codebase.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all from `pubspec.yaml`, nothing installed.
- Architecture: HIGH — every pattern copied from existing verified files.
- Pitfalls: HIGH — Pitfall 1 (sweep over-reach) and Pitfall 2 (array vs ListResult) verified by direct file inspection.
- API contracts: HIGH — verified against shipped web `+server.ts` routes.

**Research date:** 2026-06-29
**Valid until:** 2026-07-29 (stable — internal patterns + shipped web PR; low churn)
