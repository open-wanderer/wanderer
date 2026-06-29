# Phase 10: Category & Subcategory Data Layer - Context

**Gathered:** 2026-06-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Update the Category freezed model with `icon`, `short_name`, and `translations` fields and make CategoryNotifier cache them in ObjectBox. Create the Subcategory freezed model, ObjectBox entity, and provider (read-from-cache on startup, background-refresh from API). Create CategoryPreference and SubcategoryPreference freezed models and their Riverpod providers. Remove the deprecated `Settings.category` field from the Settings model, SettingsEntity, and all call sites.

</domain>

<decisions>
## Implementation Decisions

### CategoryEntity Update
- **D-01:** Extend `CategoryEntity` with `icon` (String?), `short_name` (String?), and `translationsJson` (String? — JSON-encoded map) to match the updated Category model. Same approach as `SettingsEntity.notificationsJson` for the JSON blob.
- **D-02:** `CategoryNotifier` writes to ObjectBox on **every** successful API fetch (overwrite all CategoryEntity rows). No staleness tracking — always fresh after a successful call.

### SubcategoryNotifier Startup Behavior
- **D-03:** `SubcategoryNotifier.build()` reads from ObjectBox first (returns cached subcategories immediately), then triggers a background refresh from `/subcategory` and writes fresh results back to ObjectBox. Subcategories are available at app start without waiting for an API call — satisfies "surviving app restarts" (CAT-03).
- **D-04:** `keepAlive: true` — subcategories are a shared reference dataset (used by filter screens, settings screen, and future UI). Consistent with categoryProvider.

### Preference Provider Shape
- **D-05:** `CategoryPreferenceNotifier` returns `List<CategoryPreference>`. `SubcategoryPreferenceNotifier` returns `List<SubcategoryPreference>`. List is the natural shape — consumers sort by priority as needed (Phase 12 sorts by priority, Phase 11 checks visibility per chip).
- **D-06:** Both providers are **parameterless** (always current user, like `SettingsNotifier`). The API endpoints `/user-category-preference` and `/user-subcategory-preference` scope to the authenticated user implicitly.
- **D-07:** Both providers return **empty list** when the user is not logged in (no API call made). Consumers treat an empty preference list as "all visible" — graceful degradation with no AsyncError boilerplate.
- **D-08:** Both providers are `keepAlive: true`.

### Settings.category Removal
- **D-09:** Remove `String? category` from `Settings` freezed model, `SettingsEntity`, and `SettingsEntity.fromModel()`/`toModel()`. Regenerate `objectbox-model.json` — ObjectBox drops removed properties gracefully on next app open (no migration script needed).
- **D-10:** A compiler-sweep catches all remaining call sites. Known files: `trail_filter_provider.dart`, `global_search_provider.dart`, `global_search_models.dart`, `trail_quick_filter_bar.dart`, `trail_filter_screen.dart`, `settings_entity.dart`, `trail_entity.dart`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Web Category Model (source of truth for field shape)
- `web/src/lib/models/category.ts` — Category interface: `id`, `name`, `short_name?`, `icon?`, `translations?: Record<string, CategoryTranslation>`. CategoryTranslation has `name?` and `short_name?`.
- `web/src/lib/models/subcategory.ts` — Subcategory interface: `id`, `category` (parent ID), `name`, `short_name?`, `icon?`, `badge_icon?`, `translations?: Record<string, CategoryTranslation>`.

### Web Preference Models (source of truth for API shape)
- `web/src/lib/models/subcategory_preference.ts` — UserSubcategoryPreference: `id?`, `user`, `subcategory`, `visible?`, `priority?`.
- `web/src/lib/models/category_preference.ts` (find via `find web/src/lib/models -name "category_preference*"`) — UserCategoryPreference: `id?`, `user`, `category`, `visible?`, `priority?`. Upsert schema: `{category, visible}`. Reorder schema: `{categories: string[]}`.

