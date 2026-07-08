# Phase 13: Glyph & Sprite Endpoint - Research

**Researched:** 2026-07-08
**Domain:** MapLibre glyph (SDF font PBF) and sprite serving from a Go/PocketBase backend, with an operator env-var override mirroring `TILE_SERVER_URL`
**Confidence:** HIGH

## Summary

Phase 13 makes the Wanderer server serve the fonts and icons the future MapLibre style will reference, under the same operator control as vector tiles. The current app style is the Dart `wandererLightTheme` / `wandererDarkTheme` in the `flomp/dart-vector-tile-renderer` fork; those two themes already declare exactly what the endpoint must serve: `glyphs: "https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf"` and `sprite: "https://protomaps.github.io/basemaps-assets/sprites/v4/{light|dark}"`. In other words, **the assets are the pre-baked Protomaps `basemaps-assets` fonts and sprites** — they do not need to be generated. Phase 13 vendors those assets and re-serves them from Wanderer's own host so the app is not dependent on `protomaps.github.io` (and so offline caching in Phase 15 has a same-origin source to pull from).

The serving architecture is already established by the `/map/cells` tile-cell endpoint: **Go owns the byte-serving route** (`db/routes/map_cells*.go`, registered in `db/main.go`, served with `e.FileFS(os.DirFS(...))`), and **SvelteKit proxies it** at `/api/v1/map/...` (binary streaming for the download route, `pb.send()` for JSON). The app's Dio client only reaches the backend through `{baseUrl}/api/v1`, so glyph/sprite bytes must be reachable there too. The operator override then mirrors `/api/v1/map/tileurl` exactly: a SvelteKit config route reads a private env var and returns the glyph + sprite base URLs, falling back to Wanderer's own endpoint when unset. This means the phase spans **both** `db/` (Go byte serving + asset bundling) and `web/` (proxy routes + config route) — the roadmap's "Go, `db/routes/`" description covers only the serving half.

**Primary recommendation:** Vendor the Protomaps `basemaps-assets` fonts (4 fontstacks — see finding below) and both v4 light+dark sprites into the `wanderer-db` image via a Docker build stage (mirroring the existing `curl`/`pmtiles` download stages), serve them from new Go routes under a `/map/fonts/...` + `/map/sprite/...` group using `e.FileFS`, proxy those through SvelteKit at `/api/v1/map/...` (mirroring `/map/cells`), and add a `/api/v1/map/tileurl`-shaped config route that returns glyph+sprite URLs honoring one new private env var. Validate `{fontstack}`, `{range}`, and sprite name against allowlists/regex to prevent path traversal.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Serve glyph PBF bytes (`{fontstack}/{range}.pbf`) | API/Backend (Go, `db/routes/`) | Frontend Server (SvelteKit proxy) | Mirrors `MapCellsDownload` file serving; `e.FileFS` is a PocketBase route primitive |
| Serve sprite files (`.json`/`.png`/`@2x`) | API/Backend (Go, `db/routes/`) | Frontend Server (SvelteKit proxy) | Same static-file serving concern as glyphs |
| Bundle/vendor the font+sprite assets | Backend build (Docker image / Go) | — | Assets ship with the `wanderer-db` container; `FROM scratch` image needs them COPY'd or embedded |
| Operator override (env var → glyph/sprite host) | Frontend Server (SvelteKit, `$env/dynamic/private`) | — | Must "mirror `TILE_SERVER_URL`", which is read in the SvelteKit `tileurl` route |
| Config resolution endpoint (returns URLs) | Frontend Server (SvelteKit) | — | Mirrors `/api/v1/map/tileurl`; app fetches it (Phase 15 GLYPH-04) |
| Inject/cache URLs into style | App (Flutter) — **OUT OF SCOPE (Phase 15)** | — | STYLE-02/03/04, GLYPH-04 belong to Phase 15 |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GLYPH-01 | Server serves SDF glyph PBFs at a `{fontstack}/{range}.pbf` route for the fontstacks the style references | The style (`wandererLightTheme`/`DarkTheme`) declares `glyphs: ".../fonts/{fontstack}/{range}.pbf"`; assets exist pre-baked in `protomaps/basemaps-assets` (256 range files per fontstack). Serve via Go `e.FileFS`. **See "4 fontstacks, not 3" finding.** |
| GLYPH-02 | Server serves a sprite sheet (`sprite.json`, `sprite.png`, `sprite@2x.png`) | The style declares `sprite: ".../sprites/v4/light"` and (dark theme) `.../v4/dark`. `basemaps-assets/sprites/v4` provides `light{,@2x}.{json,png}` and `dark{,@2x}.{json,png}`. Serve as static files; MapLibre appends `.json`/`.png`/`@2x`. |
| GLYPH-03 | Operator overrides glyph+sprite origin by env var, mirroring `TILE_SERVER_URL` | `TILE_SERVER_URL` is read in `web/src/routes/api/v1/map/tileurl/+server.ts` via `$env/dynamic/private` and returned as JSON. Replicate: one env var, one config route returning glyph+sprite URLs, falling back to Wanderer's own endpoint. |
</phase_requirements>

