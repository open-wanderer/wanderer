---
phase: 01-backend-api
reviewed: 2026-06-12T00:00:00Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - web/src/routes/api/v1/valhalla/navigate/+server.ts
  - web/src/routes/api/v1/valhalla/navigate/server.test.ts
  - web/src/lib/models/api/valhalla_navigate_schema.ts
  - web/src/lib/models/valhalla.ts
findings:
  critical: 2
  warning: 3
  info: 1
  total: 6
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-06-12T00:00:00Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Reviewed the Valhalla navigate API endpoint, its Zod schema, the supporting response model file, and the test suite. The endpoint is architecturally sound — auth guard, Zod validation, and upstream error handling all present. Two blockers were found: a missing `Content-Type` header on the outbound Valhalla request (will cause Valhalla to reject the body in most configurations), and an unguarded access to `.trip` on the upstream response that produces an uncontrolled 500 when Valhalla returns a 200 without a `trip` field. Three warnings cover: internal error leakage in the catch-all branch of `handleError`, a return-type lie on `getValhallaBaseUrl` that makes the null guard brittle, and the undocumented encode/decode asymmetry in `polyline_util` that has already caused confusion (see test comments). One info item covers an unused MapLibre import in the model file.

---

## Critical Issues

### CR-01: Missing `Content-Type: application/json` header on Valhalla fetch

**File:** `web/src/routes/api/v1/valhalla/navigate/+server.ts:97`

**Issue:** The `event.fetch` call sends a JSON-serialised body to Valhalla but does not set `Content-Type: application/json`. Valhalla's HTTP API requires this header; without it, most server configurations will return a 400 or attempt to parse the body as form data, causing the upstream call to fail for every request even though the route logic is correct. The test suite stubs `event.fetch` with a hardcoded mock, so this bug is invisible to the tests.

**Fix:**
```typescript
const res = await event.fetch(baseUrl + "/route", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(valhallaReq),
});
```

---

### CR-02: Unguarded `.trip` access causes uncontrolled 500 on unexpected Valhalla 200

**File:** `web/src/routes/api/v1/valhalla/navigate/+server.ts:107`

**Issue:** `const trip = (await res.json()).trip;` is executed immediately after the `res.ok` check. If Valhalla returns HTTP 200 with a body that lacks a `trip` field (e.g., a Valhalla error envelope with `{"error":"...","error_code":...}` that still carries a 200 status, or a network proxy that returns a 200 HTML error page), `trip` is `undefined` and `trip.legs` on line 112 throws `TypeError: Cannot read properties of undefined`. This TypeError propagates to the catch block, which hits `handleError`'s final `else` branch and returns a 500 whose `message` field is the raw exception object — violating the "never a 500" contract the tests assert elsewhere (Tests 5 and 6).

**Fix:**
```typescript
const data = await res.json();
const trip = data?.trip;
if (!trip?.legs) {
  return json({ message: "valhalla_error", detail: "unexpected response shape" }, { status: 502 });
}
```

---

## Warnings

### WR-01: `handleError` catch-all leaks raw exception objects in 500 responses

**File:** `web/src/lib/util/api_util.ts:167`

**Issue:** The final `else` branch of `handleError` returns `json({ message: e }, { status: 500 })`. If `e` is an `Error` object, this serialises to `{ "message": {} }` (opaque). If `e` is a string or has a `.toString()` that includes a stack trace or internal path, that information reaches the client. The navigate endpoint calls `handleError(e)` in its outer catch (line 131 of `+server.ts`), so any unhandled exception (e.g., the CR-02 TypeError) will hit this path.

**Fix:**
```typescript
} else {
  const message = e instanceof Error ? e.message : String(e);
  return json({ message: "internal_error", detail: message }, { status: 500 });
}
```

This keeps a consistent `message` key string and limits what is exposed. For production, `detail` should be omitted or replaced with a generic phrase.

---

### WR-02: `getValhallaBaseUrl` return type hides empty-string case; null guard is fragile

**File:** `web/src/routes/api/v1/valhalla/navigate/+server.ts:84`

**Issue:** `getValhallaBaseUrl()` is typed as returning `string` (never `null` or `undefined`). `resolveBaseUrl` → `normalizeBaseUrl` returns `""` (empty string) when the env var is unset. The check `if (!baseUrl)` at line 85 correctly catches an empty string at runtime, but TypeScript does not enforce it — any future caller that trusts the non-nullable return type will skip the guard and construct a broken URL like `"/route"` without a host, producing a fetch error rather than a clean 400. The guard must stay in sync with the implementation of `resolveBaseUrl` manually.

**Fix:** Change the return type to `string | null` (or return `null` from `getValhallaBaseUrl` when the resolved URL is empty) so TypeScript enforces the null check at every call site:
```typescript
// web/src/lib/server/valhalla.ts
export function getValhallaBaseUrl(): string | null {
  const url = resolveBaseUrl("VALHALLA_URL");
  return url || null;
}
```

---

### WR-03: `encodePolyline` and `decodePolyline` use opposite coordinate conventions — undocumented asymmetry

**File:** `web/src/lib/util/polyline_util.ts:26,71`

**Issue:** `decodePolyline` internally accumulates `lat` first and `lng` second (standard polyline encoding), but pushes each coordinate as `[lng / factor, lat / factor]` (GeoJSON order, lon first). `encodePolyline` treats the first element of each input pair as the first encoded value — which, given the decode convention, means callers must pass `[lat, lng]` to `encodePolyline` for encode/decode to be invertible. This mismatch is not documented on either function. The test suite (lines 51–59) has an extended comment explaining the asymmetry because the author already encountered confusion from it. The navigate endpoint works correctly (line 115: destructures decode output as `[lng, lat]`), but the asymmetry is a reliability hazard for future code that uses either utility without reading the decode source.

**Fix:** Either:
1. Document the conventions with JSDoc on both functions (minimum fix), or
2. Make `decodePolyline` return `[lat, lng]` pairs and update all callers (`polylineToGeoJSON` already calls `flipped()` to compensate, which would simplify).

---

## Info

### IN-01: `valhalla.ts` imports `maplibre-gl` but is a shared model file also pulled in by server-side code paths

**File:** `web/src/lib/models/valhalla.ts:1`

**Issue:** Line 1 imports `* as M from "maplibre-gl"`, used only for the `M.Marker` type in the `ValhallaAnchor` interface (line 143). `maplibre-gl` is a browser-only bundle. This file is currently not imported by the navigate server route, so it does not cause an immediate SSR failure. However, placing a browser-only import in `web/src/lib/models/` (a shared layer) means any future server-side consumer of these types will pull in MapLibre's browser bundle and likely break SSR or cause build warnings.

**Fix:** Replace the MapLibre import with a type-only inline definition:
```typescript
// Remove: import * as M from "maplibre-gl";

interface ValhallaAnchor {
  id: string;
  lat: number;
  lon: number;
  marker?: { remove(): void };  // replace M.Marker with minimal structural type
}
```
Or move `ValhallaAnchor` to a browser-only file under `web/src/lib/components/` or `web/src/lib/util/valhalla_anchor_util.ts` where a browser import is expected.

---

_Reviewed: 2026-06-12T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
