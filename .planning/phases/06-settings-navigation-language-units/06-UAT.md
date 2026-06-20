---
status: complete
phase: 06-settings-navigation-language-units
source: [06-VERIFICATION.md]
started: 2026-06-20T00:00:00Z
updated: 2026-06-20T01:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Settings sub-navigation
expected: Tap each Settings row (Privacy, Language, Notifications) and confirm navigation to correct sub-screen with titled AppBar and back button
result: pass

### 2. Live locale re-render
expected: Select French in Language & Units screen; UI switches to French and persists after backgrounding and reopening the app; untranslated strings fall back to English
result: pass

### 3. Live unit re-render
expected: Toggle imperial switch; trail card distances, navigation stats, and elevation profile re-render in mi/ft immediately without restart
result: pass

### 4. Regression smoke
expected: /settings/account and /settings/appearance remain reachable and unbroken after Phase 6 router changes
result: pass

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