## Standard Stack

This phase adds **no new libraries**. It uses primitives already in the codebase.

### Core
| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| PocketBase Go router (`core.RequestEvent`, `e.FileFS`, `se.Router.Group`) | pocketbase 0.26.8 (`db/go.mod`) | Serve glyph/sprite bytes from a filesystem | Exact pattern already used by `MapCellsDownload` (`e.FileFS(os.DirFS("./pb_data/pmtiles_cache"), ...)`) [VERIFIED: codebase grep] |
| Go `os`/`io/fs` (`os.DirFS`, `fs.ValidPath`) | Go 1.25.0 (`db/Dockerfile`) | Filesystem access + path-traversal safety | Standard library; `os.DirFS` root-scopes serving [VERIFIED: codebase grep] |
| SvelteKit `+server.ts` route + `$env/dynamic/private` + `zod` | SvelteKit 2.60.1, zod 3.24.1 | Proxy byte routes + config/override route | Exact pattern in `map/cells/*` and `map/tileurl` routes [VERIFIED: codebase grep] |
| `e.locals.pb.send()` / `event.fetch(pb.baseURL + ...)` | — | SvelteKit → Go proxy (JSON vs binary stream) | Used in `cells/+server.ts` (JSON) and `cells/[cellKey]/download/+server.ts` (binary) [VERIFIED: codebase grep] |

### Supporting (build-time asset acquisition)
| Component | Purpose | When to Use |
|-----------|---------|-------------|
| Docker multi-stage `curl` download stage | Fetch `basemaps-assets` at image build | Mirrors existing `download-curl` / `download-pmtiles` stages in `db/Dockerfile` [VERIFIED: codebase grep] |
| `go:embed` (alternative) | Embed assets into the `pocketbase` binary | If assets are committed to the build context and you prefer a self-contained binary over a COPY'd dir |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Pre-baked `basemaps-assets` PBFs | Generate SDF glyphs with `font-maker`/`fontnik` | Pointless and risky — Protomaps already publishes the exact PBFs this schema expects; regenerating invites glyph-metric drift. **Don't.** |
| Pre-baked `basemaps-assets` sprites | Generate sprites with `spreet`/`spritezero` | Same — the shields + `arrow` icon are already in the Protomaps sprite. Regenerating means re-authoring every icon. **Don't.** |
| Vendoring assets into the image (build-time fetch) | Committing ~1000 binary files into git | Git bloat vs. a build-time network dependency on `protomaps.github.io`. Build-time fetch mirrors the existing pmtiles/curl pattern; committing is more reproducible/air-gap-friendly. Planner decision. |
| Go serves bytes + SvelteKit proxies | Go redirects (302) to operator host directly | Redirect is simpler but doesn't match GLYPH-04's "app resolves URLs from the server" model; the config-route approach mirrors `tileurl` faithfully. |

**Installation:** None. No `npm install` / `go get` required.

## Package Legitimacy Audit

No external packages are installed in this phase. The only external artifacts are **static font/sprite assets** downloaded from `github.com/protomaps/basemaps-assets`.

| Asset source | Registry | Age | Source Repo | Disposition |
|--------------|----------|-----|-------------|-------------|
| `protomaps/basemaps-assets` fonts (Noto Sans SDF PBFs) | GitHub (not a package registry) | Actively maintained; official Protomaps org | github.com/protomaps/basemaps-assets | Approved — official Protomaps assets, already referenced by the current app theme |
| `protomaps/basemaps-assets` sprites v4 (light/dark) | GitHub | same | same | Approved — same repo, same authority |

