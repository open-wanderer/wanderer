# Phase 13: Glyph & Sprite Endpoint - Pattern Map

**Mapped:** 2026-07-08 (revised after user redirected scope to a unified `/map/config` endpoint replacing `/map/tileurl`)
**Files analyzed:** 4 (1 new, 1 replaced, 2 modified)
**Analogs found:** 4 / 4 (exact)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `web/src/routes/api/v1/map/config/+server.ts` (new; replaces `tileurl/+server.ts`; exact segment name is Claude's discretion) | route (SvelteKit config endpoint) | request-response (config lookup, JSON out) | `web/src/routes/api/v1/map/tileurl/+server.ts` (being replaced — same file, superset shape) | exact |
| `web/src/routes/api/v1/map/tileurl/+server.ts` | deleted | — | — | n/a — removed, folded into `config/+server.ts` |
| `app/lib/provider/tile_url_provider.dart` (+ generated `tile_url_provider.g.dart`) | Riverpod provider (data fetch) | Dio GET → parse JSON → typed provider value | itself (modify in place, or rename to `map_config_provider.dart` — Claude's discretion per CONTEXT.md) | exact (self-analog, shape unchanged, just repoints URL + parses more fields) |
| `app/lib/provider/map_style_provider.dart` | Riverpod provider (consumer) | reads `tileUrlProvider.future` → builds `Style` | itself (modify in place — must keep reading a tile URL string with identical behavior) | exact (self-analog, minimal diff expected) |

**Scope note (from CONTEXT.md):** This phase creates/modifies exactly these files. RESEARCH.md's Go byte-serving routes (`db/routes/map_fonts.go`, `map_sprite.go`), SvelteKit binary proxies (`fonts/`, `sprite/`), Docker asset vendoring, and any `db/` changes are explicitly **superseded and out of scope**. The endpoint returns URL *templates* (defaulting to Protomaps' public assets), not bytes. Actually wiring `glyphs`/`sprite` into the rendered style is Phase 15 (GLYPH-04) — this phase only fetches and holds those two fields on the Flutter side.

## Pattern Assignments

### `web/src/routes/api/v1/map/config/+server.ts` (route, request-response — replaces `tileurl/+server.ts`)

**Analog:** `web/src/routes/api/v1/map/tileurl/+server.ts` (full file, 45 lines — read below, this is the file being folded into the new one)

```typescript
import { env } from '$env/dynamic/private';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/map/tileurl:
 *   get:
 *     summary: Get the configured tile server URL
 *     description: >
 *       Returns the vector tile URL template for this instance.
 *       Requires authentication. Operators can override the default by setting
 *       TILE_SERVER_URL; otherwise falls back to the built-in Protomaps endpoint.
 *     tags:
 *       - Maps
 *     responses:
 *       200:
 *         description: Tile URL template
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 url:
 *                   type: string
 *                   example: "https://tiles.example.com/{z}/{x}/{y}.mvt"
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.pb.authStore.record) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const tileUrl = env.TILE_SERVER_URL
            ?? `https://api.protomaps.com/tiles/v4/{z}/{x}/{y}.mvt?key=${env.PROTOMAPS_API_KEY ?? ''}`;

        return json({ url: tileUrl });
    } catch (e) {
        return handleError(e);
    }
}
```

**How to fold into the unified route** (per CONTEXT.md "One unified endpoint, not three"):
- Keep the tile URL logic **exactly as-is** (same `TILE_SERVER_URL` env var, same Protomaps fallback URL, same `PROTOMAPS_API_KEY` usage) — this is unchanged behavior, just relocated.
- Add the glyph/sprite logic alongside it, gated by a **new, separate** env var (name Claude's discretion, e.g. `GLYPH_SERVER_URL`) — do not conflate with `TILE_SERVER_URL`:
```typescript
// Unset -> return the public Protomaps basemaps-assets URLs verbatim:
//   glyphs: https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf
//   sprite: https://protomaps.github.io/basemaps-assets/sprites/v4/light  (dark via /v4/dark)
// Set   -> operator base host, same basemaps-assets path layout:
//   {base}/fonts/{fontstack}/{range}.pbf , {base}/sprites/v4/{light,dark}
```
- Merge into one response object, e.g. `return json({ tileUrl, glyphs, sprite });` (exact key names Claude's discretion — must be self-describing, distinct from the old ambiguous `{ url }` key).
- Keep the same auth check, same `try/catch` → `handleError` wrapper, same `@swagger` JSDoc convention — update the JSDoc path to `/api/v1/map/config` and the response schema to document all three fields.
- **Delete** `web/src/routes/api/v1/map/tileurl/+server.ts` (and its directory if empty) once the new route is in place — do not leave both routes live.

Notes for the planner:
- Keep the `{fontstack}` and `{range}` tokens literal (unescaped) in the returned glyph template — MapLibre substitutes them (RESEARCH Code Examples, glyphs contract). The template is fontstack-agnostic, so all 4 fontstacks incl. `Noto Sans Devanagari Regular v1` resolve automatically.
- Sprite is a *base* URL (no `.json`/`.png` suffix) — MapLibre appends `.json`/`.png`/`@2x` (RESEARCH Pitfall 3). Both `light` and `dark` variants must be resolvable by the returned shape (CONTEXT.md "Sprite variants").
- URL-shape validation of the override env var is optional/low-priority (operator-controlled config, not user input — CONTEXT.md discretion + RESEARCH Security SSRF note).

---

### `app/lib/provider/tile_url_provider.dart` (Flutter Riverpod provider — modify or rename)

**Analog:** itself, current full content:
```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'tile_url_provider.g.dart';

