# Phase 4: Serialization Fix + Entity Schema - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix the `NavigateResponse.toJson()` serialization bug that blocks ObjectBox caching, add the `navCacheJson` field to `TrailEntity`, and verify the entity↔model mapping excludes the new field from the `Trail` model. This is a prerequisite infrastructure phase — no user-facing changes, no OFFLINE-xx requirements. Phase 5 builds on this foundation.

</domain>

<decisions>
## Implementation Decisions

### Serialization Fix
- **D-01:** Add `@JsonSerializable(explicitToJson: true)` to `NavigateResponse` (not `NavigateManeuver` — maneuver already has its own `toJson()`, the issue is that `NavigateResponse.toJson()` doesn't call it). Place the annotation above `@freezed` on `NavigateResponse`.
- **D-02:** Run `build_runner` after the fix to regenerate `navigate_response.g.dart`. Commit `objectbox-model.json` immediately after any `build_runner` run to prevent UID conflicts.

### Roundtrip Unit Test
- **D-03:** Add a new `group('NavigateResponse roundtrip')` to the existing `test/models/navigate_response_test.dart`. Do NOT create a new test file.
- **D-04:** Test coverage for the roundtrip group:
  - **Full field roundtrip:** `jsonEncode(response.toJson())` → `jsonDecode(...)` → `NavigateResponse.fromJson(...)` — assert all `NavigateManeuver` fields (instruction, length, beginShapeIndex, bearing, type) and shape entries survive.
  - **Edge cases:** Empty maneuvers list, empty shape, maneuver with only required fields (bearing and type rely on `@Default` values).

### Shape Helper
- **D-05:** Do NOT extract `shapeAsLatLng` from `NavigateResponse`. The extension already works for both the online and cache paths — Phase 5 decodes `navCacheJson` back to `NavigateResponse` and calls `shapeAsLatLng` naturally. No extraction needed.

### TrailEntity Schema
- **D-06:** Add `String? navCacheJson` to `TrailEntity` following the `gpxData: String?` precedent. ObjectBox handles nullable String fields natively; no migration guard needed.
- **D-07:** `navCacheJson` is entity-only. It must NOT be added to `Trail` or `TrailExpand` models. `fromModel()` does not touch it. `toModel()` does not expose it. Phase 5 reads `navCacheJson` directly from `TrailEntity` in the ObjectBox service layer.
- **D-08:** Verify that `TrailEntity.fromModel()` and `TrailEntityMapping.toModel()` remain correct after adding the field — confirm the field is absent from both conversion paths.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Serialization Bug (confirmed)
- `app/lib/models/navigate_response.dart` — `NavigateResponse` and `NavigateManeuver` freezed classes; `@JsonSerializable(explicitToJson: true)` goes on `NavigateResponse`
- `app/lib/models/navigate_response.g.dart` — generated file; line showing the bug: `'maneuvers': instance.maneuvers` (no `.toJson()` call) — will be regenerated after fix

### Entity Schema
- `app/lib/entities/trail_entity.dart` — `TrailEntity` class; add `String? navCacheJson` here, following `gpxData: String?` pattern; also update constructor signature
- `app/lib/models/trail.dart` — `Trail` model; confirm `navCacheJson` is NOT added here

### Existing Tests (extend, don't replace)
- `app/test/models/navigate_response_test.dart` — existing test file with `fromJson` group; add `roundtrip` group here

### Navigation Callers (must not regress)
- `app/lib/routes/navigation_screen.dart` — calls `widget.response.shapeAsLatLng` (lines 130–131); must still work after serialization fix
- `app/lib/provider/navigation_provider.dart` — calls `state.response.shapeAsLatLng` (line 100); must still work

### Requirements Reference
- `.planning/REQUIREMENTS.md` — "Infrastructure prerequisite" note under v1.1 requirements; Phase 4 satisfies this before Phase 5 can begin

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/models/navigate_response.dart` `NavigateResponseX` extension — `shapeAsLatLng` getter stays here; no extraction
- `app/test/models/navigate_response_test.dart` — existing test structure and sample JSON to reuse in roundtrip group

### Established Patterns
- `gpxData: String?` on `TrailEntity` — exact precedent for `navCacheJson: String?` (nullable, entity-only, not in Trail model)
- `@freezed` + `@JsonSerializable(explicitToJson: true)` — standard freezed pattern for models with nested serializable objects; see any freezed model with nested types in the project
- `build_runner` → commit `objectbox-model.json` immediately — established convention (STATE.md)

### Integration Points
- `TrailEntity` constructor: add `this.navCacheJson` as an optional named parameter with default `null`
- ObjectBox `objectbox-model.json` will gain a `navCacheJson` property under `TrailEntity` after `build_runner` — commit this file in the same commit as the entity change

</code_context>

<specifics>
## Specific Ideas

- The serialization bug is fully understood and confirmed from `navigate_response.g.dart`: `_$NavigateResponseToJson` outputs `'maneuvers': instance.maneuvers` instead of `'maneuvers': instance.maneuvers.map((e) => e.toJson()).toList()`. Adding `@JsonSerializable(explicitToJson: true)` to `NavigateResponse` causes codegen to emit the correct call.
- Test sample data: reuse the `sampleJson` fixture already in `navigate_response_test.dart` for the roundtrip test to keep setup minimal.

</specifics>

<deferred>
## Deferred Ideas

- Migration guard for existing ObjectBox records with `navCacheJson = null` — not needed; ObjectBox handles nullable fields transparently on schema evolution
- Exposing `navCacheJson` on the `Trail` model — deferred; Phase 5 reads directly from `TrailEntity` in the service layer
- Extracting `shapeAsLatLng` to a standalone utility function — not needed; extension on `NavigateResponse` works for both online and cache paths

</deferred>

---

*Phase: 04-serialization-fix-entity-schema*
*Context gathered: 2026-06-14*