**License note (must carry through to attribution — see CORE-04 in Phase 15):**
- Fonts: **SIL Open Font License** (`fonts/OFL.txt` in the repo) [VERIFIED: GitHub API + repo README]
- Sprites: **MIT**, derived from `tangrams/icons` [CITED: github.com/protomaps/basemaps-assets README]

slopcheck not applicable (no registry package installs). No `postinstall` surface.

## Architecture Patterns

### System Architecture Diagram

```
                    Flutter app (Dio client, base = {origin}/api/v1)
                                    │
        ┌───────────────────────────┼─────────────────────────────┐
        │ (Phase 15 fetches config) │ (MapLibre-native fetches bytes)
        ▼                           ▼
  GET /api/v1/map/glyphurl    GET /api/v1/map/fonts/{fontstack}/{range}.pbf
  (SvelteKit config route)    GET /api/v1/map/sprite/{name}[@2x].{json|png}
        │                           │  (SvelteKit proxy routes)
        │ reads $env private        │ event.fetch(pb.baseURL + ...) — stream bytes
        │ GLYPH_SERVER_URL?         ▼
        │                     Go / PocketBase  (db/routes/)
        ▼                     GET /map/fonts/{fontstack}/{range}   → e.FileFS
  { glyphs, sprite } URLs     GET /map/sprite/{name}               → e.FileFS
   ├─ set   → operator host          │
   └─ unset → Wanderer's own         ▼
             /api/v1/map/...   bundled assets on disk (COPY'd into image
                               or go:embed): fonts/<stack>/<range>.pbf,
                               sprites/v4/{light,dark}{,@2x}.{json,png}
                                       ▲
                                       │ build time
                               Docker stage: curl basemaps-assets tarball
```

Trace of the primary case (unset env var): app asks `/api/v1/map/glyphurl` → SvelteKit sees `GLYPH_SERVER_URL` unset → returns `glyphs = {origin}/api/v1/map/fonts/{fontstack}/{range}.pbf`, `sprite = {origin}/api/v1/map/sprite/light` → MapLibre requests each range/sprite → SvelteKit proxy streams from Go `e.FileFS` reading the bundled file.

