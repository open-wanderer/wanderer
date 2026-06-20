# Phase 7: Privacy - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 7-privacy
**Areas discussed:** Subtitle text, Section headers

---

## Subtitle text

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — show descriptions | Uses 6 existing ARB keys. Matches web client exactly. | ✓ |
| No — title only | Clean and compact. Just Public / Private radio labels. | |

**User's choice:** Yes — show descriptions
**Notes:** All 6 ARB keys already exist in app_en.arb and all 14 locale files — no new strings needed.

---

## Section headers

| Option | Description | Selected |
|--------|-------------|----------|
| Account privacy / Trails / Lists | Uses account_privacy ARB key + pluralized trail and list keys. Matches web layout. | ✓ |
| Account / Trails / Lists | Shorter — drops "privacy" from Account header. | |

**User's choice:** Account privacy / Trails / Lists
**Notes:** Uses ICU plural keys: l10n.trail(n: 2) → "Trails", l10n.list(n: 2) → "Lists".

---

## Claude's Discretion

- Save pattern: `SettingsPrivacy.copyWith()` for partial updates (freezed-generated; standard pattern)
- Default null privacy values: match web client defaults (account: public, trails: private, lists: private)
- Label pairs: public/private for account; public/only_me for trails and lists (matching web)

## Deferred Ideas

None.
