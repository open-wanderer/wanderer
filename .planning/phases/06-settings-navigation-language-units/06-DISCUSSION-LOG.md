# Phase 6: Settings Navigation + Language & Units - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-19
**Phase:** 6-Settings Navigation + Language & Units
**Areas discussed:** Language behavior, Unit display wiring, Settings screen layout, Language picker UI

---

## Language behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Server only | Save to server/ObjectBox; Flutter UI stays en/de; web client uses the preference | |
| Live locale switch | Save to server AND switch Flutter app's displayed UI language via MaterialApp locale: param | ✓ |
| Local-only override | Override app locale in-memory for en/de only; fall back to en for unsupported | |

**User's choice:** Live locale switch

---

| Option | Description | Selected |
|--------|-------------|----------|
| Add all 14 ARB files | Create the missing 12 ARB files with full translations ported from web i18n | ✓ |
| Add ARB stubs, fall back to en | Add 12 files as English copies; real translations later | |
| Wire en/de only for now | Only en/de wired; other locales stored server-side only | |

**User's choice:** Add all 14 ARB files (port from web/src/lib/i18n/locales/*.json)

---

| Option | Description | Selected |
|--------|-------------|----------|
| settingsProvider → locale provider | Derive localeProvider from settingsProvider; MaterialApp reads locale: ref.watch(localeProvider) | ✓ |
| localSettingsProvider + ObjectBox | Store locale in LocalSettingsEntity alongside themeMode | |

**User's choice:** settingsProvider → locale provider (follows themeModeProvider pattern)

---

## Unit display wiring

| Option | Description | Selected |
|--------|-------------|----------|
| Wire it now | Derive unitProvider from settingsProvider; update all format_util call sites | ✓ |
| Save only — wire later | Store preference; leave format_util hardcoded to 'metric' for now | |

**User's choice:** Wire it now

---

| Option | Description | Selected |
|--------|-------------|----------|
| Navigation stats sheet | formatDistance/Speed/Elevation in DraggableScrollableSheet | ✓ |
| Trail detail screens | Distance, elevation, duration on trail_detail_screen | ✓ |
| All format_util call sites | Replace every hardcoded 'metric' across the codebase | ✓ |

**User's choice:** All three — replace all hardcoded 'metric' strings app-wide

---

## Settings screen layout

| Option | Description | Selected |
|--------|-------------|----------|
| Account, Privacy, Language, Notifications, Appearance | Groups data/content settings before visual preference | ✓ |
| Account, Appearance, Privacy, Language, Notifications | Keeps current Account+Appearance pairing | |
| Check web client order | Match web settings list exactly | |

**User's choice:** Account → Privacy → Language → Notifications → Appearance → [Divider] → [Logout]

---

| Option | Description | Selected |
|--------|-------------|----------|
| No sections — flat list | Keep existing flat ListView + Divider before logout | ✓ |
| Section headers | Add headers like 'Account', 'Preferences', 'App' | |
| Two sections with a single Divider | One divider between Notifications/Appearance | |

**User's choice:** Flat list (no section headers)

---

| Option | Description | Selected |
|--------|-------------|----------|
| FontAwesome: lock, globe, bell | faLock (Privacy), faGlobe (Language), faBell (Notifications) | ✓ |
| Material Icons: lock, language, notifications | Mix of icon sets | |
| You decide | Any FontAwesome icons | |

**User's choice:** FontAwesome icons — faLock, faGlobe, faBell

---

## Language picker UI

| Option | Description | Selected |
|--------|-------------|----------|
| Scrollable RadioListTile list | Same pattern as SettingsAppearanceScreen; 14 RadioListTile items | ✓ |
| Dropdown / DropdownButton | Single-line selector; breaks RadioGroup pattern | |
| Search-filtered list | TextField above list; overkill for 14 items | |

**User's choice:** Scrollable RadioListTile list (matches Appearance pattern)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Native name only | Deutsch, Español, Français, 中文, etc. | ✓ |
| English name only | German, Spanish, French, Chinese | |
| Native + English | Deutsch (German), Español (Spanish) | |

**User's choice:** Native name only

---

| Option | Description | Selected |
|--------|-------------|----------|
| SwitchListTile below language list on same screen | Section label "Units" separates groups; single save | ✓ |
| RadioListTile for Metric / Imperial | Matches RadioGroup pattern but more vertical space | |
| Separate Units sub-route | Contradicts phase name and scope | |

**User's choice:** SwitchListTile on same screen as language picker

---

## Claude's Discretion

None — all areas had clear user selections.

## Deferred Ideas

None — discussion stayed within Phase 6 scope.
