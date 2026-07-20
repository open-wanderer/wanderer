# Quick Task 260720-s7m: Clean up ARB translation files - Context

**Gathered:** 2026-07-20
**Status:** Ready for planning

<domain>
## Task Boundary

The app's `app/lib/i18n/*.arb` files were bootstrapped by copying all strings from web's
`web/src/lib/i18n/locales/*.json`. Clean this up:

1. Remove ARB keys not referenced anywhere in `app/lib/**/*.dart`
2. Add hard-coded strings found in app Dart code to the ARB files
3. Add `@key` metadata blocks for ICU-placeholder/plural keys to silence `gen-l10n` warnings
4. Extend the Crowdin setup (`crowdin.yml`) to cover the app's ARB files alongside web's JSON

Translating existing/new strings into locales other than en/de is out of scope, EXCEPT where
a string already has a translation on the web side (web's `*.json` already has it) — that
existing web translation may be reused for the matching app key.

</domain>

<decisions>
## Implementation Decisions

### Locale scope
- All 14 ARB files (en, de, cs, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh) end up with the
  **same set of keys** after this cleanup.
- Unused-key removal applies to all 14 files.
- New keys (from hard-coded strings) are added to all 14 files:
  - en.arb gets the real English string.
  - de.arb gets the real German string if a matching web translation already exists,
    otherwise the English string as a placeholder fallback.
  - The other 12 locales get the English string as a placeholder fallback (per project
    scope: translating into other languages is out of scope for now), UNLESS the web
    locale file already has a matching translated string for that key — in that case, reuse
    the existing web translation.

### Unused-key removal
- Approach: remove all keys with zero textual references in `app/lib/**/*.dart`, but first
  do a safety pass grepping for indirect/dynamic key usage (e.g. keys built from a category,
  activity-type, or amenity-type name, switch-based lookups, or reflection-like access)
  before deleting, to avoid false-positive removals.
- Baseline scan (en.arb, direct references only): ~156 of 525 keys referenced, ~362
  unreferenced. This number must be re-verified during planning/execution with the dynamic
  usage check applied, not deleted blindly off the raw diff.

### New string translations
- App-only hard-coded strings with no existing web translation: add to de.arb (and other
  locales) using the English text as a placeholder value. Real translation happens later via
  Crowdin.

### ARB metadata
- Add full `@key` metadata for every key using ICU placeholders/plurals (~32 keys in en.arb,
  e.g. `n_days_ago`, `file_too_big`, `error_setting_up_integration`): include both
  `placeholders` (name + type) and a short human-readable `description`. The description
  exists specifically to give Crowdin translators context on what each placeholder means.
- Metadata blocks only need to live in the template file (`app_en.arb`); other locale ARB
  files don't require `@key` blocks (standard `gen-l10n` behavior — only the template arb
  needs metadata).

### Crowdin strategy
- Extend the existing `crowdin.yml` with a second file entry for the app's ARB files,
  mirroring the pattern already used for web:
  ```yaml
  files:
    - source: /web/src/lib/i18n/locales/en.json
      translation: /web/src/lib/i18n/locales/%two_letters_code%.%file_extension%
    - source: /app/lib/i18n/app_en.arb
      translation: /app/lib/i18n/app_%two_letters_code%.arb
  ```
  Same Crowdin project, same language/locale-code mapping as web — no new project needed.

</decisions>

<specifics>
## Specific Ideas

- `l10n.yaml` config: `arb-dir: lib/i18n`, `template-arb-file: app_en.arb`,
  `output-localization-file: app_localizations.dart`.
- Localization is accessed in Dart code almost exclusively via a `final l10n =
  AppLocalizations.of(context)!;` (or occasionally `ctx`) local variable, then
  `l10n.some_key`. A few call sites use `AppLocalizations.of(context)!.some_key` directly
  without an intermediate variable. Both patterns must be covered when scanning for usage.
- Existing precedent for partial `@key` metadata already present in en.arb: `@in_distance`,
  `@resume_navigation_prompt`, `@subcategories`,
  `@settings_categories_confirm_disable_body` — follow their existing shape/style.

</specifics>

<canonical_refs>
## Canonical References

- `crowdin.yml` (repo root) — existing web-only Crowdin file mapping, to be extended.
- `app/l10n.yaml` — Flutter gen-l10n config.
- `web/src/lib/i18n/locales/*.json` — source of truth for any already-existing translations
  to reuse for matching app keys.

</canonical_refs>