### Existing Flutter Patterns to Mirror
- `app/lib/models/category.dart` — current Category freezed model (to be extended)
- `app/lib/entities/category_entity.dart` — current CategoryEntity (to be extended with icon/short_name/translationsJson)
- `app/lib/provider/trail/category_provider.dart` — CategoryNotifier pattern (to be updated to write to ObjectBox)
- `app/lib/models/settings.dart` — Settings freezed model (Settings.category field to be removed)
- `app/lib/entities/settings_entity.dart` — SettingsEntity (category field + fromModel/toModel to be removed; also the pattern for JSON blob storage via notificationsJson)
- `app/lib/provider/settings_provider.dart` — SettingsNotifier pattern (read from ObjectBox in build(), updateFromServer(), saveToServer()) — model for preference providers
- `app/lib/objectbox-model.json` — ObjectBox schema (to be regenerated after entity changes)

### Requirements
- `.planning/REQUIREMENTS.md` §v1.3 Requirements > Category Data Layer — CAT-01..05, SETCAT-03..05 (8 requirements for this phase)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ListResult<T>` (`app/lib/models/list_result.dart`) — used by CategoryNotifier for API list responses; use same pattern for SubcategoryNotifier
- `apiProvider` (`app/lib/provider/api_provider.dart`) — Dio HTTP client; `api.get('/subcategory')`, `api.get('/user-category-preference')`, etc.
- `objectBoxProvider` — provides the ObjectBox Store; all entities accessed via `ref.read(objectBoxProvider).box<T>()`
- `@JsonSerializable(explicitToJson: true)` on factory constructor — **critical for freezed 3.x with nested objects** (CategoryTranslation will be a nested type)

### Established Patterns
- **Freezed model pattern:** `@freezed abstract class Foo with _$Foo { const factory Foo({...}) = _Foo; factory Foo.fromJson(...) => _$FooFromJson(json); }` — `@JsonSerializable(explicitToJson: true)` goes on the factory constructor line, NOT above `@freezed`
- **ObjectBox entity pattern:** `@Entity()` class, `@Id() int obxId = 0`, `@Index() @Unique(onConflict: ConflictStrategy.replace) String id`, `fromModel()` factory, `toModel()` extension
- **JSON blob storage in ObjectBox:** `String? translationsJson` stored via `jsonEncode(translations)` / `jsonDecode(translationsJson)` — see `SettingsEntity.notificationsJson` for the exact approach
- **keepAlive provider with ObjectBox read-first:** `SettingsNotifier.build()` reads from `box.getAll().firstOrNull?.toModel()` — SubcategoryNotifier mirrors this but returns a list and also triggers an async refresh
- **Background refresh:** After reading ObjectBox in `build()`, kick off an async fetch and call `ref.invalidateSelf()` or update state directly

### Integration Points
- `app/lib/provider/trail/trail_filter_provider.dart` — references `Settings.category`; must be updated as part of CAT-05 sweep
- `app/lib/routes/trail_filter_screen.dart` — references `Settings.category`; must be updated
- `app/lib/components/trail/trail_quick_filter_bar.dart` — references `Settings.category`; must be updated
- `app/lib/models/global_search_models.dart` + `global_search_provider.dart` — reference `Settings.category`; must be updated
- `app/lib/entities/trail_entity.dart` — listed as a Settings.category call site; verify whether it references the settings field or the trail's own category field before removing

</code_context>

<specifics>
## Specific Ideas

- Locale-aware display name resolution (CAT-01): lookup `translations[activeLocale]?.name`, fallback to `translations['en']?.name`, fallback to raw `name`. Active locale comes from `localeProvider` (already wired from Phase 6).
- `translationsJson` in CategoryEntity is the same pattern as `notificationsJson` in SettingsEntity — JSON-encode the `Map<String, CategoryTranslation>` map on write, decode on read.
- `trail_entity.dart` is in the Settings.category sweep list but may reference the trail's own category field (not Settings.category) — check carefully before modifying it.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 10-category-subcategory-data-layer*
*Context gathered: 2026-06-29*
