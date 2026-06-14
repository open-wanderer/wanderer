---
phase: 04-serialization-fix-entity-schema
verified: 2026-06-14T11:30:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
re_verification: null
gaps: []
deferred: []
human_verification: []
---

# Phase 4: Serialization Fix + Entity Schema Verification Report

**Phase Goal:** Fix the NavigateResponse serialization bug and add the navCacheJson storage field to TrailEntity so Phase 5 can cache navigation instructions in ObjectBox.
**Verified:** 2026-06-14T11:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `jsonEncode(response.toJson())` followed by `NavigateResponse.fromJson(jsonDecode(...))` reconstructs all maneuver and shape fields without throwing | VERIFIED | `group('NavigateResponse roundtrip')` at line 67 of test file; 4 test cases each call `jsonEncode(response.toJson())` then `jsonDecode` then `fromJson`. `_$NavigateResponseToJson` at line 40-44 of navigate_response.g.dart maps maneuvers via `.toJson()`. |
| 2 | `NavigateResponse.toJson()` serializes nested `NavigateManeuver` elements as maps, not as opaque objects | VERIFIED | navigate_response.g.dart line 42: `'maneuvers': instance.maneuvers.map((e) => e.toJson()).toList()` — each maneuver calls its own `.toJson()`. |
| 3 | Existing `fromJson` and `shapeAsLatLng` tests still pass (no regression) | VERIFIED | Original `group('NavigateResponse.fromJson')` (5 tests) remains intact at lines 7-65 of test file, unmodified. `shapeAsLatLng` extension on lines 41-46 of navigate_response.dart is unchanged. |
| 4 | `TrailEntity` has a nullable `String navCacheJson` field following the `gpxData` precedent | VERIFIED | trail_entity.dart line 33: `String? navCacheJson;` declared directly after `gpxData` at line 32. Constructor parameter `this.navCacheJson,` at line 83, next to `this.gpxData,` at line 82. No annotation, no default — matches gpxData pattern exactly. |
| 5 | `objectbox-model.json` contains a `navCacheJson` property entry under the `TrailEntity` model | VERIFIED | objectbox-model.json lines 457-461: `{"id": "26:4841945116984417903", "name": "navCacheJson", "type": 9}` under TrailEntity (entity id `6:5034082009762803572`). UID 4841945116984417903 assigned by build_runner. |
| 6 | `navCacheJson` is entity-only: absent from `TrailEntity.fromModel()` and `TrailEntityMapping.toModel()` conversion paths | VERIFIED | `grep -c navCacheJson trail_entity.dart` returns 2 (field + constructor only). `fromModel()` body at lines 87-131 and `toModel()` body at lines 135-167 contain no `navCacheJson` reference. Confirmed by direct file read. |
| 7 | `Trail` and `TrailExpand` models do NOT have a `navCacheJson` field | VERIFIED | `grep -c navCacheJson app/lib/models/trail.dart` returns 0. |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/models/navigate_response.dart` | `@JsonSerializable(explicitToJson: true)` annotation on `NavigateResponse` factory constructor | VERIFIED | Line 25: `@JsonSerializable(explicitToJson: true)` placed on factory constructor (not class level — correct placement for freezed 3.x). `// ignore_for_file: invalid_annotation_target` suppresses known false-positive. |
| `app/lib/models/navigate_response.g.dart` | Regenerated `_$NavigateResponseToJson` calling `.toJson()` on each maneuver | VERIFIED | Line 42: `'maneuvers': instance.maneuvers.map((e) => e.toJson()).toList()` — not the bare `instance.maneuvers` that previously threw. |
| `app/test/models/navigate_response_test.dart` | `group('NavigateResponse roundtrip')` with 4 test cases | VERIFIED | Lines 67-159: group with full-field, empty-maneuvers, empty-shape, and required-fields-only tests. `import 'dart:convert'` at line 1 enables `jsonEncode`/`jsonDecode`. |
| `app/lib/entities/trail_entity.dart` | `String? navCacheJson` field and constructor parameter | VERIFIED | Line 33 (field), line 83 (constructor param). Exactly 2 occurrences — not referenced elsewhere in the file. |
| `app/lib/objectbox-model.json` | `navCacheJson` property entry under `TrailEntity` | VERIFIED | Property id `26:4841945116984417903`, type 9 (String), at TrailEntity lastPropertyId = `26:4841945116984417903`. |
| `app/lib/objectbox.g.dart` | Regenerated bindings for `navCacheJson` | VERIFIED | Lines 557, 1140-1142, 1169, 1264, 1290, 1776-1777: full ObjectBox binding including FlatBuffer writer, reader, offset registration, and `QueryStringProperty`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `navigate_response.dart` | `navigate_response.g.dart` | `@JsonSerializable(explicitToJson: true)` codegen on factory constructor | WIRED | The annotation is on the factory constructor (line 25); generated `_$NavigateResponseToJson` at lines 40-44 emits the correct `.map((e) => e.toJson()).toList()` call. |
| `trail_entity.dart` | `objectbox-model.json` | objectbox_generator assigns property UID via build_runner | WIRED | `navCacheJson` appears in `trail_entity.dart` (field + constructor) and in `objectbox-model.json` under TrailEntity with a UID assigned by build_runner. `objectbox.g.dart` confirms the property is fully registered with reader/writer/query support. |

