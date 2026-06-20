---
phase: 06-settings-navigation-language-units
plan: 04
subsystem: mobile-app
tags: [i18n, localization, arb, gen-l10n, flutter]
requires:
  - app_en.arb (template, 482 content keys)
  - web/src/lib/i18n/locales/*.json (web translation source)
  - localeProvider + supportedLocales wiring (Plan 01)
provides:
  - 12 new locale ARB files (cs, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh)
  - Regenerated AppLocalizations covering all 14 locales
  - Live language switch coverage for all 14 Language enum values
affects:
  - app/lib/i18n/app_localizations.dart (regenerated)
  - Language & Units screen (Plan 02 — picker now switches to real translations)
tech-stack:
  added: []
  patterns:
    - "ARB translation files are flat snake_case key->value with @@locale header and no @key metadata blocks (only the en template carries metadata)"
    - "Port pipeline: iterate the en template key set, kebab-case each key, look up in web JSON, omit empty/missing for English fallback"
    - "ICU plural/select branch text is NOT a placeholder — only {name,...} args and bare {name} refs count for placeholder-safety validation"
key-files:
  created:
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
    - app/lib/i18n/app_localizations_cs.dart
    - app/lib/i18n/app_localizations_es.dart
    - app/lib/i18n/app_localizations_eu.dart
    - app/lib/i18n/app_localizations_fr.dart
    - app/lib/i18n/app_localizations_hu.dart
    - app/lib/i18n/app_localizations_it.dart
    - app/lib/i18n/app_localizations_nl.dart
    - app/lib/i18n/app_localizations_no.dart
    - app/lib/i18n/app_localizations_pl.dart
    - app/lib/i18n/app_localizations_pt.dart
    - app/lib/i18n/app_localizations_ru.dart
    - app/lib/i18n/app_localizations_zh.dart
  modified:
    - app/lib/i18n/app_localizations.dart
decisions:
  - "Empty-string web values are treated as missing and omitted, so the key falls back to English instead of rendering a blank UI string"
  - "Placeholder-safety check ignores ICU plural/select branch literal text and validates only real argument names, so multi-category Slavic plurals (one/few/many) are kept verbatim"
  - "Chinese ported from the single web zh.json (no script subtag) per RESEARCH Open Q3"
metrics:
  duration: ~10 min
  completed: 2026-06-19
requirements: [LANG-01]
---

# Phase 6 Plan 04: Port 12 Locale ARB Files + Regenerate AppLocalizations Summary

Ported the 12 missing locale ARB files (`cs`, `es`, `eu`, `fr`, `hu`, `it`, `nl`, `no`, `pl`, `pt`, `ru`, `zh`) from the web client's existing JSON translations into Flutter ARB format and regenerated `AppLocalizations`, making the live language switch (Plan 01 + Plan 02) real for all 14 locales.

## What Was Built

- **12 new `app_<locale>.arb` files** under `app/lib/i18n/`, each a flat snake_case key->value map with a `"@@locale"` header and no `@key` metadata blocks (translation files mirror `app_de.arb`, not the `app_en.arb` template).
- **Port pipeline** (one-shot Python, not committed): iterates the 482 content keys of the `app_en.arb` template; for each key it kebab-cases the name (`account_delete_confirm` -> `account-delete-confirm`), looks it up in the matching `web/src/lib/i18n/locales/<code>.json`, and copies the translated value. Keys missing from the web JSON or whose web value is an empty string are omitted so gen-l10n auto-fills them from the English template (D-02 fallback).
- **Placeholder-safe ICU handling**: the validator extracts only real ICU argument names (`{name, plural, ...}` args and bare `{name}` refs), explicitly ignoring plural/select branch literal text. This kept all plural keys (`activity`, `card`, `comment`, `route`, `trail`, `waypoints`, etc.) including multi-category Slavic plurals, with zero placeholder mismatches.
- **Regenerated `app_localizations*.dart`**: `flutter gen-l10n` produced 14 per-locale files and an `app_localizations.dart` whose `supportedLocales` now lists all 14 locales. This is what makes Plan 01's `AppLocalizations.supportedLocales` wiring cover every `Language` enum value.

### Per-locale translated key counts (out of 482 template keys)

| Locale | Translated | Falls back to English |
| ------ | ---------- | --------------------- |
| cs | 404 | 78 |
| es | 405 | 77 |
| eu | 405 | 77 |
| fr | 405 | 77 |
| hu | 405 | 77 |
| it | 404 | 78 |
| nl | 406 | 76 |
| no | 446 | 36 |
| pl | 405 | 77 |
| pt | 405 | 77 |
| ru | 405 | 77 |
| zh | 404 | 78 |

Untranslated keys are exactly the count gen-l10n reported and are covered by the English template (intended behavior, not a defect).

## Task Commits

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Port 12 ARB files + regenerate AppLocalizations | 27f944e4 | 12 ARB + 12 new + 1 modified localizations dart |

## Verification

- All 12 ARB files parse as valid JSON, begin with the correct `"@@locale"` header, and contain no `@`-prefixed metadata keys (`python3 json.load` + structural assertion exit 0 for all 12).
- `grep -L '"@[a-z]'` lists `app_fr.arb` and `app_zh.arb` (confirming flat translation files with no metadata).
- `flutter gen-l10n` exits 0. Output contains only ICU **warnings** (Polish `pl` plural strings use a non-canonical `many {...}=1 {...}` form that triggers "plural part overridden" warnings) and the expected "N untranslated message(s)" notices — no errors.
- 14 `app_localizations_<code>.dart` files generated; `grep -c "Locale('" app_localizations.dart` = 14.
- `flutter analyze lib/i18n/app_localizations.dart` -> "No issues found!".
- `app_en.arb` and `app_de.arb` were not modified (not staged, not in commit).

## Deviations from Plan

None — plan executed exactly as written. The plan's `<action>` step 2 said to copy any value the web JSON has; empty-string web values are omitted (treated as "not present") to preserve English fallback rather than render blank strings. This is consistent with the plan's stated fallback intent (D-02) and RESEARCH Assumption A1, so it is recorded as a clarification rather than a deviation.

## Known Stubs

None. Every ported key carries a real translated string; untranslated keys are intentionally absent so gen-l10n supplies the English template value.

## Self-Check: PASSED

- FOUND: app/lib/i18n/app_cs.arb, app_es.arb, app_eu.arb, app_fr.arb, app_hu.arb, app_it.arb, app_nl.arb, app_no.arb, app_pl.arb, app_pt.arb, app_ru.arb, app_zh.arb
- FOUND: app/lib/i18n/app_localizations_{cs,es,eu,fr,hu,it,nl,no,pl,pt,ru,zh}.dart (14 per-locale files total incl. en/de)
- FOUND: app/lib/i18n/app_localizations.dart with 14 supportedLocales
- FOUND: commit 27f944e4
