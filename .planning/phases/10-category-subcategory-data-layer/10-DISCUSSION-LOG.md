# Phase 10: Category & Subcategory Data Layer - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-29
**Phase:** 10-category-subcategory-data-layer
**Areas discussed:** CategoryEntity update scope, SubcategoryNotifier startup behavior, Preference provider shape, Settings.category removal scope

---

## CategoryEntity Update Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Extend it | Add icon, short_name, translationsJson to CategoryEntity; CategoryNotifier writes to ObjectBox on fetch | ✓ |
| Leave it lean | Only update freezed model and provider; entity stays as id + name only | |

**User's choice:** Extend it

| Option | Description | Selected |
|--------|-------------|----------|
| On every successful fetch | Always overwrites CategoryEntity with fresh data | ✓ |
| First fetch only | Writes to ObjectBox once per build(); skips subsequent writes | |

**User's choice:** On every successful fetch
**Notes:** Mirrors trail/waypoint sync pattern; simpler, no staleness logic needed.

---

## SubcategoryNotifier Startup Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Read from ObjectBox first | build() returns cached subcategories immediately, then background-refresh from API | ✓ |
| Fetch from API, then cache | build() always goes to API; subcategories unavailable until first API call succeeds | |

**User's choice:** Read from ObjectBox first

| Option | Description | Selected |
|--------|-------------|----------|
| keepAlive: true | Subcategories stay loaded across route changes | ✓ |
| Auto-dispose | Provider recreated when listeners detach; re-fetches on each re-entry | |

**User's choice:** keepAlive: true
**Notes:** Consistent with categoryProvider; subcategories are shared reference data.

---

## Preference Provider Shape

| Option | Description | Selected |
|--------|-------------|----------|
| List<CategoryPreference> | Consistent with CategoryNotifier; consumers sort as needed | ✓ |
| Map<String, CategoryPreference> | O(1) lookup by category ID; adds transformation on provider side | |

**User's choice:** List<CategoryPreference>

| Option | Description | Selected |
|--------|-------------|----------|
| Parameterless — always current user | Like SettingsNotifier; API scopes to auth user implicitly | ✓ |
| Parameterized by user ID | Useful if viewing others' preferences (not needed here) | |

**User's choice:** Parameterless — always current user

| Option | Description | Selected |
|--------|-------------|----------|
| Empty list when unauthenticated | No API call; consumers treat empty as "all visible" | ✓ |
| Throw / AsyncError | Forces callers to handle unauthenticated state explicitly | |

**User's choice:** Empty list when unauthenticated
**Notes:** Same shape and behavior applies to SubcategoryPreferenceNotifier.

---

## Settings.category Removal Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Remove from SettingsEntity + regen model | ObjectBox drops removed properties gracefully; no migration script needed | ✓ |
| Keep as nullable stub | Avoid schema change risk; leaves dead code | |

**User's choice:** Remove from SettingsEntity + regen model

| Option | Description | Selected |
|--------|-------------|----------|
| Just a call-site sweep is fine | Compiler catches remaining references after field removal | ✓ |
| Need to check trail_entity.dart carefully | May store category ID on trail entity separately | |

**User's choice:** Just a call-site sweep is fine
**Notes:** trail_entity.dart should still be inspected to confirm whether it references Settings.category (the settings field) vs. the trail's own category field before applying changes.

---

## Claude's Discretion

None — all gray areas were decided by the user.

## Deferred Ideas

None — discussion stayed within phase scope.
