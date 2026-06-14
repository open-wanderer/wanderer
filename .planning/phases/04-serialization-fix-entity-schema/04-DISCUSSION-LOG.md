# Phase 4: Serialization Fix + Entity Schema - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-14
**Phase:** 04-serialization-fix-entity-schema
**Areas discussed:** Shape helper extraction, Roundtrip test scope, Phase 4 scope boundary

---

## Shape Helper Extraction

| Option | Description | Selected |
|--------|-------------|----------|
| Extract to a standalone function | Move List<List<double>> → List<LatLng> conversion to a top-level function in gpx_util.dart. Extension delegates to it. More testable. | |
| Keep extension, skip extraction | shapeAsLatLng on NavigateResponse already works for both online and cache paths. Phase 5 decodes navCacheJson → NavigateResponse and calls it naturally. | ✓ |
| New nav_util.dart helper | Create a dedicated nav_util.dart with shape conversion and other navigation utilities Phase 5 might need. | |

**User's choice:** Keep extension, skip extraction
**Notes:** Phase 5 will decode `navCacheJson` back to `NavigateResponse` and call `shapeAsLatLng` naturally — the extension already covers the cache path without changes.

---

## Roundtrip Test Scope

### What should the roundtrip cover beyond 'does not throw'?

| Option | Description | Selected |
|--------|-------------|----------|
| Full field roundtrip | Assert all NavigateManeuver fields (instruction, length, beginShapeIndex, bearing, type) and shape entries survive encode/decode. | ✓ |
| Throw-or-not only | Minimal: just verify jsonEncode doesn't throw and fromJson parses back. | |

**User's choice:** Full field roundtrip

### Should the roundtrip test also cover edge cases?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — add edge cases | Empty maneuvers list, empty shape, maneuver with only required fields (@Default bearing and type). | ✓ |
| No — happy path only | One roundtrip test with the same sample data as existing tests. | |

**User's choice:** Yes — add edge cases
**Notes:** Add to existing `test/models/navigate_response_test.dart` as a new `group('NavigateResponse roundtrip')`. Do not create a new test file.

---

## Phase 4 Scope Boundary

### Is there anything else Phase 4 should handle?

| Option | Description | Selected |
|--------|-------------|----------|
| No — the three tasks are sufficient | Serialization fix + test, navCacheJson on TrailEntity, build_runner with objectbox-model.json committed. | |
| Yes — also add navCacheJson migration guard | Handle existing TrailEntity records with navCacheJson = null. | |
| Yes — also verify TrailEntity ↔ Trail model mapping | Confirm fromModel() and toModel() are correct and navCacheJson is excluded from model conversion. | ✓ |

**User's choice:** Verify TrailEntity ↔ Trail model mapping

### What should the mapping verification confirm?

| Option | Description | Selected |
|--------|-------------|----------|
| Entity-only — not in Trail model | navCacheJson NOT added to Trail or TrailExpand. fromModel() doesn't touch it. toModel() doesn't expose it. Phase 5 reads directly from entity. | ✓ |
| Add to Trail model as optional | Expose navCacheJson on Trail as String? for navigation provider convenience. | |

**User's choice:** Entity-only — not in Trail model
**Notes:** `navCacheJson` stays in the entity layer only. Phase 5 service code reads it directly from ObjectBox without going through the Trail model.

---

## Claude's Discretion

None — all decisions were explicit.

## Deferred Ideas

- Migration guard for existing ObjectBox records — not needed; ObjectBox handles nullable fields transparently
- `shapeAsLatLng` extraction to a standalone utility — confirmed not needed for Phase 5
- Exposing `navCacheJson` on the `Trail` model — deferred; service layer access is sufficient
