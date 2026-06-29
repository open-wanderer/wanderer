# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

---

## Milestone: v1.2 — Settings Screens

**Shipped:** 2026-06-29
**Phases:** 4 (Phases 6-9) | **Plans:** 9 | **Timeline:** 2026-06-19 → 2026-06-21 (3 days active)
**Dart files touched:** ~36 | **Phase commits:** ~60

### What Was Built
- **Phase 6:** Five-row settings list wired to sub-routes; 14-locale language picker + metric/imperial unit toggle that live-switches app-wide via `localeProvider` / `unitProvider`; ~50 `format_util` call sites ported to read unitProvider
- **Phase 7:** `SettingsPrivacyScreen` — three `RadioGroup<String>` sections (account/trails/lists visibility) with auto-save via `settingsProvider`
- **Phase 8:** Full `SettingsAccountScreen` — avatar upload (image_picker multipart POST), change-aware bio editor, email/password bottom-sheet forms with proper credential handling, AlertDialog-gated account deletion
- **Phase 9:** `SettingsNotificationsScreen` — 9 notification types × web/email SwitchListTile toggles, map-copy pattern auto-save, widget test, new `l10n.web` ARB key

### What Worked
- **Reusing `settingsProvider`**: All four screens read/write from the same `Settings` freezed model via `settingsProvider.saveToServer()` — zero new persistence infrastructure needed
- **Wave parallelism in Phase 6**: Plans 06-02, 06-03, and 06-04 ran concurrently with no file overlap — pure parallel execution
- **Auto-save pattern (D-09 map-copy)**: Copying the notifications map, mutating the copy, and saving — same pattern reused across privacy and notifications screens cleanly
- **Stub screens from Phase 6**: Stubs for Privacy/Notifications created upfront; Phases 7 and 9 were pure fills with no scaffolding work
- **Widget tests as acceptance**: Tall-viewport widget tests (1080×4000) for lazy ListViews caught real rendering issues early

### What Was Inefficient
- **`--force` flag missing from CLI wrapper**: `gsd milestone complete --force` silently failed because `gsd-tools.cjs` didn't forward the flag to `cmdMilestoneComplete`. Required a one-line patch at close time.
- **Phase 7 one-liner extraction**: The 07-01-SUMMARY.md one_liner field captured a rule annotation rather than a human description — MILESTONES.md has a low-quality entry for that phase
- **Human-needed verification items**: Phases 6-8 all have `human_needed` verification entries that were acknowledged-and-deferred rather than tested on device. Device testing should be scheduled as a quick task before close, not discovered at close

### Patterns Established
- **`State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers** — `mounted` is a State property only; ConsumerWidget doesn't inherit State. Verify before every async BuildContext guard.
- **`@JsonSerializable(explicitToJson: true)` on freezed factory constructor** — class-level placement breaks codegen in freezed 3.x; always on the factory.
- **`Colors.red.shade400` for destructive foreground text** — `colorScheme.error` is a background token (#FEF2F2), not a foreground color
- **Hardcoded native-name map for language labels** — the only approved hardcoded-string exception; localized names require the locale to be active, which creates a bootstrap problem
- **`findWidgets` not `findsOneWidget` when hintText appears in header + field** — duplicate widget keys in lazy lists require `findsWidgets`

### Key Lessons
1. **Device testing is a milestone blocker, not a nice-to-have**: Acknowledging `human_needed` verification items at close is a smell — these should be scheduled as a quick task (`/gsd-quick "device test phases 6-8"`) before milestone close
2. **Stub screens reduce planning cost**: Creating stubs in the foundational phase (Phase 6) meant downstream phases (7, 9) had zero route/scaffold work — just filling the screen body
3. **Wave parallelism requires explicit no-overlap verification in the plan**: Phase 6 Wave 2 succeeded because plans were written with explicit non-overlapping file lists; this needs to be a planning checklist item
4. **CLI `--force` flags need end-to-end testing**: The missing `--force` passthrough is a category of bug that only surfaces at milestone close — worth a smoke test after any gsd-tools update

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 MVP | 3 | 6 | Established navigation screen pattern (freezed models, Riverpod notifiers, flutter_map) |
| v1.1 Offline | 2 | 6 | Added ObjectBox caching; established DioException-only offline gate pattern |
| v1.2 Settings | 4 | 9 | Shared `settingsProvider` pattern; live locale/unit switching; wave parallelism |

### Cumulative Quality

| Milestone | Widget Tests Added | Notable |
|-----------|--------------------|---------|
| v1.0 | ~6 (navigation + stats) | TDD approach for navigation notifier |
| v1.1 | ~4 (serialization roundtrip, offline fallback) | ObjectBox integration tests via unit tests |
| v1.2 | ~5 (one per settings screen) | Tall-viewport pattern for lazy ListViews |

### Top Lessons (Verified Across Milestones)

1. **Stub screens in foundational phases pay forward** — confirmed in v1.2 (Phase 6 stubs → fast Phases 7+9)
2. **Wave parallelism requires explicit file-overlap analysis at plan time** — confirmed valuable in v1.2 Phase 6
3. **Human/device testing needs a dedicated quick task before milestone close** — first surfaced v1.2; carry forward
