---
phase: quick-260720-s7m
plan: 01
subsystem: i18n
tags: [flutter, arb, gen-l10n, crowdin, icu-messageformat]

# Dependency graph
requires: []
provides:
  - 14 app_*.arb files with an identical, dead-key-free 267-key set
  - 43 new l10n keys covering previously hard-coded Dart string literals, wired at call sites
  - @key ICU-placeholder metadata for all 11 surviving plural/placeholder keys in app_en.arb
  - crowdin.yml second file entry mapping app ARB files alongside web's JSON
affects: [app/lib/i18n, future app l10n additions, future Crowdin sync]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ARB key keep/delete inventory via dynamic-usage safety scan (member-access token
       collection) before any deletion, gated as its own task"
    - "Non-en locale backfill: reuse existing web translation via kebab-case key mapping
       (key.replace(/_/g,'-')) when present and non-empty, else English fallback placeholder"
    - "ICU plural placeholders that double as the plural control variable are typed 'num';
       plain interpolated placeholders are typed 'String' or 'int' per content"

key-files:
  created:
    - .planning/quick/260720-s7m-clean-up-arb-translation-files-remove-un/key-inventory.json
  modified:
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_cs.arb
    - app/lib/i18n/app_es.arb
    - app/lib/i18n/app_eu.arb
    - app/lib/i18n/app_fr.arb
    - app/lib/i18n/app_hu.arb
    - app/lib/i18n/app_it.arb
    - app/lib/i18n/app_nl.arb
    - app/lib/i18n/app_no.arb
    - app/lib/i18n/app_pl.arb
    - app/lib/i18n/app_pt.arb
    - app/lib/i18n/app_ru.arb
    - app/lib/i18n/app_zh.arb
    - app/lib/i18n/app_localizations*.dart (regenerated via flutter gen-l10n)
    - crowdin.yml
    - app/lib/components/base/wanderer_error.dart
    - app/lib/components/base/wanderer_layout.dart
    - app/lib/components/base/wanderer_rich_text_editor.dart
    - app/lib/components/route_planner/elevation_tab.dart
    - app/lib/components/route_planner/route_anchor_list_tab.dart
    - app/lib/components/route_planner/settings_tab.dart
    - app/lib/components/trail/elevation_profile.dart
    - app/lib/components/trail/trail_panel.dart
    - app/lib/components/trail/trail_timeline.dart
    - app/lib/routes/global_search_screen.dart
    - app/lib/routes/library_screen.dart
    - app/lib/routes/list_screen.dart
    - app/lib/routes/location_search_screen.dart
    - app/lib/routes/map_screen.dart
    - app/lib/routes/profile_follow_screen.dart
    - app/lib/routes/profile_list_screen.dart
    - app/lib/routes/profile_screen.dart
    - app/lib/routes/profile_trail_screen.dart
    - app/lib/routes/route_planner_screen.dart
    - app/lib/routes/server_selection_screen.dart
    - app/lib/routes/settings_language_screen.dart
    - app/lib/routes/settings_screen.dart
    - app/lib/routes/trail_create_screen.dart

key-decisions:
  - "Safety scan (Task 1) ran and was fully documented before any ARB key was deleted, per the locked decision"
  - "224 of 515 candidate en.arb keys were directly used; 291 dead keys removed from all 14 locales"
  - "Fixed a pre-existing ICU plural-syntax bug (redundant 'one' clause fully shadowed by '=1') in es/eu/pl surfaced by gen-l10n during Task 2, since it blocked the plan's own zero-warnings success criterion"
  - "New placeholder-bearing keys use double-quote literals around interpolated values instead of single quotes, avoiding ICU MessageFormat's single-quote escape character silently swallowing the placeholder"

requirements-completed: [ARB-CLEAN]

# Metrics
duration: ~70min (across 2 sessions; session 2 continued from Task 3 after a connection interruption)
completed: 2026-07-20
---

# Quick Task 260720-s7m: Clean Up ARB Translation Files Summary

**Removed 291 dead ARB keys, equalized all 14 locales to one 267-key set, extracted 43 hard-coded Dart string literals into wired l10n keys, added @key ICU metadata for all 11 surviving placeholder/plural keys, and added the app ARB files to crowdin.yml.**

## Performance

- **Duration:** ~70 min combined (Tasks 1-2 in an earlier session cut off by a connection error; Tasks 3-4 completed in this session)
- **Completed:** 2026-07-20
- **Tasks:** 4/4
- **Files modified:** 14 ARB files + 14 generated `app_localizations_*.dart` files + `crowdin.yml` + 24 Dart call-site files + 1 new working artifact (`key-inventory.json`)

