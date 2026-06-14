# Phase 1: Backend API - Pattern Map

**Mapped:** 2026-06-12
**Files analyzed:** 3 (1 new endpoint, 1 new schema, 1 model extension)
**Analogs found:** 3 / 3

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `web/src/routes/api/v1/valhalla/navigate/+server.ts` (NEW) | route (SvelteKit endpoint) | request-response / transform | `web/src/routes/api/v1/valhalla/route/+server.ts` (file structure) + `web/src/routes/api/v1/search/actor/+server.ts` (auth gate) | exact (role) + role-match (auth) |
| `web/src/lib/models/api/valhalla_navigate_schema.ts` (NEW) | model (Zod schema) | transform / validation | `web/src/lib/models/api/trail_schema.ts` | exact |
| `web/src/lib/models/valhalla.ts` (MODIFY) | model (types) | transform | self (extend existing `Leg` type) | exact |

## Pattern Assignments

### `web/src/routes/api/v1/valhalla/navigate/+server.ts` (route, request-response/transform)

**Primary analog:** `web/src/routes/api/v1/valhalla/route/+server.ts` (file structure, Valhalla call, 502 wrap)
**Auth analog:** `web/src/routes/api/v1/search/actor/+server.ts` (lines 97-100)
**Error analog:** `web/src/lib/util/api_util.ts` `handleError` (lines 158-168)

**Imports pattern** (compose from route/+server.ts:1-3 + search/actor auth + api_util):
```typescript
import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { handleError } from '$lib/util/api_util';
import { decodePolyline } from '$lib/util/polyline_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
```
Note: the analog `route/+server.ts` imports `proxyJsonResponse` from `$lib/server/http` — do NOT import it here (this endpoint transforms, per D-04/D-05).

**Swagger JSDoc block** (copy structure from `route/+server.ts` lines 9-34) — every endpoint in this codebase has an `@swagger` block with `summary`, `description`, `tags: [Valhalla]`, `requestBody`, and `responses` (200/400/500). Add 401 and 502 entries.

**Auth gate** (copy from `search/actor/+server.ts` lines 97-100) — this is the established gate, NOT present in the `route` analog:
```typescript
export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return error(401, "Unauthorized");
    }
    // ...
}
```

**Valhalla base-url guard + call** (copy from `route/+server.ts` lines 36-50):
```typescript
const baseUrl = getValhallaBaseUrl();
if (!baseUrl) {
    return json({ message: "VALHALLA_URL not set" }, { status: 400 });
}
const response = await event.fetch(baseUrl + '/route', {
    method: "POST",
    body: JSON.stringify(valhallaReq),
});
```
The analog wraps the fetch in try/catch and returns 502 on throw (lines 42-50). Reuse that 502 wrap, AND additionally check `response.ok` (the analog does NOT — Pitfall 5: `event.fetch` does not throw on HTTP error status).

**Core transform pattern** (NEW logic — no analog; build per RESEARCH Pattern 1 & 2):
- Build upstream body explicitly: `{ locations: waypoints, costing, directions_type: 'instructions' }`. Do NOT spread raw client body (injection mitigation).
- `directions_type: 'instructions'` — required for instruction text (overrides CONTEXT shorthand "maneuvers", see RESEARCH Q2 / Pitfall 2).
- Multi-leg shape offset: maintain `offset = shape.length` before each leg; add to every `begin_shape_index` (Pitfall 1).
- Coordinate flip: `decodePolyline` returns `[lng, lat]` (see polyline_util.ts line 64); push `[lat, lng]` for the response (Pitfall 3). Use default precision 6 (Pitfall 4).

**Error handling** (copy `handleError` usage from `trail/+server.ts` catch blocks):
```typescript
} catch (e: any) {
    return handleError(e);   // ZodError->400, SyntaxError->400, else 500
}
```
`handleError` source: `api_util.ts:158-168` — already maps `ZodError`/`SyntaxError` to 400.

---

### `web/src/lib/models/api/valhalla_navigate_schema.ts` (model, validation)

**Analog:** `web/src/lib/models/api/trail_schema.ts` (lines 1-28)

**Imports + export pattern** (copy from trail_schema.ts:1, 5):
```typescript
import { z, ZodType } from "zod";

const NavigateRequestSchema = z.object({ /* ... */ });
```
Named const exports per the `*_schema.ts` convention (e.g. `TrailCreateSchema`, `TrailUpdateSchema`).