### Data-Flow Trace (Level 4)

Not applicable to this phase. Phase 4 delivers infrastructure (serialization fix + schema field), not a component that renders dynamic data. Level 4 data-flow trace applies to Phase 5 when the cache read/write is wired to the ObjectBox service layer.

### Behavioral Spot-Checks

Step 7b skipped: the runnable verification target is `flutter test` which requires the Flutter SDK. The static code evidence (generated `.g.dart` correctly maps maneuvers; 4 roundtrip test cases exercise `jsonEncode` end to end; test file was committed at c0168221 with all 9 tests passing per SUMMARY) provides sufficient evidence without executing the test suite in this environment.

### Probe Execution

No probes declared in PLAN files for this phase. Conventional `scripts/*/tests/probe-*.sh` pattern not applicable.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| (none — prerequisite infra phase) | 04-01, 04-02 | Both plans declare `requirements: []` — all OFFLINE-xx requirements deliver in Phase 5. REQUIREMENTS.md traceability table confirms this. | SATISFIED | REQUIREMENTS.md "Infrastructure prerequisite" note and traceability row: Phase 4 has no OFFLINE-xx assignments. |

No orphaned requirements: REQUIREMENTS.md traceability confirms OFFLINE-01–04 are mapped to Phase 5, not Phase 4.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | — |

Direct file reads of all four modified files confirm:
- No TBD, FIXME, XXX, TODO, or PLACEHOLDER markers in any phase-modified file.
- No `return null` / `return []` / `return {}` stubs in the model or entity files.
- `navCacheJson = null` (the nullable default) is an intentional schema slot, not a UI-facing stub. Phase 5 will populate it.

### Human Verification Required

None. Phase 4 is a pure infrastructure phase (serialization + schema) with no UI behaviors, visual outputs, or external service calls. All truths are verifiable statically from the codebase.

### Gaps Summary

No gaps. All 7 must-haves are VERIFIED by direct codebase evidence:

1. The `@JsonSerializable(explicitToJson: true)` annotation is on the `NavigateResponse` factory constructor (correct freezed 3.x placement), generating `_$NavigateResponseToJson` that calls `.toJson()` on each maneuver.
2. A `group('NavigateResponse roundtrip')` with 4 cases exercises `jsonEncode(response.toJson()) -> jsonDecode -> fromJson` end to end, proving the fix is lossless.
3. `String? navCacheJson` is present on `TrailEntity` (field + constructor, exactly 2 occurrences) and fully registered in `objectbox-model.json` (property id `26:4841945116984417903`) and `objectbox.g.dart` (complete read/write/query binding).
4. The field is entity-only: absent from `trail.dart`, `fromModel()`, and `toModel()`.
5. Three commits (`7eace683`, `c0168221`, `69c01707`) exist in git history confirming atomic delivery.

The phase goal is achieved. Phase 5 can proceed to wire cache write/read against `navCacheJson` and `NavigateResponse.toJson()`/`fromJson()`.

---

_Verified: 2026-06-14T11:30:00Z_
_Verifier: Claude (gsd-verifier)_