## Accomplishments

- Built an authoritative, safety-scanned keep/delete inventory (224 keep / 291 delete out of 515 candidate `app_en.arb` keys) BEFORE touching any ARB file, confirming no reflection or dynamic-key-construction path exists in the codebase and that `Category`/`Subcategory.displayName` resolve from the server's `translations` map, not ARB keys
- Removed all 291 dead keys from all 14 locale ARB files and equalized every locale to the same 224-key surviving set, backfilling stale locales (which sat at ~408-488 keys) by reusing existing web translations (kebab-case key mapping) where present, else falling back to English
- Found and fixed a real pre-existing ICU plural-syntax bug in es/eu/pl (`one {}` clause fully shadowed by `=1`, per CLDR — Dart's intl codegen always evaluates exact-value matches ahead of keyword categories) that was surfaced by `flutter gen-l10n` warnings once those keys were touched
- Extracted 43 hard-coded user-facing Dart string literals (empty states, dialog/tooltip text, rich-text-editor toolbar, hint texts, a hardcoded " & " conjunction) into new l10n keys across all 14 ARB files, wiring every call site to `l10n.<key>` / `AppLocalizations.of(context)!.<key>`, adding the `AppLocalizations` import where missing, and reusing 3 existing keys (`cancel`, `start`, `finish`) where semantically identical
- Added full `@key` metadata (typed `placeholders` + human-readable `description`) to `app_en.arb` for all 11 surviving ICU-placeholder/plural keys, silencing `flutter gen-l10n` warnings entirely
- Extended `crowdin.yml` with a second `files:` entry mapping `app_en.arb` -> `app_%two_letters_code%.arb`, putting the app on the same Crowdin pipeline as web

## Task Commits

Each task was committed atomically:

1. **Task 1: Dynamic-usage safety scan + authoritative keep/delete inventory** - `bde01f50` (chore)
2. **Task 2: Remove unused keys and equalize the key set across all 14 ARB files** - `4116ecba` (refactor)
3. **Task 3: Extract hard-coded Dart string literals into new l10n keys** - `dc6b98a9` (feat)
4. **Task 4: Add @key metadata for ICU-placeholder keys + extend crowdin.yml** - `f10dc9d8` (docs)

**Plan metadata:** pending final docs commit (STATE.md/ROADMAP.md/REQUIREMENTS.md handled by the orchestrator after this summary)

## Files Created/Modified

- `.planning/quick/260720-s7m-clean-up-arb-translation-files-remove-un/key-inventory.json` - Task 1's authoritative keep/delete/indirect_keep inventory with dynamic-usage findings
- `app/lib/i18n/app_*.arb` (all 14) - dead keys removed, equalized to 267 identical keys, 43 new keys added, `@key` metadata added to `app_en.arb`
- `app/lib/i18n/app_localizations*.dart` (all 14 + base) - regenerated via `flutter gen-l10n`
- `crowdin.yml` - added app ARB file mapping entry
- `app/lib/components/base/wanderer_error.dart` - "Something went wrong" / "Technical Details" -> l10n
- `app/lib/components/base/wanderer_layout.dart` - "Library" nav label -> l10n
- `app/lib/components/base/wanderer_rich_text_editor.dart` - dialog title/label/hint/actions + 7 toolbar tooltips -> l10n
- `app/lib/components/route_planner/elevation_tab.dart` - empty-state hint -> l10n
- `app/lib/components/route_planner/route_anchor_list_tab.dart` - "Reverse direction"/"Delete all" chips -> l10n
- `app/lib/components/route_planner/settings_tab.dart` - "Auto-routing"/hint/"Travel profile" -> l10n
- `app/lib/components/trail/elevation_profile.dart` - "No track data" -> l10n
- `app/lib/components/trail/trail_panel.dart` - "Offline"/"Available offline" badges -> l10n
- `app/lib/components/trail/trail_timeline.dart` - "Start"/"Finish" caps -> reused `start`/`finish` keys
- `app/lib/routes/global_search_screen.dart` - "No results for query" -> l10n (placeholder)
- `app/lib/routes/library_screen.dart` - search hint -> l10n
- `app/lib/routes/list_screen.dart` - search hint + "No lists found" -> l10n
- `app/lib/routes/location_search_screen.dart` - search hint/placeholder + "No results for query" -> l10n
- `app/lib/routes/map_screen.dart` - "Filter" chip -> l10n
- `app/lib/routes/profile_follow_screen.dart` - "No {label} yet." -> l10n (placeholder)
- `app/lib/routes/profile_list_screen.dart` - "No lists yet." -> l10n
- `app/lib/routes/profile_screen.dart` - "No bio yet."/"Feed" -> l10n
- `app/lib/routes/profile_trail_screen.dart` - "No trails yet." -> l10n
- `app/lib/routes/route_planner_screen.dart` - "Search location"/"Undo"/"Redo" -> l10n
- `app/lib/routes/server_selection_screen.dart` - title/hint/empty-state/button -> l10n
- `app/lib/routes/settings_language_screen.dart`, `settings_screen.dart` - hardcoded " & " conjunction -> `language_and_units` placeholder key
- `app/lib/routes/trail_create_screen.dart` - "Edit route" tooltip -> l10n

## Decisions Made

- Applied the CONTEXT.md-locked reuse rule uniformly across Tasks 2 and 3: non-en locale backfill/additions reuse an existing web translation (kebab-case key mapping) when present and non-empty, else fall back to the English string as a Crowdin-ready placeholder
- Reused 3 existing keys (`cancel`, `start`, `finish`) at newly-discovered call sites instead of creating duplicates, per the plan's explicit reuse-if-semantically-identical instruction
- Typed new ICU placeholders per the plan's rule: plural control variables (`n`, `count` inside `{x, plural, ...}`) -> `num`; free-text substitutions (`query`, `label`, `language`, `units`) -> `String`
- Chose double-quote literals for the two new "echo the user's query back" placeholder keys (`no_results_for_query`, `no_servers_match_query`) instead of the original code's single quotes, since a single quote is ICU MessageFormat's escape character and would have silently broken the `{query}` placeholder substitution once the string was ICU-parsed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pre-existing ICU plural-syntax bug in es/eu/pl locale ARB files**
- **Found during:** Task 2 (`flutter gen-l10n` regeneration step)
- **Issue:** `card`, `comment`, `list`, `route`, `trail`, `waypoints` (es, eu) and `trail`, `waypoints` (pl) carried a redundant `one {...}` plural category that coexists with an `=1 {...}` exact match on the same message. Per CLDR, the `one` category for these locales only ever matches `n==1`, and Dart's intl plural codegen always evaluates exact-value matches ahead of keyword categories — so `one` was 100% dead code, which `gen-l10n` correctly flagged as an "ICU Syntax Warning: plural part overridden by a later plural part." This predates this plan (values were unchanged by Task 2's equalization, only reordered) but blocks this plan's own "zero warnings" success criterion since it lives in the exact files this plan modifies.
- **Fix:** Dropped the dead `one` clause in all 14 affected key/locale pairs, keeping the `=1`/`other` (plus `few`/`many` for Polish) structure that already produced the correct output.
- **Files modified:** `app/lib/i18n/app_es.arb`, `app/lib/i18n/app_eu.arb`, `app/lib/i18n/app_pl.arb`
- **Verification:** `flutter gen-l10n` re-run emits zero warnings; `flutter analyze` output unchanged from baseline.
- **Committed in:** `4116ecba` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix)
**Impact on plan:** Necessary to satisfy the plan's own "zero gen-l10n warnings" success criterion; no scope creep beyond the 14 files this plan already touches.