**lat/lon bounds pattern** (copy directly from trail_schema.ts:14-15):
```typescript
lat: z.number().min(-90).max(90),
lon: z.number().min(-180).max(180),
```

**enum + default pattern** (difficulty enum at trail_schema.ts:13 is the analog shape):
```typescript
costing: z.enum(["pedestrian", "bicycle"]).default("pedestrian"),
```

**Composed schema** (per RESEARCH):
```typescript
const NavigateRequestSchema = z.object({
    waypoints: z.array(z.object({
        lat: z.number().min(-90).max(90),
        lon: z.number().min(-180).max(180),
    })).min(2).max(/* generous cap, e.g. 2000 */),  // DoS mitigation (RESEARCH Security)
    costing: z.enum(["pedestrian", "bicycle"]).default("pedestrian"),
});
```

---

### `web/src/lib/models/valhalla.ts` (model, MODIFY — extend types)

**Analog:** the file itself (existing `Leg`, `Summary`, `Trip` interfaces, lines 89-123)

**Existing `Leg` to extend** (lines 106-109) — currently only `summary` + `shape`; add a `maneuvers` field:
```typescript
export interface Leg {
    summary: Summary
    shape: string
    maneuvers?: ValhallaManeuver[]   // ADD
}
```

**New maneuver type** (follow the existing `interface` + `export` convention of this file; `Summary` at 111-123 is the shape analog):
```typescript
export interface ValhallaManeuver {
    instruction: string
    length: number
    begin_shape_index: number
    bearing_before?: number
    bearing_after?: number
}
```
Add navigate request/response types here too, or co-locate the response type with the endpoint — planner's discretion. Follow the file's existing `export { type X }` style at line 136 if using the named re-export block.

## Shared Patterns

### Authentication
**Source:** `web/src/routes/api/v1/search/actor/+server.ts` lines 97-100
**Apply to:** the navigate endpoint (D-09). Distinct from the existing Valhalla proxies which are unauthenticated.
```typescript
if (!event.locals.user) {
    return error(401, "Unauthorized");
}
```

### Error Handling
**Source:** `web/src/lib/util/api_util.ts` lines 6-17 (`APIError`), 158-168 (`handleError`)
**Apply to:** the navigate endpoint catch block. Throw `APIError(status, msg)` for domain errors; `handleError(e)` in catch maps ZodError/SyntaxError->400, else 500.
```typescript
} catch (e: any) {
    return handleError(e);
}
```

### Valhalla URL Resolution
**Source:** `web/src/lib/server/valhalla.ts` lines 3-5 (`getValhallaBaseUrl()` -> `resolveBaseUrl("VALHALLA_URL")`)
**Apply to:** the endpoint's upstream call. Always use this — never read `env.VALHALLA_URL` directly (closes SSRF surface; URL is server-controlled).

### Polyline Decoding
**Source:** `web/src/lib/util/polyline_util.ts` lines 26-68 (`decodePolyline`, default precision 6, returns `[lng, lat]` at line 64)
**Apply to:** decoding `leg.shape`. Flip each pair to `[lat, lon]` for the response. A `flipped()` helper already exists at lines 86-93 for reference (it builds `[coord[1], coord[0]]`), but it is not exported — replicate the flip inline.

### Swagger Documentation
**Source:** `web/src/routes/api/v1/valhalla/route/+server.ts` lines 9-34
**Apply to:** the navigate endpoint — all routes in this codebase carry an `@swagger` JSDoc block (project convention per CLAUDE.md).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (Valhalla->Flutter response transform logic) | route internal | transform | No existing endpoint transforms a Valhalla response — the only analogs (`route`, `height`) are raw proxies via `proxyJsonResponse`. The transform + multi-leg shape-index offset is genuinely new; follow RESEARCH Pattern 1 & 2. |

## Metadata

**Analog search scope:** `web/src/routes/api/v1/valhalla/`, `web/src/routes/api/v1/search/`, `web/src/routes/api/v1/trail/`, `web/src/lib/models/api/`, `web/src/lib/models/`, `web/src/lib/util/`, `web/src/lib/server/`
**Files scanned:** 7
**Pattern extraction date:** 2026-06-12
