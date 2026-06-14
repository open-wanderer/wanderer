---
phase: 01-backend-api
fixed_at: 2026-06-12T00:00:00Z
review_path: .planning/phases/01-backend-api/01-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-06-12T00:00:00Z
**Source review:** .planning/phases/01-backend-api/01-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (CR-01, CR-02, WR-01, WR-02, WR-03; IN-01 excluded by fix_scope: critical_warning)
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: Missing `Content-Type: application/json` header on Valhalla fetch

**Files modified:** `web/src/routes/api/v1/valhalla/navigate/+server.ts`
**Commit:** 9b4729a2
**Applied fix:** Added `headers: { "Content-Type": "application/json" }` to the `event.fetch` call sending the JSON body to Valhalla at line 97–101.

---

### CR-02: Unguarded `.trip` access causes uncontrolled 500 on unexpected Valhalla 200

**Files modified:** `web/src/routes/api/v1/valhalla/navigate/+server.ts`
**Commit:** 5054a972
**Applied fix:** Changed `const trip = (await res.json()).trip` to use optional chaining (`data?.trip`) and added a guard `if (!trip?.legs)` that returns a 502 with `{ message: "valhalla_error", detail: "unexpected response shape" }` before attempting to iterate `trip.legs`.

---

### WR-01: `handleError` catch-all leaks raw exception objects in 500 responses

**Files modified:** `web/src/lib/util/api_util.ts`
**Commit:** 2772c31b
**Applied fix:** Replaced `return json({ message: e }, { status: 500 })` with a safe extraction that returns `{ message: "internal_error", detail: message }` where `message` is `e.message` for Error instances or `String(e)` otherwise, preventing opaque or stack-trace-containing values from reaching clients.

---

### WR-02: `getValhallaBaseUrl` return type hides empty-string case; null guard is fragile

**Files modified:** `web/src/lib/server/valhalla.ts`
**Commit:** d8dbf1ee
**Applied fix:** Changed the return type from `string` to `string | null` and updated the function body to return `url || null` so TypeScript enforces the null check at every call site. The existing `if (!baseUrl)` guard in `+server.ts` already handles null correctly.

---

### WR-03: `encodePolyline` and `decodePolyline` use opposite coordinate conventions — undocumented asymmetry

**Files modified:** `web/src/lib/util/polyline_util.ts`
**Commit:** 63dac2f1
**Applied fix:** Added JSDoc comments to both `decodePolyline` and `encodePolyline` explicitly documenting the `[lng, lat]` output convention of `decodePolyline` (GeoJSON order, longitude first) and the `[lat, lng]` input requirement of `encodePolyline` for encode/decode invertibility. This is the minimum-impact fix that resolves the hazard without requiring callers to change.

---

_Fixed: 2026-06-12T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
