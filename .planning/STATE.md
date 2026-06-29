---
gsd_state_version: 1.0
milestone: v1.3
milestone_name: Category Redesign
status: planning
stopped_at: Phase 10 context gathered
last_updated: "2026-06-29T12:25:47.194Z"
last_activity: 2026-06-29 — v1.3 roadmap created (Phases 10-12), all 19 requirements mapped
progress:
  total_phases: 3
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-06-29)

**Core value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.
**Current focus:** Phase 10 — Category & Subcategory Data Layer

## Current Position

Phase: 10 of 12 (Category & Subcategory Data Layer)
Plan: — (not yet planned)
Status: Ready to plan
Last activity: 2026-06-29 — v1.3 roadmap created (Phases 10-12), all 19 requirements mapped

Progress: [░░░░░░░░░░] 0%

## v1.3 Phases

- [ ] **Phase 10: Category & Subcategory Data Layer** — CAT-01..05 (models, ObjectBox entity, provider, remove `Settings.category`)
- [ ] **Phase 11: Trail Filter Subcategory Support** — FILTER-01..05 (TrailFilter model, TrailFilterScreen chips, quick filter bar)
- [ ] **Phase 12: Settings Categories Screen** — SETCAT-01..09 (preference models + providers, SettingsCategoriesScreen, router wiring)

Execution order: 10 → 11 → 12 (11 and 12 both depend on 10; independent of each other).

## Performance Metrics

**Velocity (v1.0–v1.2):**

- Total plans completed: 21
- Average duration: — min
- Total execution time: — hours

**By Phase (recent):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 06 | 4 | - | - |
| 07 | 1 | - | - |
| 08 | 3 | - | - |
| 09 | 1 | - | - |

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [v1.3 roadmap] Phase 10 is the data-layer foundation (Category + Subcategory models, ObjectBox entity, provider, `Settings.category` removal); both Phase 11 and Phase 12 depend on its locale-resolved names and Subcategory provider
- [v1.3 roadmap] CAT-05 (remove `Settings.category`) lives in Phase 10 with the rest of the model cleanup, not in the Settings screen phase — it's a data-model change, replaced by priority-based category preferences
- [v1.3 roadmap] Phases 11 (filters) and 12 (Settings Categories) are independent of each other; coarse granularity keeps each requirement cluster as one cohesive phase rather than splitting model/provider/UI into separate phases
- [v1.2] freezed 3.x: @JsonSerializable(explicitToJson: true) must be on the factory constructor, not above @freezed — critical for the new Subcategory/preference freezed models in v1.3
- [v1.2] `State.mounted` in ConsumerState, `context.mounted` in ConsumerWidget helpers — reuse for SettingsCategoriesScreen async guards
- [v1.2] `Settings` freezed model + `settingsProvider.saveToServer()` shared across settings screens — but v1.3 category preferences use dedicated `/user-category-preference` and `/user-subcategory-preference` endpoints, not Settings

### Pending Todos

- Plan Phase 10 first — its Subcategory provider and locale-aware Category names are prerequisites for Phases 11 and 12
- Reference web PR #1059 for the category model shape (translations/icon/short_name), subcategory fields, and the `/user-category-preference` + `/user-subcategory-preference` + `/reorder` API contracts
- Verify `Settings.category` removal (CAT-05) covers all call sites before declaring Phase 10 done

### Blockers/Concerns

- None for v1.3 — work mirrors a shipped web PR (#1059); API endpoints exist server-side and the Flutter settings/filter infrastructure from v1.0–v1.2 is in place.

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| Trail form | Category/subcategory picker in create/edit form (TRAILFORM-01/02) | v1.4 | v1.3 requirements |
| Bulk edit | Bulk-edit modal for category/subcategory/difficulty | Out of scope (web-only) | v1.3 requirements |
| Filters | Subcategory reordering within a category | Future milestone | v1.3 requirements |
| Audio | TTS maneuver announcements (AUDIO-01, AUDIO-02) | v2 | Init |
| Routing | Navigate from user position to trailhead; off-trail re-routing | Out of scope | Init |
| Account | API token management (ACCT-F01) | Future | v1.2 requirements |
| Settings | Favourite sport picker, Export, Integrations, Maintenance, Map settings | Out of scope | v1.2 requirements |

## Session Continuity

Last session: 2026-06-29T12:25:47.180Z
Stopped at: Phase 10 context gathered
Resume file: .planning/phases/10-category-subcategory-data-layer/10-CONTEXT.md

## Operator Next Steps

- Plan Phase 10 with `/gsd-plan-phase 10`