### Recommended Route/File Structure
```
db/
├── routes/
│   ├── map_fonts.go      # MapFontsGet: GET /map/fonts/{fontstack}/{range}
│   └── map_sprite.go     # MapSpriteGet: GET /map/sprite/{name...}
├── main.go               # register a Router.Group("/map") for fonts+sprite (public, like /map/cells)
├── map_assets/           # (option A) committed, or populated by a Docker build stage
│   ├── fonts/<fontstack>/<range>.pbf
│   └── sprites/v4/{light,dark}{,@2x}.{json,png}
└── Dockerfile            # add download stage + COPY map_assets ./map_assets

web/src/routes/api/v1/map/
├── tileurl/+server.ts        # existing precedent (do not change behavior)
├── glyphurl/+server.ts       # NEW: config route, returns {glyphs, sprite}, reads env var
├── fonts/[fontstack]/[range]/+server.ts   # NEW: binary proxy → Go
└── sprite/[...file]/+server.ts             # NEW: binary/json proxy → Go
```
(Exact SvelteKit route segment shapes are a planner decision; the byte routes must land under `/api/v1/map/` so the app's Dio base URL reaches them.)

### Pattern 1: Go static-file serving via `e.FileFS`
**What:** Serve a bundled file by validated path, root-scoped to an assets dir.
**When to use:** Both the glyph and sprite byte routes.
```go
// Source: pattern from db/routes/map_cells_id.go:91 (MapCellsDownload) [VERIFIED: codebase grep]
// e.FileFS(os.DirFS("./pb_data/pmtiles_cache"), cell.CacheKey()+".pmtiles")

func MapFontsGet(e *core.RequestEvent) error {
    fontstack := e.Request.PathValue("fontstack") // already URL-decoded by net/http
    rng := e.Request.PathValue("range")
    // VALIDATE before touching the filesystem (see Security Domain)
    if !allowedFontstack(fontstack) || !rangeRe.MatchString(rng) {
        return e.NotFoundError("unknown glyph", nil)
    }
    name := fontstack + "/" + rng + ".pbf"
    // os.DirFS root-scopes; fs.ValidPath rejects traversal
    return e.FileFS(os.DirFS("./map_assets/fonts"), name)
}
```

### Pattern 2: Config/override route (mirror `tileurl`)
**What:** Return glyph+sprite base URLs honoring one private env var.
**When to use:** GLYPH-03.
```typescript
// Source: mirror of web/src/routes/api/v1/map/tileurl/+server.ts [VERIFIED: codebase grep]
import { env } from '$env/dynamic/private';
import { json, type RequestEvent } from '@sveltejs/kit';
import { handleError } from '$lib/util/api_util';

export async function GET(event: RequestEvent) {
  try {
    if (!event.locals.pb.authStore.record) {
      return json({ message: 'Unauthorized' }, { status: 401 }); // tileurl requires auth
    }
    const base = env.GLYPH_SERVER_URL?.replace(/\/$/, '');          // [ASSUMED] env var name
    const origin = event.url.origin;
    const glyphs = base
      ? `${base}/fonts/{fontstack}/{range}.pbf`
      : `${origin}/api/v1/map/fonts/{fontstack}/{range}.pbf`;
    const sprite = base
      ? `${base}/sprites/v4/light`
      : `${origin}/api/v1/map/sprite/light`;
    return json({ glyphs, sprite });
  } catch (e) {
    return handleError(e);
  }
}
```

### Pattern 3: SvelteKit binary proxy (mirror cells download)
**What:** Stream bytes from Go through SvelteKit with correct `Content-Type`.
```typescript
// Source: pattern from web/src/routes/api/v1/map/cells/[cellKey]/download/+server.ts [VERIFIED]
const response = await event.fetch(`${event.locals.pb.baseURL}/map/fonts/${fontstack}/${range}`, {
  headers: { Authorization: event.locals.pb.authStore.token ? `Bearer ${...}` : '' },
});
return new Response(response.body, {
  status: response.ok ? 200 : response.status,
  headers: { 'Content-Type': 'application/x-protobuf', 'Cache-Control': 'public, max-age=31536000, immutable' },
});
```

### Anti-Patterns to Avoid
- **Generating glyphs/sprites at build or runtime.** They already exist pre-baked in `basemaps-assets`; regenerating drifts from the schema the tiles were built against.
- **Serving only the 3 named fontstacks.** The theme also uses `Noto Sans Devanagari Regular v1` in a data-driven `text-font` expression (see finding). Omitting it silently breaks Devanagari place-name labels.
- **Serving only one sprite variant.** The dark theme references `sprites/v4/dark`; CORE-02 (Phase 15) switches themes live. Serve both light and dark.
- **Passing the raw path param straight into `os.DirFS`.** Path traversal. Validate first.
- **Reading the override env var in Go.** `TILE_SERVER_URL` is read in SvelteKit via `$env/dynamic/private`; "same override shape" means the config route is SvelteKit, not `os.Getenv` in Go.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SDF glyph PBFs | `font-maker`/`fontnik` generation pipeline | Pre-baked `basemaps-assets/fonts/*` | Protomaps publishes the exact glyphs this basemap schema renders; 256 ranges × fontstack already generated |
| Sprite sheet + `sprite.json` | `spreet`/`spritezero` build | Pre-baked `basemaps-assets/sprites/v4/*` | The `arrow` icon and route-network shields are already authored in the Protomaps sprite |
| Static file serving | Custom `io.Copy` + MIME logic | PocketBase `e.FileFS(os.DirFS(...))` | Handles `Content-Type`, range requests, `fs.FS` root-scoping; already the codebase pattern |
| Path-param validation in SvelteKit | Ad-hoc string checks | `zod` schema (as in `cells/+server.ts`) | Consistent with existing map routes; rejects malformed input at the edge |

**Key insight:** This phase is almost entirely *plumbing and vendoring*, not generation. The single highest-leverage decision is recognizing the assets are Protomaps' own pre-built files, referenced verbatim by the current theme.

## Common Pitfalls

### Pitfall 1: URL-encoded fontstack path segment
**What goes wrong:** MapLibre requests `Noto%20Sans%20Regular` (spaces encoded). If the route or the vendored directory names don't line up after decoding, every glyph 404s.
**Why it happens:** The `{fontstack}` token is a font name with spaces; `basemaps-assets` stores them as literal-space directory names (`Noto Sans Regular/`).
**How to avoid:** Go's `net/http` decodes `PathValue` automatically → matches the literal-space dir. In the SvelteKit proxy, forward the encoded segment and let Go decode. Verify with a live request to `/fonts/Noto%20Sans%20Regular/0-255.pbf`.
**Warning signs:** Labels render as boxes/blank; 404s on `/fonts/...` in server logs.

### Pitfall 2: Only 3 fontstacks bundled (Devanagari dropped)
**What goes wrong:** Devanagari-script place names render blank while every other label works — a subtle, region-specific regression.
**Why it happens:** Requirements say "3 fontstacks"; the actual theme references a 4th via `["case", ["==", ["get","script"],"Devanagari"], ["literal",["Noto Sans Devanagari Regular v1"]], ...]`.
**How to avoid:** Bundle all 4 fontstacks (Regular, Medium, Italic, `Noto Sans Devanagari Regular v1`) — all exist in `basemaps-assets`. Zero extra tooling cost.
**Warning signs:** Requests for `/fonts/Noto%20Sans%20Devanagari%20Regular%20v1/...` returning 404.

### Pitfall 3: Sprite base-URL suffix convention
**What goes wrong:** Serving a file literally named `sprite.json` when the style's `sprite` value is a *base* (e.g. `.../sprite/light`), so MapLibre requests `light.json`, `light.png`, `light@2x.png`.
**Why it happens:** `sprite` in the style spec is a base path; the renderer appends `.json`, `.png`, and `@2x` for hi-dpi.
**How to avoid:** Route on `/map/sprite/{name}` and serve `{name}.json`, `{name}.png`, `{name}@2x.json`, `{name}@2x.png` where `name ∈ {light, dark}`. Set `Content-Type: application/json` for `.json`, `image/png` for `.png`.
**Warning signs:** Icons/shields missing; 404s for `@2x.png` on retina devices.

### Pitfall 4: `FROM scratch` image has no shell and a relative CWD
**What goes wrong:** `os.DirFS("./map_assets/fonts")` resolves relative to the process working dir. The container entrypoint is `/pocketbase serve --dir=/pb_data`; the CWD is `/`. If assets are COPY'd to `/map_assets`, the relative path must match (`./map_assets` from `/`).
**Why it happens:** The final stage is `FROM scratch` with `WORKDIR /`; existing code uses `./pb_data/...` relative paths.
**How to avoid:** COPY assets to `/map_assets` and reference `os.DirFS("./map_assets/...")`, or use `go:embed` to sidestep filesystem layout entirely. Confirm against the `ENTRYPOINT` working dir.
**Warning signs:** `file does not exist` errors at runtime despite the file being in the image.

### Pitfall 5: Missing cache headers / wrong Content-Type
**What goes wrong:** Glyph/sprite requests are re-fetched constantly, or the renderer rejects a PBF served as `text/plain`.
**How to avoid:** `Content-Type: application/x-protobuf` for `.pbf`, `application/json` for sprite json, `image/png` for sprite png. Set `Cache-Control: public, max-age=31536000, immutable` — these assets are versioned/static.

## Code Examples

### MapLibre glyphs URL template (the contract the route must satisfy)
```
glyphs = "<base>/{fontstack}/{range}.pbf"
  {fontstack} → comma-separated font stack from a layer's text-font (here: single font, space-containing, URL-encoded)
  {range}     → a 256-codepoint block: 0-255, 256-511, ... 65280-65535 (chosen dynamically by the renderer)
// Source: https://maplibre.org/maplibre-style-spec/glyphs/ [CITED]
```

### Sprite request expansion (what MapLibre fetches from the sprite base)
```
sprite = "<base>/light"   →  GET <base>/light.json
                             GET <base>/light.png
                             GET <base>/light@2x.json   (hi-dpi)
                             GET <base>/light@2x.png    (hi-dpi)
// Confirmed by basemaps-assets/sprites/v4 file listing [VERIFIED: GitHub API]
```

### Current theme declarations (source of truth for what to serve)
```dart
// wanderer_light_theme.dart (flomp/dart-vector-tile-renderer @ d52dd7d) [VERIFIED: pub-cache grep]
"sprite": "https://protomaps.github.io/basemaps-assets/sprites/v4/light",
"glyphs": "https://protomaps.github.io/basemaps-assets/fonts/{fontstack}/{range}.pbf",
// wanderer_dark_theme.dart → sprites/v4/dark
// icon-image "arrow" (line 2046); route shields via icon-image match on network (US:I, NL:S-road, generic_shield-<n>char)
```

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| App renders fonts client-side via `vector_tile_renderer` bundled Noto (no glyph endpoint) | MapLibre-native fetches SDF glyph PBFs over HTTP from a `{fontstack}/{range}.pbf` endpoint | Phase 13 must stand up that endpoint before Phase 15 can render or cache labels |
| Sprites/shields silently dropped (`flutter_map` stack renders no `icon-image`) | Server-hosted sprite sheet; MapLibre renders `arrow` + shields | New capability; icons that "render nowhere today" begin rendering in Phase 15 |

**Deprecated/outdated:** Nothing to remove in this phase (Phase 18 removes the forks). Success criterion 4 requires the current `flutter_map` app to keep building untouched.

## Runtime State Inventory

Not a rename/refactor phase — this is additive backend serving. No stored data, live-service config, OS-registered state, secrets, or build artifacts are renamed or migrated. New env var and new asset files are purely additive.

**Verified:** No existing route, migration, collection, or federation path is modified (success criterion 4). The `/map/cells` group and `tileurl` route are read as precedent only, not changed.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The new override env var is a single variable named e.g. `GLYPH_SERVER_URL` covering both glyphs and sprites | Pattern 2, GLYPH-03 | Low — success criterion says "one environment variable"; only the *name* is unconfirmed. Planner/operator picks the final name. |
| A2 | The config route requires auth (returns 401 unauthenticated), matching `tileurl` | Pattern 2 | Low — `tileurl` requires auth; byte routes likely public like `/map/cells` (auth commented out). Confirm desired auth posture. |
| A3 | Assets are best acquired via a Docker build-stage fetch of the `basemaps-assets` tarball | Standard Stack, structure | Low/Medium — committing to git or `go:embed` are equally valid; this is a reproducibility-vs-repo-size tradeoff for the planner. |
| A4 | The Phase 15 style will keep the `Noto Sans Devanagari Regular v1` data-driven font | Pitfall 2 | Medium — if Phase 15 drops it, only 3 stacks are needed; bundling the 4th anyway is cheap insurance. |
| A5 | Both light and dark sprites are needed | Pitfall 3 | Low — dark theme references `v4/dark` and CORE-02 switches themes live. |

## Open Questions

1. **Env var name and whether it is one var or two.**
   - What we know: Success criterion 3 says "one environment variable" for both glyph and sprite; must mirror `TILE_SERVER_URL`.
   - What's unclear: Exact name (`GLYPH_SERVER_URL`? `MAP_ASSETS_URL`?), and whether the operator supplies a single base host from which both `/fonts/...` and `/sprites/v4/...` derive, or full templates.
   - Recommendation: One base-URL var; the config route derives glyph template + sprite base from it. Document the expected path layout an operator's host must expose (must match `basemaps-assets` shape).

2. **Asset bundling mechanism (build-fetch vs. commit vs. embed).**
   - What we know: `FROM scratch` image; existing `curl`/`pmtiles` build stages; `migrations`/`templates` COPY'd as dirs; `e.FileFS(os.DirFS(...))` is the serving idiom.
   - What's unclear: Team preference on git size (~1000 small binaries) vs. build-time network dependency.
   - Recommendation: Docker build-stage fetch of the `basemaps-assets` tarball → COPY into image → `os.DirFS`. Pin a commit/tag for reproducibility.

3. **Sprite naming exposed to the style: `light`/`dark` vs. generic `sprite`.**
   - What we know: Success criterion literally names `sprite.json`/`sprite.png`/`sprite@2x.png`; the assets are named `light`/`dark`.
   - Recommendation: Serve under `/map/sprite/{name}` with `name ∈ {light,dark}`; the config route returns the light base by default and the app (Phase 15) swaps to dark for the dark style. Avoid a single generic `sprite` name that can't express both themes.

4. **Full 256-range set vs. a needed subset per fontstack.**
   - What we know: The renderer picks ranges dynamically from label codepoints worldwide; `basemaps-assets` ships all 256 ranges per stack. Phase 15 offline caching (OFFL-01) will fetch whatever ranges a downloaded region needs.
   - Recommendation: Bundle the complete 256-range set per fontstack (what the upstream host serves). Do not attempt to prune — a pruned range 404s for legitimate labels.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building the new routes | ✓ | 1.25.0 (`db/Dockerfile`) | — |
| PocketBase | Route host | ✓ | 0.26.8 (`db/go.mod`) | — |
| Docker build network → `github.com` / `protomaps.github.io` | Build-time asset fetch | ✓ (build already curls github.com for curl/pmtiles) | — | Commit assets to git instead |
| Node/SvelteKit | Proxy + config routes | ✓ | 22 / 2.60.1 | — |
| `curl` (build stage) | Fetch `basemaps-assets` | ✓ (already a build stage) | 8.18.0 | `git clone --depth 1` |

**Missing dependencies with no fallback:** None.
**Missing with fallback:** Build-time network to `protomaps.github.io`/`github.com` — fallback is committing the assets into the repo.

## Security Domain

`security_enforcement: true`, ASVS level 1. The dominant risk is **path traversal via untrusted path parameters** feeding a filesystem read.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | maybe | Config route mirrors `tileurl` (auth-gated); byte routes likely public like `/map/cells`. Decide posture (A2). |
| V3 Session Management | no | Stateless static serving |
| V4 Access Control | no | Public/derived assets; no per-user authorization |
| V5 Input Validation | **yes** | Allowlist `{fontstack}`, regex `{range}` (`^\d+-\d+$`), allowlist sprite `{name}` (`^(light|dark)(@2x)?$`) + extension allowlist; reject before filesystem access. Use `zod` in SvelteKit proxy (as `cells/+server.ts` does) and explicit checks in Go. |
| V6 Cryptography | no | No secrets handled beyond an env var URL |

### Known Threat Patterns for Go file-serving + SvelteKit proxy
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal (`{fontstack}` / `{range}` = `../../etc/...`) | Tampering / Info Disclosure | Validate against allowlist/regex; serve via `os.DirFS` (root-scoped) + `fs.ValidPath`; never `filepath.Join` raw params to an absolute path |
| SSRF via override env var | — (operator-set, not user input) | Env var is operator-controlled config, not request input; low risk. Do not reflect user-supplied hosts. |
| Content-type confusion (PBF served as HTML/JS) | — | Explicit `Content-Type` per extension; `X-Content-Type-Options: nosniff` |
| Proxy open-forwarding (SvelteKit fetches arbitrary Go paths) | Tampering | Validate params in the proxy before constructing the `pb.baseURL + path`; do not pass unvalidated segments |

## Sources

### Primary (HIGH confidence)
- Codebase (`db/routes/map_cells.go`, `map_cells_id.go`, `db/main.go:213-219`, `db/Dockerfile`, `web/src/routes/api/v1/map/tileurl/+server.ts`, `web/src/routes/api/v1/map/cells/**`, `app/lib/provider/{tile_url_provider,api_provider,map_style_provider}.dart`) [VERIFIED: grep/read]
- `flomp/dart-vector-tile-renderer @ d52dd7d` `wanderer_light_theme.dart` / `wanderer_dark_theme.dart` (pub-cache, the pinned fork) — glyphs/sprite URLs, 14 symbol layers, fontstacks, `arrow` + shield icon-images [VERIFIED: pub-cache grep]
- `github.com/protomaps/basemaps-assets` fonts + sprites/v4 directory listings and licenses [VERIFIED: GitHub API]
- MapLibre Style Spec — glyphs URL template + `{fontstack}`/`{range}` semantics [CITED: https://maplibre.org/maplibre-style-spec/glyphs/]

### Secondary (MEDIUM confidence)
- Sprite `@2x`/`.json`/`.png` request expansion — inferred from MapLibre spec + confirmed by the `basemaps-assets/sprites/v4` file names

### Tertiary (LOW confidence)
- None load-bearing. Env var name (A1) is the only unverified item and is flagged as a decision.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every primitive is already used in the codebase for the analogous `/map/cells` + `tileurl` routes.
- Architecture: HIGH — serving/proxy/override patterns are directly mirrored from existing, verified code.
- Assets & fontstacks: HIGH — read directly from the pinned theme fork and confirmed against the live `basemaps-assets` repo (incl. the 4th Devanagari fontstack).
- Pitfalls: HIGH — derived from the actual theme declarations, the scratch-image Dockerfile, and MapLibre URL conventions.

**Research date:** 2026-07-08
**Valid until:** 2026-08-07 (stable — depends on established Protomaps assets and existing codebase patterns; re-check only if `basemaps-assets` v4 layout or the fork's theme URLs change)