@Riverpod(keepAlive: true)
Future<String> tileUrl(Ref ref) async {
  final api = ref.watch(apiProvider);
  final response = await api.get('/map/tileurl');
  return response.data['url'] as String;
}
```

**Required change:** repoint the request to `/map/config` and parse the merged response. Per CONTEXT.md, pick one of:
- **(a) Minimal diff:** keep `tileUrlProvider` name and `Future<String>` return type; internally call `/map/config` and return just `response.data['tileUrl'] as String` (or whatever key name the new endpoint uses). `map_style_provider.dart` needs zero changes.
- **(b) New typed provider:** add `app/lib/provider/map_config_provider.dart` exposing a typed value (e.g. a small Dart class/record with `tileUrl`, `glyphs`, `sprite` fields) fetched once via `@Riverpod(keepAlive: true)` from `/map/config`; update `map_style_provider.dart` to read `.tileUrl` off it instead of watching `tileUrlProvider`.

Either way:
- Preserve `@Riverpod(keepAlive: true)` — config must still be fetched once and cached.
- Use the existing `apiProvider` (`app/lib/provider/api_provider.dart`) Dio client — same `api.get(...)` call shape, just a new path.
- Regenerate the Riverpod codegen file (`build_runner`) for whichever provider changes — `tile_url_provider.g.dart` currently declares `final tileUrlProvider = TileUrlProvider._();` and related codegen boilerplate; if renaming to `map_config_provider.dart`, a new `.g.dart` is generated the same way, and the old `tile_url_provider.dart`/`.g.dart` pair is deleted if provider (a) is not chosen.

---

### `app/lib/provider/map_style_provider.dart` (Flutter Riverpod provider — consumer, modify only if approach (b) chosen)

**Analog:** itself, current full content:
```dart
import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:vector_tile_renderer/vector_tile_renderer.dart' as vtr;
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/tile_url_provider.dart';

part 'map_style_provider.g.dart';

@Riverpod(keepAlive: true)
Future<Style> mapStyle(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final tileUrl = await ref.watch(tileUrlProvider.future);
  final brightness = effectiveBrightness(mode);
  final asset = brightness == Brightness.dark
      ? vtr.wandererDarkTheme(tileUrl)
      : vtr.wandererLightTheme(tileUrl);
  return StyleReader.map(asset).read();
}
```

**Required change:** only if approach (b) above is chosen — swap the import and the `ref.watch(tileUrlProvider.future)` line for the new provider's tile-URL field, keeping every other line (theme selection, `StyleReader`) untouched. `vtr.wandererDarkTheme`/`wandererLightTheme` still take only a tile URL string in this phase — do NOT thread `glyphs`/`sprite` through here; that wiring is Phase 15 (GLYPH-04).

---

## Shared Patterns

### Authentication (SvelteKit)
**Source:** `web/src/routes/api/v1/map/tileurl/+server.ts` lines 34-36
```typescript
if (!event.locals.pb.authStore.record) {
    return json({ message: "Unauthorized" }, { status: 401 });
}
```
**Apply to:** the new `config/+server.ts` route.

### Error Handling (SvelteKit)
**Source:** `web/src/lib/util/api_util.ts:161` (`handleError`), used at tileurl lines 42-44
**Apply to:** the new `config/+server.ts` route (wrap handler body in `try { ... } catch (e) { return handleError(e); }`).

### Env-var override (SvelteKit)
**Source:** `web/src/routes/api/v1/map/tileurl/+server.ts` lines 1, 38-39 (`import { env } from '$env/dynamic/private'` + `env.TILE_SERVER_URL ?? <default>`)
**Apply to:** the new `config/+server.ts` route — reuse verbatim for tile URL; add a second, independent env var for glyph/sprite with the same `?? <public Protomaps default>` shape.

### Dio fetch + Riverpod keepAlive provider (Flutter)
**Source:** `app/lib/provider/tile_url_provider.dart` (full file) + `app/lib/provider/api_provider.dart` (shared Dio client, `baseUrl/api/v1`)
**Apply to:** whichever provider ends up calling `/map/config`.

## No Analog Found

None. All four files have exact self- or sibling-analogs (`tileurl/+server.ts` for the route; `tile_url_provider.dart`/`map_style_provider.dart` for themselves).

## Metadata

**Analog search scope:** `web/src/routes/api/v1/map/` (contains `cells/` and `tileurl/`), `app/lib/provider/` (`tile_url_provider.dart`, `map_style_provider.dart`, `api_provider.dart`)
**Files scanned:** `tileurl/+server.ts` (full), `api_util.ts` (handleError location), `tile_url_provider.dart` (full), `tile_url_provider.g.dart` (codegen shape), `map_style_provider.dart` (full), `api_provider.dart` (full)
**Pattern extraction date:** 2026-07-08 (revised for unified `/map/config` scope)