## Issues Encountered

- The original executor session was cut off mid-Task-3 by a connection error. Tasks 1-2 were independently verified complete (commits `bde01f50`, `4116ecba`; `flutter gen-l10n`/`flutter analyze` both green) before this session resumed from Task 3, per the coordinator's explicit continuation instructions. No rework was performed on Tasks 1-2.

## User Setup Required

None - no external service configuration required. (Crowdin project sync itself is an out-of-band step for the project maintainer, not part of this plan's scope.)

## Next Phase Readiness

- All 14 `app_*.arb` files are clean, equalized, and Crowdin-ready; future app string additions should follow the same pattern (add to all 14, reuse-or-fallback for non-en, `@key` metadata for any new ICU placeholders)
- `crowdin.yml` now covers both web and app translation files on the same project - the project maintainer can push `app_en.arb` to Crowdin and start receiving translations for the 43 non-en/de locale strings that currently carry English-fallback placeholder values
- No blockers for future work

---
*Phase: quick-260720-s7m*
*Completed: 2026-07-20*

## Self-Check: PASSED

All claimed files exist on disk (`app/lib/i18n/app_en.arb`, `key-inventory.json`, `crowdin.yml`) and all 4 task commit hashes (`bde01f50`, `4116ecba`, `dc6b98a9`, `f10dc9d8`) are present in git history.
