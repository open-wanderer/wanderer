# Phase 45: App Map Overlays and POIs - Research

**Researched:** 2026-11-04
**Domain:** SvelteKit server-side Overpass caching/rate-limiting proxy + Flutter/MapLibre runtime raster overlays and symbol-layer POIs
**Confidence:** MEDIUM-HIGH (server design and repo facts HIGH; three native-platform behaviours must be confirmed on a device, see Open Questions)

<user_constraints>
## User Constraints (from `.planning/notes/app-map-overlays-pois.md`, no CONTEXT.md exists)

Copied from the /gsd-explore decisions note. Treat as locked.

### Locked Decisions

- **Base styles are deferred.** Pending a maintainer meeting. The app keeps the bundled Protomaps-schema wanderer style.
- **Overlays and POIs only, online only.** Waymarked Trails rasters (hiking, cycling, MTB, skiing) and the web's ~30 Overpass POI categories. While `onlineStatusProvider` reports offline, the layers are hidden and the toggles disabled; the selection is kept and returns on reconnect. `onlineStatusProvider` measures backend reachability, not third-party tile hosts, acceptable for v1.
- **One global state**, like the web's `MapState`. Picker on the main map, trail detail and route planner. Not on navigation/recording in v1 (battery, live Overpass traffic).
- **Overlays** are added at runtime as raster source + layer, inserted below the app's trail and route layers (same `style.addSource`/`addLayer` path as `trail_layer.dart`). Runtime layers bypass `rewriteStyleForProxy`, so they go straight to the network.
- **POI detail sheet: curated OSM tags.** Address, opening hours, website and phone (tappable), wheelchair, elevation for summits/passes, drinking water/fee where relevant; raw tags behind "Show all tags". No backend work beyond the POI endpoint.
- **Protomaps POIs rejected as a data source.** Overpass stays the only POI backend.
- **POI icons** are SVGs on a coloured background on the web; MapLibre needs bitmaps, so the app rasterizes each once before registering it.
- **Overpass rate-limit plan:** (1) Server: structured `/api/v1/pois?tile=x,y&category=...` endpoint. The server builds the query, caches per (tile, category) for days in persistent storage, single-flights concurrent misses, sends one upstream request at a time, backs off 30 s on 429/406 while serving stale cache, and sets a wanderer `User-Agent`. The web Overpass layer moves onto it. (2) App: persistent per-(tile, category) cache in ObjectBox with a TTL; fetch only after the camera settles, from zoom 12, with a per-screen tile cap. (3) Operators: document pointing `OVERPASS_API_URL` at an own or keyed instance for larger installs.
- Server and API changes ship as a **new PR stacked on #1397** (the app-merge 1394-1397 stack), not inside #1397 itself. App work lands on `feature/app`.

### Claude's Discretion

Everything the note leaves open: tile key format, cache storage technology, response shape, validation, queue/backoff mechanics, client-facing behaviour on cold miss during backoff, User-Agent string, app state persistence mechanism, SVG rasterization method, ObjectBox entity design, layer anchoring, detail-sheet implementation, i18n keys. Recommendations below.

### Deferred Ideas (OUT OF SCOPE)

- Base map styles (OpenTopoMap/OpenHikingMap/CyclOSM/OpenFreeMap/Carto): maintainer meeting; "Offline-capable base styles" seed.
- POI enrichment (Wikipedia/Commons): separate seed. If ever done, follow only explicit OSM `wikidata`/`wikipedia`/`wikimedia_commons` tags, never nearest article.
- Porting the curated formatting back to the web popup ("could be ported back later").
- Stretch (decided only as "stretch"): tap POI in route planner to add it as an anchor.
</user_constraints>

**Phase requirement IDs:** none mapped (ROADMAP: "Requirements: none mapped"), so there is no `<phase_requirements>` table. The five ROADMAP success criteria are the acceptance targets.

## Project Constraints (from CLAUDE.md and project memory)

- TypeScript strict; explicit return types on functions; zod for API input validation; named exports; snake_case file names for utilities (`CLAUDE.md` Conventions). Note: the existing server files (`web/src/lib/server/overpass.ts`, `nominatim.ts`) use **4-space** indentation although CLAUDE.md says 2. Match the surrounding file. [VERIFIED: read both files]
- CLAUDE.md's "Project" section (instance federation) is unrelated to this phase; only Stack/Conventions apply.
- CLAUDE.md "GSD Workflow Enforcement": file edits go through a GSD command. Memory `feedback-no-gsd-unless-asked` says do not invoke gsd skills unless asked; the planner/executor flow is user-initiated here.
- Memory directives that bind execution: never run `flutter build` / `adb install` (user builds; hand off after `flutter analyze` + `flutter test`); no push without permission; commits headline-only (no body); no emojis in user-facing copy; web full test suite is left to CI (use targeted vitest); no multi-arch Docker builds; PR drafts as raw markdown in a fenced block; Riverpod: no `late final` in `build()`, use `.value` not `.valueOrNull`; do not `npm run build` the docs.
- Memory `project-upgrade-paths-release-and-app`: any DB migration must upgrade from both `vX` and `vX-app`. This is a reason to avoid a PocketBase-backed cache (Section "Cache storage").
- Memory `feedback-audit-against-real-dependencies`: verify against the real Overpass with realistic tiles (dense city tile, empty tile, restart) rather than fakes only.

## Summary

The web already has the whole client side: a 0.1 degree grid, zoom-12 gate, per-category icons, popup. The only server piece is a pass-through proxy (`web/src/lib/server/overpass.ts`, no cache, no User-Agent, immediate retry only on thrown errors) that forwards arbitrary client-built Overpass QL. The phase replaces that with a structured `/api/v1/pois` endpoint whose key space (integer tile index x whitelisted category) makes caching effective, then builds the app features on top of it.

Recommended server design: file-per-(tile, category) JSON cache in a configurable directory (`POI_CACHE_DIR`, new compose volume), stale-while-revalidate with 7-day fresh / 30-day retention, in-process serial upstream queue (the web container is single-process), per-key single-flight, a 30 s global backoff on 429/406/5xx/timeouts during which stale data is served and cold keys report `unavailable`, and a wanderer User-Agent. No new npm packages. PocketBase and SQLite are rejected (migration/upgrade-path cost and public-collection surface; `node:sqlite` is still experimental on Node 22).

Recommended app design: one keepAlive Riverpod notifier holding the selection (persisted as a JSON string field on the device-level `LocalSettingsEntity`), a `reconcile()` routine run from every host's `onStyleLoaded` (a style reload drops runtime layers and images), raster overlays via `RasterSource(tileSize: 256)` + `RasterStyleLayer`, one GeoJSON source + one symbol layer for POIs fed from an ObjectBox TTL cache, camera-idle driven fetching, bitmaps rasterized once with `flutter_svg`'s `vg.loadPicture`. Layers are hidden by removing them (the Dart StyleController has no visibility/filter setter).

**Primary recommendation:** Build the server PR first on a branch cut from `origin/app-merge/4-web-app-support` (its touched files are byte-identical to `feature/app`, so the commits cherry-pick cleanly onto `feature/app`), define the response contract below as the interface, then build the app against that contract.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Category id -> Overpass tag filters (query construction) | API / Backend (SvelteKit server) | - | Roadmap: server builds the query itself; also closes the arbitrary-QL proxy hole |
| Per-(tile, category) persistent cache | Server disk (file store under `POI_CACHE_DIR`) | In-memory Map in front | Must survive restart; web container has no DB access of its own |
| Single-flight, serial upstream queue, 30 s backoff, UA | API / Backend | - | All users share one server IP; only the server can enforce policy |
| Tile index math (lon/lat -> integer tile) | Browser / App client | Server validates range | Both clients compute indices; server only derives bbox from them |
| Icons, colours, category display names, grouping | Browser / App client | - | Presentation; web keeps `layers.ts`, app has its own Dart table |
| Waymarked Trails raster overlays | Third-party tile host (direct from device) | - | Locked: runtime layers bypass the proxy |
| App POI TTL cache | App local DB (ObjectBox) | In-memory map | Locked decision; avoids chatty requests while panning |
| Overlay/POI selection state | App local settings (device-level) | Riverpod notifier | Global, survives restart and logout (like theme) |
| Offline hide/disable | App client (`onlineStatusProvider`) | - | Locked decision |
| POI detail formatting | App client | - | Curated formatting is app-only in v1 |

## Standard Stack

No new external packages are needed on either side. [VERIFIED: every capability below is already a dependency]

### Core (existing, versions read from manifests this session)

| Library | Version | Purpose | Source |
|---------|---------|---------|--------|
| zod | ^3.24.1 | Validate `tile` / `category` query params on the new route | `web/package.json` ("zod": "^3.24.1"); pattern in `web/src/routes/api/v1/waypoint/cluster/+server.ts` |
| Node built-ins (`node:fs/promises`, `node:path`, `node:os`) | Node 22 (`FROM node:22-alpine`) | File cache | `web/Dockerfile` |
| vitest | ^4.1.11 | Server unit tests | `web/package.json` |
| maplibre (Dart) | **0.3.7** (pinned; `CLAUDE.md` and the phase brief say 0.3.5, which is outdated) | Runtime sources/layers, camera idle, feature queries | `app/pubspec.yaml:43` `maplibre: 0.3.7` |
| flutter_svg | ^2.2.4 | Rasterize POI SVG on a coloured circle | `app/pubspec.yaml:29` |
| objectbox / objectbox_flutter_libs / objectbox_generator | **6.0.0-preview.5** (CLAUDE.md says 5.3.1, outdated) | POI TTL cache entity | `app/pubspec.yaml:45` |
| url_launcher | ^6.3.2 | Tappable website/phone | `app/pubspec.yaml:62` |
| riverpod_annotation / riverpod_generator | ^4.0.2 / ^4.0.3 | Selection notifier | `app/pubspec.yaml` |
| xml | ^6.3.0 | Already used to massage SVG strings (`app/lib/util/route_placeholder.dart`) | `app/pubspec.yaml` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| File-per-key cache | `node:sqlite` | Works on the local Node 22.23.3 but prints `ExperimentalWarning: SQLite is an experimental feature` [VERIFIED: ran it]. Rejected while experimental |
| File-per-key cache | `better-sqlite3` | Native dependency, musl prebuild risk on alpine, new supply-chain surface for ~thousands of tiny records. Rejected |
| File-per-key cache | PocketBase collection | Needs a Go migration (upgrade-path rule for `vX` and `vX-app`), a collection is a public REST surface (see `.planning/notes/pocketbase-collections-are-a-public-api-surface.md`), and the web server has no privileged PocketBase client: the only superuser-ish secret is `POCKETBASE_PROXY_SECRET`, used solely as an `X-Internal-Secret` header on the ActivityPub inbox proxy [VERIFIED: grep, `web/src/routes/api/v1/activitypub/user/[handle]/inbox/+server.ts:56`; no `_superusers` usage anywhere in `web/src`]. Rejected |
| Raster per-icon via `flutter_svg` | `FontAwesomeIcons` + `addImageFromIconData` | app uses font_awesome_flutter 11.0.0 = FA **7.2.0** while the web SVGs are FA **6.7.2** plus six non-FA SVGs (attraction, bike-*, climbing, mountain-pass), so glyphs would not match [VERIFIED: pubspec.lock, package README, SVG headers]. Use the SVG files |

**Installation:** none. Dart-side after adding the entity/provider: `cd app && dart run build_runner build --delete-conflicting-outputs` [VERIFIED: `docs/src/content/docs/develop/app-development.md`].

## Package Legitimacy Audit

No new packages are installed by this phase (server: zod, Node built-ins; app: all dependencies already in `pubspec.yaml`). Audit table intentionally empty.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none) | - | - | - | - | - | - |

**Packages removed due to SLOP:** none. **Packages flagged SUS:** none.

## Architecture Patterns

### System Architecture Diagram

```
 App (Flutter)                                  Web (browser)
 MapEventCameraIdle -> debounce                 map 'moveend'
   -> visible tiles (z>=12, cap N)                -> visible tiles (z>=12)
   -> ObjectBox/memory TTL cache hit? -- yes --> render           (in-memory per page)
            | miss/stale                                  |
            +-----------------+----------------------------+
                              v
             GET /api/v1/pois?tile=<x>,<y>&category=a,b,c   (SvelteKit, hooks: csrf -> auth)
                              |  zod: x,y ints in range; categories whitelisted, <=12
                              v
        for each (tile, category): file cache lookup  (POI_CACHE_DIR/<x>/<y>/<cat>.json)
          fresh  -> serve
          stale  -> serve (status "stale") + enqueue background refresh (deduped)
          miss   -> need upstream
                              v
        inflight Map<"x,y|cat", Promise>  (single-flight; join if present)
                              v
        serial queue (one Overpass request at a time)
          backoffUntil > now? -> do not call upstream; cold keys -> "unavailable"
          build QL from category table, bbox from tile index, UA header, timeout
          200 + JSON + no `remark` runtime error -> partition elements per category,
                                                     keep only coords inside the tile,
                                                     write each (tile, cat) file (empty = valid)
          429/406/5xx/timeout/HTML/remark-error -> backoffUntil = now+30 s; nothing cached
                              v
        200 {tile, results:{cat:{status,fetchedAt,elements}}, retryAfter?}
                              v
 clients dedupe by type/id, render GeoJSON symbol layer, filter by active categories
 App only: tap -> featuresInRect(POI layer) -> look up element in Dart cache -> detail sheet
```

### Recommended Project Structure

```
web/src/lib/pois/categories.ts          # NEW shared, no svg/maplibre imports: category ids, tag filters
web/src/lib/server/pois/query.ts        # NEW buildOverpassQuery(), partitionElements()
web/src/lib/server/pois/cache.ts        # NEW file store (get/put/prune), POI_CACHE_DIR
web/src/lib/server/pois/upstream.ts     # NEW serial queue, backoff, UA, fetch + response validation
web/src/lib/server/pois/service.ts      # NEW SWR + single-flight orchestration
web/src/routes/api/v1/pois/+server.ts   # NEW route, zod, @swagger JSDoc (like map/style-sources)
web/src/lib/vendor/maplibre-layer-manager/layers.ts      # import tags from categories.ts, keep icon/bg
web/src/lib/vendor/maplibre-layer-manager/overpass-layer.ts  # call /api/v1/pois, drop QL building
app/lib/entities/poi_tile_entity.dart                    # NEW ObjectBox entity
app/lib/store/poi_cache_store.dart                       # NEW thin Store functions (like search_history_store.dart)
app/lib/provider/map_layer_selection_provider.dart       # NEW keepAlive notifier (+ .g.dart)
app/lib/provider/poi_provider.dart                       # NEW fetch/cache/online gating (+ .g.dart)
app/lib/components/map/map_overlay_layers.dart           # NEW reconcile(): overlays + POI source/layer + icons
app/lib/components/map/poi_icons.dart                    # NEW SVG -> PNG composite
app/lib/components/map/map_layer_picker.dart             # NEW picker sheet + trigger button
app/lib/components/map/poi_detail_sheet.dart             # NEW curated sheet
app/assets/svgs/pois/*.svg                               # NEW copy of web SVGs (+ pubspec asset line)
```

### Pattern 1: Shared category module so the server can build queries (Question 4)

`layers.ts` imports 28 SVGs with Vite's `?raw` and maplibre types, so the server must not import it. Split the `tags` data out:

- `web/src/lib/pois/categories.ts` exports `poiTags: Record<string, PoiTags>` (the `tags` objects moved **verbatim** from `layers.ts:213-494`) and `poiCategoryIds`.
- `layers.ts` keeps `icon` and builds `pois[name] = { icon, tags: poiTags[name] }` so the web client sees the same `pois` object.

Category ids actually offered in the UI, verbatim from `defaultMapState` (`layers.ts:503-553`): food: `"food-drinks"`, `"grocery-store"`, `bakery`; tourism: `campsite`, `attraction`, `hotel`, `hut`, `viewpoint`; amenity: `water`, `barrier`, `shelter`, `shower`, `toilets`; hiking: `"mountain-pass"`, `climbing`, `summit`; cycling: `"bicycle-rental"`, `"bicycle-shop"`, `"bicycle-parking"`; "car-motorcycle": `"garage"`, `"gas-station"`, `parking`; "public-transport": `"railway-station"`, `"subway-stop"`, `"tram-stop"`, `"bus-stop"`, `ferry`. That is **27** UI categories. `pois` also defines `picnic`, which no `defaultMapState` group lists (never toggleable on the web); skip it in the app unless the user wants it.

Tag-filter shapes in the table (verbatim examples): `shop: ['supermarket', 'convenience']` (array = OR over values), `water` is an **array of two filter objects** (`{amenity: ['drinking_water', 'water_point']}` and `{natural: 'spring', drinking_water: 'yes'}`), `'bus-stop'` is `{public_transport: ['stop_position', 'platform'], bus: 'yes'}` (AND), and `barrier` is `{ barrier: true }` (see Pitfall 11).

### Pattern 2: Query construction (Question 4)

The current client query, verbatim (`overpass-layer.ts:233`):
```
`[bbox:${bounds.getSouth()},${bounds.getWest()},${bounds.getNorth()},${bounds.getEast()}][out:json];(${this.getQueries(activeQueries)});out center;`
```
Server version: same shape plus explicit limits and a lighter output, with quoted values:

```
[out:json][timeout:25][maxsize:67108864][bbox:S,W,N,E];(nwr["amenity"="toilets"];nwr["shop"="bakery"];...);out center tags;
```
- `[timeout:..]`, `[maxsize:..]`, `[bbox:south,west,north,east]`, `[out:json]` syntax [CITED: wiki.openstreetmap.org/wiki/Overpass_API/Overpass_QL]. The wiki page does not explicitly confirm that `out center tags;` is a valid combination, only that verbosity and geometry modifiers are separate and may appear in any order. [ASSUMED: widely used; verify with one real request]. If rejected, fall back to `out center;`.
- Always quote keys and values (`["amenity"="toilets"]`): the wiki's rule is that only pure-letter literals may be unquoted, and the table has values with underscores/hyphens. Values come from the server-side table (never from the request), so quoting is correctness, not injection defence. [CITED: same page]
- `bbox` numbers: derive from integers, `S = y / 10`, `W = x / 10`, `N = (y + 1) / 10`, `E = (x + 1) / 10`, formatted `toFixed(1)`.
- **Partition results exactly:** keep an element only if its (center) lat/lon lies in `[S, N) x [W, E)`. A global `[bbox]` returns ways whose geometry intersects the box even when the center is outside, so adjacent tiles would otherwise return the same way twice. Then assign each element to every category whose filter matches (port `belongsToQuery`/`belongsToQueryItem` from `overpass-layer.ts:218-229`, reading `element.tags`).
- Response shape to clients: keep **all** tags (needed for "Show all tags"); drop elements without coordinates; element key is `type/id` (node 5 and way 5 are different objects, the web's `id`-only key collides).

### Pattern 3: Endpoint contract (the interface between server, web and app)

```
GET /api/v1/pois?tile=<x>,<y>&category=<id>[,<id>...]
  x = floor(lon * 10)  in [-1800, 1799]   (west edge = x/10)
  y = floor(lat * 10)  in [-900, 899]     (south edge = y/10)
  category: 1..12 ids from the whitelist (deduped); unknown id -> 400 invalid_params

200 application/json
{
  "tile": { "x": 1234, "y": 567 },
  "results": {
    "<category>": {
      "status": "fresh" | "stale" | "unavailable",
      "fetchedAt": 1730000000000 | null,
      "elements": [ { "id": 123, "type": "node", "lat": 47.1, "lon": 8.2, "tags": { "name": "..." } } ]
    }
  },
  "retryAfter": 30          // seconds; present when any category is "unavailable"
}
```
Rules: always HTTP 200 for a valid request (so the app's Dio interceptor, which marks online on any non-connection error, and the web client do not treat backoff as a transport failure); also set a `Retry-After` header when something is `unavailable`. `unavailable` has `elements: []` and **must not be negatively cached by clients**; `stale` is usable. An empty `fresh` list is a real result and is cached (most tile/category pairs are empty). Add `Cache-Control: private, max-age=300`. Rationale for multi-category per call: the `auth` hook in `hooks.server.ts` does `authRefresh` plus two PocketBase lookups for signed-in users on every request (`hooks.server.ts:169,181,182`), so per-(tile, category) calls from the app would multiply that cost; the server then batches all missing categories of one tile into **one** Overpass query. The "12" cap is a judgement. [ASSUMED]

Tile index choice: integer 0.1 degree grid (not slippy z/x/y) because it matches the web's existing `tileSize = 0.1` (`overpass-layer.ts:28`), keeps cache keys independent of zoom, and is trivial in Dart and JS. Both clients must iterate **integer indices** `floor(south*10)..floor(north*10)` and `floor(west*10)..floor(east*10)` inclusive. The current web loop `for (let lat = south; lat < north; lat += this.tileSize)` (`overpass-layer.ts:131-132`) steps in float increments from the unaligned viewport edge and skips the last partially visible row/column (for example south=47.09, north=47.19 visits only tile 47.0), a latent bug the migration should fix.

### Pattern 4: Serial queue, single-flight, backoff (Question 5)

The repo already has the queue shape to copy: `withNominatimQueue` (`nominatim.ts`, a `previousRequest` promise chain released in `finally`), `waitTimer`, `retryAfterMs`, a 10 s fetch timeout via `AbortController`, and `"User-Agent": \`wanderer/${version}\`` (line 95).

```ts
// web/src/lib/server/pois/upstream.ts (sketch; module-level state is correct: single process)
const BACKOFF_MS = 30_000;
let backoffUntil = 0;
let tail: Promise<unknown> = Promise.resolve();
export const backoffRemainingMs = (): number => Math.max(0, backoffUntil - Date.now());

export function enqueueUpstream<T>(job: () => Promise<T>): Promise<T> {
    const run = tail.then(job, job);
    tail = run.catch(() => undefined);   // chain never rejects
    return run;
}
// inside job: if (backoffRemainingMs() > 0) throw new BackoffError(...)
// on 429|406|502|503|504|timeout|non-JSON body|`remark` containing "error": backoffUntil = Date.now() + BACKOFF_MS
```
- **Single-flight:** `inflight: Map<"x,y|cat", Promise<Entry>>`. A request joins existing entries and creates **one** queue job for the remainder of its missing categories on that tile, registering each category key against that job's promise. Delete keys in `finally`.
- **Waiting budget:** a client request awaits its jobs for at most ~20 s (proxies commonly cut at 60 s), then answers `unavailable` for the rest while the job keeps running and fills the cache for the retry. [ASSUMED numbers]
- **Stale-while-revalidate:** fresh TTL 7 days; keep files 30 days (prune on startup and lazily); an expired-but-present entry is returned immediately as `stale` and a deduplicated background refresh is enqueued. During backoff stale entries are served and no refresh is attempted; cold keys are `unavailable`. Use global `fetch` (not `event.fetch`) in queue jobs so background work does not depend on a finished request. `hooks.server.ts` defines no `handleFetch` (only `export const handle`, line 229), so nothing is lost. [VERIFIED]
- **What counts as failure:** policy says pause 30 s on HTTP 429 or 406 [CITED: wiki.openstreetmap.org/wiki/Overpass_API]. Treating 502/503/504/timeouts the same is a conservative extension [ASSUMED]; the public instance was observed during this research answering `# 502 Gateway Timeout ... at capacity ... request queue is full` in a plain-text body, then timing out on retries [VERIFIED: curl, 2026-11-04]. Overpass reports query timeouts and out-of-memory as **HTTP 200** with a top-level `remark` such as `runtime error: Query timed out ...`, sometimes partial elements, and some failures return an HTML page with 200 [CITED: overpass.ch/guides/overpass-api-errors]. Never cache such a response.
- **Multi-process:** the image runs `node server.js` (plus `node watcher.js`, which never calls Overpass) [VERIFIED: `web/package.json` start script, `web/server.js`]; adapter-node does not cluster. In-memory queue/backoff is therefore exact. Document that running several web replicas multiplies the upstream rate; the shared disk cache still deduplicates data but not in-flight work.
- **Optional safety valve (recommended):** a rolling daily upstream-request counter (default ~5000, `0` = off) that behaves like backoff when exceeded. The public policy is ~10,000 queries/day as an upper bound and "divide by 100" for regular use [CITED: wiki.openstreetmap.org/wiki/Overpass_API]; `/api/v1/pois` is anonymous on non-private instances (see Security), so only a counter bounds a scraper walking the world grid. Default value is a judgement. [ASSUMED]

### Pattern 5: Cache storage (Question 2)

Writable storage in the shipped setup: the `web` service mounts **only** `./data/uploads:/app/uploads` (`docker-compose.yml:75-77`); the image creates `/app/uploads` and runs as root (no `USER`) [VERIFIED: `web/Dockerfile`]. There is no `web/src/lib/server` file store to copy (`region_archive.ts` proxies the Go backend; nothing writes to disk). **Do not put the cache under `UPLOAD_FOLDER`**: `watcher.js` chokidar-watches that whole folder, takes the first path segment as a Bearer token, and PUTs every added file to `/api/v1/trail/upload` (`watcher.js:6-44`); cache files would be uploaded as trails and fail.

Recommendation: new env var `POI_CACHE_DIR`.
- `web/Dockerfile`: `RUN mkdir -p uploads cache` and `ENV POI_CACHE_DIR=/app/cache`.
- `docker-compose.yml` and `docs/.../installation/docker.mdx`: add `POI_CACHE_DIR: /app/cache` and volume `./data/cache:/app/cache` (`data*/` is gitignored); update the "four volumes" sentence in `docker.mdx` ("By default, four volumes are mounted").
- Default when unset (from-source installs): `path.join(os.tmpdir(), "wanderer-poi-cache")`. If the directory is not writable, log once and run memory-only (the endpoint must still work).
- Layout `<dir>/<x>/<y>/<category>.json`, body `{"v":1,"fetchedAt":<ms>,"elements":[...]}`; atomic write (`writeFile` to `.tmp-<rand>` then `rename`); a corrupt/unparsable file is treated as a miss. Paths are built only from validated integers and whitelisted ids (no traversal). An optional bounded in-memory Map (e.g. 500 entries) sits in front of disk.
- Survives restart: a container restart keeps its writable layer even without a volume, a recreate/upgrade does not, hence the volume. Success criterion 1 ("cache hit survives a server restart") is tested by constructing a second store over the same directory.

### Pattern 6: App layer reconciliation (Questions 7, 10)

Facts from the plugin source, `maplibre_platform_interface-0.3.7/lib/src/style_controller.dart` (read this session):
- `addSource(Source)`, `addLayer(StyleLayer, {belowLayerId, aboveLayerId, atIndex})`, `updateGeoJsonSource({id, data})`, `removeLayer(id)`, `removeSource(id)`, `addImage(id, Uint8List)`, `removeImage(id)`, `getAttributionsSync()`. There is **no** visibility, paint, layout or filter setter, so "hide" means remove, "show" means add.
- Adding a layer or source whose id already exists **throws** on Android (`A Layer with the id "..." already exists in the map style.`) and iOS; removing an unknown id throws (hence the guarded `remove` in `TrailLayer.remove`, `trail_layer.dart`).
- `belowLayerId`: iOS throws `Layer "$id" does not exist.` (`maplibre_ios-0.3.7/lib/src/style_controller.dart:123-126`); Android calls `_jStyle.addLayerBelow(...)` (`maplibre_android-0.3.7/.../style_controller.dart:137`) and what it does for a missing id is **not verified** (likely append, see Open Questions).
- `RasterSource` defaults `tileSize = 512`, `maxZoom = 22`, has `attribution`; `RasterStyleLayer({id, sourceId, ...})` [VERIFIED: `raster_source.dart`, `raster_style_layer.dart`].
- Events: `MapEventCameraIdle` ("No changes to the map camera though gestures or camera transitions are in progress", `map_events.dart:~157-170`), `MapEventClick` (`.point`, `.screenPoint`), `MapEventStyleLoaded`. `MapController.featuresAtPoint(Offset, {layerIds})` and `featuresInRect(Rect, {layerIds})` return `RenderedFeature{id, properties, geometry}`; the doc warns `id` is unreliable for GeoJSON, "rely on `[properties]`". `getVisibleRegion()` returns `LngLatBounds`; `getCamera().zoom`.
- A style swap (theme toggle) drops runtime layers **and** images: `trail_layer.dart:67` ("a theme toggle rebuilds the style and drops added layers/images") and `trail_map.dart:279` ("Re-add track + arrows after every style load - setStyle drops them").

Pattern: one idempotent `MapOverlayLayers.reconcile(style, desired, {below})`, called from **every** host's `onStyleLoaded` and whenever the desired state changes. It tracks what it believes is attached; `onStyleLoaded` resets that tracking to empty first. All operations go through a single per-host Future chain (the existing `TrailLayer.add(...).ignore()` is fire-and-forget, so unserialized overlay calls can interleave with it).

Anchoring (Question 7, "below trail/route layers"). Layer ids that exist today, verbatim:
- Trail detail (`TrailMap` + `TrailLayer`): `'trail-casing'`, `'trail-route'`, `'trail-arrows'` (`trail_layer.dart` constructor defaults).
- Main map: `'clusters'`, `'cluster-count'` (`cluster_layer.dart:31,63`) and declarative layers named `maplibre-layer-$index` (`layer.dart:26` `getLayerId(int index) => 'maplibre-layer-$index'`), which are removed and re-added **at the top** on change (`layer_manager.dart`).
- Route planner: `'route-segments-routed-casing'`, `'route-segments-routed'`, `'route-segments-straight'`, `'route-segments-blocked'`, `'route-segments-hit'` (`route_segment_layer.dart:21-24` and `hitLayerId`), added lazily when segments exist.

Strategy: the host passes an ordered list of candidate anchor ids; try `addLayer(..., belowLayerId: candidate)`, catching the exception and moving to the next candidate, finally plain append. Because later-added host layers append on top anyway, the only case that needs an anchor is "overlay toggled after the host layers exist". On a fresh style load, call `reconcile` **before** the host adds its own layers when possible. The POI symbol layer is added last (on top) so it is tappable above routes; Flutter `WidgetLayer` markers draw above the native map regardless.

Waymarked Trails overlay specs, verbatim from `layers.ts:119-206`: tile URLs `'https://tile.waymarkedtrails.org/hiking/{z}/{x}/{y}.png'`, `'.../cycling/{z}/{x}/{y}.png'`, `'.../mtb/{z}/{x}/{y}.png'`, `'.../slopes/{z}/{x}/{y}.png'` (UI key `skiing`), each `tileSize: 256`, `maxzoom: 18`, attribution `&copy; <a href="https://www.waymarkedtrails.org" target="_blank">Waymarked Trails</a>`. A live tile request returned `256 x 256` PNG with `expires:` +3 h and `access-control-allow-origin: *` [VERIFIED: curl]. **Set `tileSize: 256` explicitly** (plugin default is 512).

```dart
// Source: maplibre_platform_interface 0.3.7 + trail_layer.dart usage
await style.addSource(ml.RasterSource(
  id: 'overlay-hiking',
  tiles: ['https://tile.waymarkedtrails.org/hiking/{z}/{x}/{y}.png'],
  tileSize: 256,
  maxZoom: 18,
  attribution: '<a href="https://www.waymarkedtrails.org">Waymarked Trails</a>',
));
await style.addLayer(const ml.RasterStyleLayer(id: 'overlay-hiking', sourceId: 'overlay-hiking'),
    belowLayerId: anchorOrNull);
```

### Pattern 7: POI source/layer and hit-testing (Question 7)

One `GeoJsonSource('poi')` + one `SymbolStyleLayer('poi', minZoom: 12)` with `icon-image: ['get','icon']`, `icon-allow-overlap` stepped like the web (`['step', ['zoom'], false, 14, true]`), `icon-size` tuned on device. GeoJSON properties stay minimal: `{ key: "node/123", cat: "bakery", icon: "poi-bakery" }`; keep the full tags in a Dart-side `Map<String, PoiElement>` and look up by `key` on tap, because nested objects in native feature properties are platform-dependent (the web needed `JSON.parse(feature.properties?.tags)`, `maplibre_util.ts` `createOverpassPopup`). Filtering by active category is done by rebuilding the GeoJSON string (no filter setter), pushed with `updateGeoJsonSource`. Tap: in each host's `onEvent`, for `MapEventClick` first call `controller.featuresInRect(Rect.fromCenter(center: event.screenPoint, width: 44, height: 44), layerIds: ['poi'])`; if non-empty, open the sheet and **return before** the host's existing click logic. Existing precedent for `featuresAtPoint(layerIds:)` in click handlers: `map_screen.dart` (`layerIds: const ['clusters']`) and `route_planner_screen.dart` (`'route-segments-hit'`).

Host click handlers that a POI tap must pre-empt: `map_screen.dart` background tap drops the location pin (`setState(() => _selectedLocation = event.point)`) and `route_planner_screen.dart` empty-map tap does `notifier.appendAnchor(event.point)`.

### Pattern 8: Fetch driver (Question 3 on the app side)

On `MapEventCameraIdle` (debounce ~400 ms, generation counter so a newer idle cancels an older run): if zoom < 12 or offline or no POI category selected, clear; else compute integer tiles from `getVisibleRegion()`, order by distance to the camera center, cap at **12** tiles (at MapLibre zoom 12 and lat 48, metres per logical pixel is `78271.51696 * cos(lat) / 2^zoom` ~ 12.8 m, the same constant as `trail_layer.dart`'s `TrailMarkerLayer`, so a 400x800 dp viewport spans ~5x10 km = about 2x2 tiles, a tablet up to ~4x3; the cap is a safety bound) [computed; cap value ASSUMED]. For each tile: read ObjectBox rows for the selected categories; categories with a fresh row (app TTL ~3 days, shorter than the server's 7) come from cache; request the rest in one `GET /pois` per tile. Persist `fresh` and `stale` results (stale with its server `fetchedAt` so it is re-asked later), do not persist `unavailable`. Sequential per-tile requests, not parallel.

### Anti-Patterns to Avoid

- **Accepting raw QL from clients** (the current `/api/v1/overpass/interpreter?data=`): unauthenticated on public instances (see Security). The new endpoint takes only integer tile + whitelisted ids.
- **Writing the cache under `UPLOAD_FOLDER`** (triggers the watcher).
- **Caching a 200 that contains `remark` / non-JSON.**
- **Using `event.fetch` inside background refresh jobs.**
- **Toggling by guessing layer existence** with `getLayerIds()` (marked `@visibleForTesting`).
- **Passing nested tag maps through native feature properties.**
- **Adding a Dart-side hardcoded copy of tag filters.** The app never needs the filters; only ids, colours, icon file, l10n key.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Request validation | manual `parseInt`/regex chains | `zod` (`z.object`, `.refine`) + `handleError` | `handleError` already maps `ZodError` to `400 {message:"invalid_params", detail}` (`web/src/lib/util/api_util.ts:253`) |
| Serial request queue | new concurrency lib | the promise-chain pattern in `nominatim.ts` | Proven in this repo, tested with fake timers |
| Retry-After parsing | new parser | `retryAfterMs` shape from `nominatim.ts` | handles seconds and HTTP-date |
| SVG parsing/rendering | own path parser | `flutter_svg` `vg.loadPicture(SvgStringLoader(...), null)` -> `PictureInfo.picture` | documented in the package README for rendering to a canvas/image |
| Opening-hours parsing | an `opening_hours` grammar | show the raw tag value (one line per `;` rule) | grammar is large; no vetted Dart library is in the project [ASSUMED] |
| URL opening | custom intent code | `url_launcher` (`launchUrl`) | already used (`wanderer_attribution.dart`, `trail_dropdown.dart`) |
| Bitmaps for map symbols | custom native bridge | `StyleController.addImage(id, pngBytes)` | present on both platforms |

**Key insight:** the hard part is policy-correct upstream behaviour (dedupe, serialize, back off, never cache failures), not data plumbing; keep it in one small server module with fake-timer tests.

## Runtime State Inventory

Not a rename/refactor phase. Closest "state" concerns, answered explicitly:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | New server cache directory (none exists today); new ObjectBox entity `PoiTileEntity` and a new `LocalSettingsEntity` field. ObjectBox model currently has 12 entities, `lastEntityId` `13:4676683893580730501` [VERIFIED: `app/lib/objectbox-model.json`] | Commit regenerated `objectbox-model.json` + `objectbox.g.dart` (precedent: `b15584134` added `SearchHistoryEntity` the same way). Adding an entity/property is a normal ObjectBox schema evolution for existing installs |
| Live service config | None. `OVERPASS_API_URL` stays as is | none |
| OS-registered state | None | none |
| Secrets/env vars | New optional `POI_CACHE_DIR` (and optional daily-limit var). `OVERPASS_API_URL` / `PUBLIC_OVERPASS_API_URL` unchanged | Document in `environment-configuration.md`; `PUBLIC_OVERPASS_API_URL=` in `web/Dockerfile:28` is legacy fallback (`url.ts` `PUBLIC_${key}`) |
| Build artifacts | `.g.dart` files for new providers; `app_localizations*.dart` + `untranslated_messages.json` after ARB additions | `build_runner` + `flutter gen-l10n`, commit results |

Account purge: `account_data_purge.dart` clears only account-scoped boxes/dirs (`accountScopedDirNames = ['avatars']`); POI cache and selection are public/device data and must **not** be added there (memory: regions/map data are account-agnostic).

## Common Pitfalls

### Pitfall 1: Raster tiles render at the wrong scale
**What goes wrong:** Waymarked overlay looks twice as large/blurry. **Why:** `RasterSource.tileSize` defaults to 512 but Waymarked tiles are 256 px. **Avoid:** `tileSize: 256`. **Detect:** route symbols look oversized versus the web.

### Pitfall 2: Layers vanish after a theme change
**What goes wrong:** overlays/POIs disappear after light/dark swap. **Why:** `setStyle` drops runtime layers and images (`trail_map.dart:279`). **Avoid:** reconcile from every `onStyleLoaded`, reset tracking, re-register images from cached PNG bytes. **Note:** `route_planner_screen.dart` passes the style only as `MapOptions.initStyle` (init-only per plugin source) and never calls `setStyle`, so it does not swap theme mid-session today; still call reconcile from its `_onStyleLoaded`.

### Pitfall 3: Duplicate-id exceptions and add/remove races
**Why:** Android and iOS throw on an existing id; `TrailLayer.add` is fire-and-forget. **Avoid:** guard removal in try/catch (copy `TrailLayer.remove`), serialize through one Future chain, track attached ids.

### Pitfall 4: `belowLayerId` of a missing layer
**Why:** iOS throws, Android unverified. **Avoid:** candidate list + catch + append fallback. **Verify on device** both platforms (Open Question 3).

### Pitfall 5: Attribution never updates
**What goes wrong:** Waymarked Trails / OSM credit missing after toggling. **Why:** `WandererAttribution` caches `getAttributionsSync()` keyed on the style object identity and says "attributions only ever change when the style itself is swapped" (`wanderer_attribution.dart:50`); a runtime source does not change that identity. **Avoid:** give the widget an `extra` attribution list driven by the selection provider (Waymarked Trails when any overlay is on; OpenStreetMap contributors when any POI is on). Waymarked's published terms (overlay CC BY-SA 3.0 DE, mention OSM and the site, moderate access, cache tiles, correct referrer, no mass download) were read through a search summary of the site's Help/Legal page; the page itself is JS-rendered and could not be fetched [CITED: hiking.waymarkedtrails.org/#help-about via search summary; unverified at source]. Native MapLibre sends no referrer; consider contacting the maintainer (Open Question 6).

### Pitfall 6: Main map shows no map at all when offline
`map_screen.dart:501` (`if (!isOnline)`) replaces the whole map with `WandererOfflineState` when `onlineStatusProvider` is false, so the "hide layers, keep selection" rule is only observable on trail detail and the route planner. Still implement gating in the shared provider so all three behave identically, and keep the picker button disabled when offline.

### Pitfall 7: Overpass failure disguised as success
200 + `remark: "runtime error..."`, HTML 200 pages, partial `elements`. **Avoid:** parse as JSON, reject non-object, reject any `remark` containing `error`, require `elements` array; none of those are cached and all trigger backoff.

### Pitfall 8: Stampede on a cold popular tile
Two phones and a browser ask the same tile. **Avoid:** key-level single-flight plus serial queue; test "two concurrent requests, one upstream fetch".

### Pitfall 9: Tile-edge duplicates and dropped edge tiles
Ways cross tile boundaries (see Pattern 2 partition rule); client loop skips partial tiles (Pattern 3). **Avoid:** integer index loops; client-side dedupe on `type/id` when merging tiles.

### Pitfall 10: Heavy tiles in cities
`parking`, `bus-stop`, `barrier`, `food-drinks` in a 0.1 degree city tile can return thousands of elements (several MB) and time out. **Avoid:** `[timeout:25]`, `[maxsize:...]`, read the upstream body with a size ceiling, and let a failed batch fall into backoff rather than retry in a loop. Same exposure exists today.

### Pitfall 11: `barrier` is effectively broken on the web today
`barrier: true` (`layers.ts:325`) is turned into `[barrier=true]` by `getQueryItem` (a literal-string equals; `[barrier]` would be the existence test [CITED: Overpass_QL wiki]) and `belongsToQueryItem` compares `element.tags[tag] === value`, which is never `true`. Success criterion 3 says the web layer "works unchanged"; porting `barrier` verbatim keeps it a no-op. Decision needed (Open Question 1). Do not silently "fix" to `[barrier]` (would pull every fence and kerb).

### Pitfall 12: Query cost per request on the server
Every `/api/v1/*` request from a signed-in client runs the `auth` hook (`authRefresh` plus settings and actor lookups, `hooks.server.ts:169-182`). Batch categories per tile; cap tiles per screen.

### Pitfall 13: Untrusted OSM tag values in the app sheet
`website=javascript:...`, `phone` with odd schemes. **Avoid:** only launch `http`/`https` (prepend `https://` to bare hosts), build `tel:` from digits/`+` only, render everything else as plain `Text` (never HTML). [ASSUMED, standard practice]

### Pitfall 14: Icon density differs per platform
Android decodes the PNG with `inScaled = true, inDensity = 160, inTargetDensity = devicePixelRatio * 160` (`maplibre_android-0.3.7/lib/src/style_controller.dart:245-265`), i.e. it upscales by the pixel ratio; iOS wraps the bytes in a `UIImage` at scale 1 (`maplibre_ios-0.3.7/lib/src/style_controller.dart:29-33`). Rasterize at a fixed high resolution (e.g. 96 px) and choose `icon-size` on device; do not trust arithmetic from source (memory: verify on device).

### Pitfall 15: Non-recursive Flutter asset directory
`pubspec.yaml:88-89` lists `assets/svgs/` and `assets/map/`; a subfolder needs its own line `- assets/svgs/pois/`.

### Pitfall 16: SVG variety
The 28 web SVGs include a file with `<?xml ...?>` and a `<!DOCTYPE`, `width="800px"` roots, `fill="none"` and `fill="#000000"` roots (`bike-*`, `mountain-pass`), and FA files with a comment. The web recolours by regex: `<path` -> `<path fill="white"` after stripping the `<svg>` wrapper and scaling to a 24 unit box inside a 40 unit circle (`overpass-layer.ts` `transformFontAwesomeIcon`, `loadIcons`). Port that same string transform to Dart and add a unit test that every one of the 27 composites parses under `vg.loadPicture` without throwing.

### Pitfall 17: Servers without the endpoint
Older `-app` servers return `404 {"message":"not_found"}` for unknown `/api/*` routes (`api_errors.ts`). The app must treat 404 on `/pois` as "POIs unavailable on this server" (hide POI section), not as an error toast. Overlays do not depend on the server. `kMinimumServerVersion` is `[0, 21, 2]` (`server_compatibility.dart`); do not raise it for this feature.

### Pitfall 18: Riverpod and ObjectBox conventions
No `late final` in `build()`, use `.value`; ObjectBox has no plain `flutter test` harness (`search_history_store_test.dart` header comment), so keep Store functions thin and test the pure logic (encoding, TTL decision, tile math); on-device check for Store code.

### Pitfall 19: Local web test environment is stale
`web/node_modules/@sveltejs/kit` is **3.0.0** while `web/package.json` requires `^2.70.3`; `npx vitest run` currently fails with `svelte.config.js is no longer used` [VERIFIED: ran it]. Run `npm ci` in `web/` (or in the new worktree) before server tests. I did not modify `node_modules`.

## Code Examples

### Zod validation (pattern from `waypoint/cluster/+server.ts`, errors via `handleError`)
```ts
import { z } from "zod";
import { poiCategoryIds } from "$lib/pois/categories";

const PoisQuerySchema = z.object({
    tile: z.string().regex(/^-?\d{1,4},-?\d{1,3}$/).transform((s) => s.split(",").map(Number))
        .refine(([x, y]) => x >= -1800 && x <= 1799 && y >= -900 && y <= 899),
    category: z.string().transform((s) => [...new Set(s.split(","))])
        .pipe(z.array(z.enum(poiCategoryIds as [string, ...string[]])).min(1).max(12)),
});
// GET: PoisQuerySchema.parse(Object.fromEntries(event.url.searchParams)); catch (e) { return handleError(e); }
```

### Server test skeleton (pattern from `nominatim.test.ts`)
```ts
vi.mock("$app/environment", () => ({ version: "test" }));
vi.mock("$lib/server/url", () => ({ resolveBaseUrl: () => "https://overpass.test" }));
// vi.useFakeTimers(); fetch mocked via vi.stubGlobal("fetch", ...); cache dir = fs.mkdtemp(os.tmpdir())
```

### Dart: composite icon -> PNG (flutter_svg README approach)
```dart
final info = await vg.loadPicture(SvgStringLoader(compositeSvg), null);
final rec = ui.PictureRecorder();
final canvas = ui.Canvas(rec);
canvas.scale(px / 40.0);               // composite svg viewBox is 40x40, as on the web
canvas.drawPicture(info.picture);
final image = await rec.endRecording().toImage(px, px);
final bytes = (await image.toByteData(format: ui.ImageByteFormat.png))!.buffer.asUint8List();
info.picture.dispose();
await style.addImage('poi-$category', bytes);   // re-run after every style load; cache `bytes`
```

### Dart: ObjectBox entity and store functions (pattern: `search_history_entity.dart`, `search_history_store.dart`)
```dart
@Entity()
class PoiTileEntity {
  @Id() int obxId = 0;
  @Unique(onConflict: ConflictStrategy.replace) String key;   // "x,y|category"
  @Index() int tileX; @Index() int tileY;
  String category; String payload;          // jsonEncode(elements)
  @Property(type: PropertyType.dateUtc) DateTime fetchedAtUtc;
  PoiTileEntity({required this.key, required this.tileX, required this.tileY,
    required this.category, required this.payload, required this.fetchedAtUtc});
}
```
Prune with a query on `fetchedAtUtc` older than the hard TTL (~14 days) at startup plus a row cap on write (`removeMany` of the oldest ids, as `recordSearchHistory` trims to its cap). `@Unique(onConflict: replace)` is already used in `category_entity.dart`/`waypoint_entity.dart`.

### Global selection persistence (Question 10)
Add `String mapLayersJson;` (default `''`) to `LocalSettingsEntity` (constructor named-param default like its other fields) and a small keepAlive notifier `MapLayerSelection` (state: immutable `{Set<String> overlays, Set<String> pois}`) that reads/writes the single row through `objectBoxProvider` like `LocalSettingsNotifier`. Rationale for a dedicated notifier rather than writing through `LocalSettingsNotifier`: that notifier does `ref.invalidateSelf()` on every write, which rebuilds every `ref.watch(localSettingsProvider)` consumer (theme, map style); a toggle should not do that. Device-level on purpose: survives logout like theme and recent servers (`local_settings_entity.dart` doc on `recentServersJson`). Parse defensively (malformed JSON -> empty selection).

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Client builds raw QL, server relays | Structured tile+category API, server builds QL | Cacheable key space; no open QL proxy |
| In-memory per-page web cache | Persistent server cache + SWR | Cross-user, restart-safe |
| Immediate retry on thrown error | Serial queue + 30 s policy backoff | Stays within public instance policy |

**Public Overpass facts** [CITED: wiki.openstreetmap.org/wiki/Overpass_API; dev.overpass-api.de/overpass-doc/en/preface/commons.html]: under 10,000 queries and 1 GB per day does not disturb others; divide by 100 for regular use; no parallel scripts; on 429 or 406 pause 30 s; add `User-Agent`/`Referer` identifying the app; "cache and rate-limit calls"; commercial use should use self-hosted or paid servers; 429 = rate limit, 504 = resource mismatch; default timeout 180 s, default `maxsize` 512 MiB, requests wait in a queue up to 15 s. Alternatives with different limits exist (for example an instance with a 16-parallel-queries-per-IP limit, others with API keys), which is what `OVERPASS_API_URL` is for.

**Deprecated/outdated:** the phase brief and CLAUDE.md quote maplibre 0.3.5 and ObjectBox 5.3.1; the manifests say 0.3.7 and 6.0.0-preview.5.

## User-Agent (Question 6)

Existing convention: `"User-Agent": \`wanderer/${version}\`` with `version` from `$app/environment` (`nominatim.ts:95`); `valhalla.ts` only resolves a URL and sets no header; `overpass.ts` sets none. `event.fetch` with that header works for Nominatim (shipped). Recommend a shared helper `wandererUserAgent()` used by Nominatim and Overpass: `wanderer/<version> (+<ORIGIN>)`, with `ORIGIN` from `$env/dynamic/private` (compose sets `ORIGIN: http://localhost:3000` placeholder, instances set their public URL), falling back to `wanderer/<version> (+https://github.com/open-wanderer/wanderer)` when ORIGIN is unset or a localhost URL. This lets Overpass operators reach the misbehaving instance instead of banning every wanderer install, with no personal email. [ASSUMED: string format is a recommendation]

## Question-by-Question Answers

1. **Web branch reality.** `git diff --stat feature/app origin/app-merge/4-web-app-support -- web/src/lib/server/overpass.ts web/src/routes/api/v1/overpass web/src/lib/vendor/maplibre-layer-manager web/src/lib/util/maplibre_util.ts web/src/hooks.server.ts` printed **nothing** after fetching `origin/app-merge/4-web-app-support` (head `65e23ccda`): identical. Whole-`web/` differences are only 15 files: 13 locale JSON files and the two `web/src/routes/.well-known/nodeinfo/**` routes that exist on `feature/app` and not on the stack branch. `feature/app` is at `d46414a61`. Base the server branch on **`origin/app-merge/4-web-app-support`** (new PR stacked on #1397), e.g. `git fetch origin app-merge/4-web-app-support && git worktree add ../wanderer-pois -b feat/pois-endpoint origin/app-merge/4-web-app-support`, then `npm ci` there. To consume before the stack merges: app code on `feature/app` develops against the contract with a fake `PoiRepository` in tests and a dev server run from the worktree (app Dio base URL is `.../api/v1`, `api_provider.dart`); once the server commits are final, cherry-pick them onto `feature/app` (clean, since the touched files are identical there) so `-app` images, which are built from `feature/app`, carry the endpoint. [Judgement on the cherry-pick; confirm with the user.]
2. **Persistent cache.** See Pattern 5. Recommend file store + `POI_CACHE_DIR` + compose volume; rejections above.
3. **Tile key.** Integer 0.1 degree grid `tile=<lonIdx>,<latIdx>` (Pattern 3); multi-category per call (Pattern 3 rationale); bbox derived server-side from indices; zod bounds.
4. **Query construction / response.** Patterns 1-3. The shared module also lets a vitest assert every `defaultMapState.pois` key exists in the tags module.
5. **Queue/backoff.** Pattern 4. Cold miss during backoff: HTTP 200, `status: "unavailable"`, `retryAfter`, `Retry-After` header.
6. **User-Agent.** Section above.
7. **maplibre 0.3.7 API.** Pattern 6 and 7.
8. **SVG to bitmap.** flutter_svg ^2.2.4 and vector_graphics are already dependencies; composite on a coloured circle via the web's own string transform (Pitfall 16); sources: `web/src/lib/assets/svgs/pois/*.svg` (28 files, ~34 KB total). Copy to `app/assets/svgs/pois/` (plus the pubspec line); copying is simplest and keeps the app offline-capable for icon rendering. The web's bg colours (verbatim from `layers.ts`): `Coral` (food), `DeepSkyBlue` (water/toilets/shower), `'#000000'` (shelter, gas-station, parking, garage, barrier), `Green` (attraction, viewpoint, picnic, summit, mountain-pass, climbing), `'#e6c100'` (hotel, campsite, hut), `HotPink` (bicycle-*), `DarkBlue` (railway/tram/subway/bus/ferry). CSS color names need a small Dart mapping (`Coral` = #FF7F50, `DeepSkyBlue` = #00BFFF, `Green` = #008000, `HotPink` = #FF69B4, `DarkBlue` = #00008B) [CSS named-color values from memory, ASSUMED; the SVG renderer in the web resolves names itself, so in Dart use these hex values].
9. **ObjectBox.** Entities in `app/lib/entities/*.dart`, generated `app/lib/objectbox.g.dart` and `app/lib/objectbox-model.json` are committed; command `cd app && dart run build_runner build --delete-conflicting-outputs`. Precedent for adding an entity: commit `b15584134`. TTL/pruning pattern: `search_history_store.dart` (`order(... descending)`, `findIds`, `removeMany`, `runInTransaction`). New entity/property is safe for existing installs (ObjectBox assigns new ids; previous entities keep theirs). No `flutter test` harness for a real Store.
10. **State.** Dedicated keepAlive notifier persisting a JSON string on `LocalSettingsEntity` (above). Riverpod conventions: `@Riverpod(keepAlive: true)`, `part '*.g.dart'` committed, e.g. `online_status_provider.dart` (`bool build() => true`) and `map_style_json_provider.dart`.
11. **Detail sheet and i18n.** Bottom sheet precedent: `showModalBottomSheet` in `components/trail/map_app_sheet.dart`, `missing_coverage_sheet.dart`; the route planner already hosts a docked `DraggableScrollableSheet` (`RouteAnchorSheet`), so use a modal bottom sheet (isScrollControlled) for the POI sheet. `url_launcher` is a dependency. ARB files: `app/lib/i18n/app_*.arb` template `app_en.arb` (`app/l10n.yaml`); the recent feature commit added keys to `app_en.arb` and `app_de.arb` by hand, then ran `flutter gen-l10n` (regenerates the committed `app_localizations*.dart` and `lib/i18n/untranslated_messages.json`); other locales via Crowdin (`crowdin.yml` maps `app_en.arb`). The web `en.json` already has display names for all 27 UI categories as bare keys (`"bakery": "Bakery"`, `"food-drinks": "Food & Drinks"`, `"mountain-pass": "Mountain pass"`, group names `"food"`, `"tourism"`, `"amenity"`, `"hiking"`, `"cycling"`, `"car-motorcycle"`, `"public-transport"`, `"skiing"`, `"pois"`); **missing in web `en.json`:** `MTB`, `overlays`, `picnic`, `opening-hours`, `website`, `phone`, `address`. Reuse the English wording as `poi_category_<id>` ARB keys (ids with hyphens become underscores) and add the detail-sheet labels new. Translations for 13 other locales exist in `web/src/lib/i18n/locales/*.json` and could seed ARB files, but Crowdin owns ARB translations; recommend en + de only and let Crowdin fill the rest.
12. **Waymarked attribution/policy.** Pitfall 5. App attribution today: `WandererAttribution` renders `MapLibre` plus `style.getAttributionsSync()` links (`wanderer_attribution.dart`).
13. **Anchor stretch.** `RouteAnchors` notifier in `app/lib/provider/route_anchor_provider.dart`; `void appendAnchor(Geographic point)` (line 669) pushes undo, appends, creates a straight segment from the previous last anchor and auto-routes if enabled. Minimal hook: `PoiDetailSheet(onUseAsAnchor: ...)` shown only from the route planner, which calls `ref.read(routeAnchorsProvider.notifier).appendAnchor(Geographic(lat: e.lat, lon: e.lon))`. The "namespace-based anchors" (PR #1377) concern how the web saves a drawn route, not the planner's in-memory anchors.
14. **Test infra.** See Verification section.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `out center tags;` is accepted by Overpass | Pattern 2 | Query returns error; fall back to `out center;` (heavier ways) |
| A2 | 429/406 plus 502/503/504/timeouts all warrant the 30 s backoff | Pattern 4 | Slightly longer pauses than policy requires, harmless |
| A3 | Android `addLayerBelow` with a missing id silently appends (maplibre-native `Collection::add` behaviour) | Pitfall 4 | Overlay could land above trail if it throws; try/catch fallback covers throw case |
| A4 | Per-request category cap 12, tile cap 12, wait budget 20 s, 7 d fresh / 30 d retain / app TTL 3 d, optional daily cap 5000 | Patterns 3, 4, 8 | Tuning only |
| A5 | OSM tag conventions for the curated sheet (`addr:street`, `addr:housenumber`, `addr:postcode`, `addr:city`, `opening_hours`, `website`/`contact:website`, `phone`/`contact:phone`, `wheelchair`, `ele`, `drinking_water`, `fee`) | Question 11 | Missing fields in the sheet; "Show all tags" still complete |
| A6 | Showing `opening_hours` raw instead of parsing | Don't Hand-Roll | UX less polished; a parser is a separate effort |
| A7 | CSS named-colour hex values for the Dart icon backgrounds | Question 8 | Slightly different shades |
| A8 | `wanderer/<version> (+<ORIGIN>)` User-Agent format | UA section | Cosmetic |
| A9 | Waymarked policy text as summarised by a search result (primary page JS-rendered) | Pitfall 5 | Attribution wording or referrer expectations differ |
| A10 | Cherry-picking server commits onto `feature/app` is what the user wants for `-app` images | Question 1 | Endpoint missing from `-app` images until stack merges |

## Open Questions

1. **`barrier` category.** What we know: ported verbatim it matches nothing (Pitfall 11). Unclear: whether to keep it as a no-op, define explicit values (gate, lift_gate, stile, ...), or hide it in the app. Recommendation: keep parity on the server this phase, omit `barrier` from the app picker (or list it only if the user supplies a value list), and record a follow-up.
2. **Remove `/api/v1/overpass/interpreter`?** Only `overpass-layer.ts` uses it in this repo (grep over `web/src`, `app/lib`, `db`, `docs`); external `wanderer_key` consumers are invisible to grep (see the PocketBase-surface note for the same lesson). Recommendation: remove it in the same PR after the web layer migrates (it is an arbitrary-QL open proxy on public instances), noting it in the PR description; if the user is wary, keep it behind an auth check instead.
3. **Android behaviour of `belowLayerId` with a missing id, and icon density.** Must be checked on a device (both platforms); the planner should schedule a human-verify checkpoint (user builds/installs per memory).
4. **Daily upstream cap default and env name.** Optional; user to confirm.
5. **Uncommitted edits claim.** The brief says the working tree has uncommitted user edits in `map_ui_controls.dart`, `map_screen.dart`, `route_planner_screen.dart`, `trail_detail_map_screen.dart`. At research time `git status --short` shows only `?? .planning/state.json`, and `git status --short app` is empty, so nothing was uncommitted. Re-check with `git status` at plan/execute time; plans that edit those four files plus `trail_map.dart`, `trail_collection_map.dart` and `wanderer_attribution.dart` should use targeted edits and re-read the files first.
6. **Waymarked Trails etiquette.** Native MapLibre requests carry no referrer; the policy asks for "a correct referrer" and moderate access. Consider a courtesy contact to the maintainer before release. Not blocking.
7. **Older servers.** Confirm the 404 -> hide-POIs behaviour is acceptable instead of raising `kMinimumServerVersion`.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node / npm | web tests, dev server | yes | node v22.23.3 | - |
| web `node_modules` | vitest | **stale** (kit 3.0.0 vs ^2.70.3) | - | `npm ci` in `web/` or in the new worktree |
| Flutter / Dart | app analyze/test/codegen | yes | Flutter 3.47.8 | - |
| `flutter test` | verified working | yes | ran 2 test files, all passed | - |
| ObjectBox native lib in `flutter test` | Store tests | no harness (project note) | - | thin Store layer, on-device check |
| Overpass public instance | real-data verification | flaky (probes: one `502 at capacity` body, then connection failures) | - | run vs a mocked fetch; try `OVERPASS_API_URL` pointing at another instance for manual runs |
| tile.waymarkedtrails.org | manual overlay check | yes (GET returned 256x256 PNG; HEAD returns 405) | - | - |
| Device/emulator | Android/iOS native-behaviour checks | user-driven | - | human-verify checkpoint |

**Missing with no fallback:** none. **Needs action:** `npm ci` before server tests.

## Security Domain

`security_enforcement` is not set in `.planning/config.json`, so this section applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2/V3 Authentication/Session | no new surface | reuse existing hook; route is anonymous on non-private instances exactly like the endpoint it replaces |
| V4 Access Control | yes | `isRouteProtected` (`authorization_util.ts`): with `PUBLIC_PRIVATE_INSTANCE=true` every path not in `publicRoutes` requires login, so `/api/v1/pois` is automatically login-only there; otherwise anonymous (parity with the old `interpreter` route, which also had no auth check) |
| V5 Input Validation | yes | zod: integer tile ranges, category whitelist, max 12; never accept free-form QL |
| V6 Cryptography | no | - |
| V8 Data Protection | low | POI data is public OSM data; cache holds no user data |
| V12 Files | yes | cache path built only from validated ints and whitelisted ids; atomic writes; size ceiling on upstream body |
| V13 API | yes | rate bound via global upstream queue/backoff/daily cap; consistent JSON errors via `handleError` |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Arbitrary Overpass QL through the proxy (existing `data=` pass-through) | Tampering / DoS | Remove or auth-gate it; new endpoint accepts ids only |
| Scraper walking the world grid to burn the instance's Overpass quota | DoS | serial queue, backoff, optional daily cap, stale-first serving |
| Cache path traversal | Tampering | integers + whitelist only |
| Cache poisoning via a bad upstream response | Tampering | strict JSON/`remark` validation; never cache failures |
| Malicious tag values (`javascript:` URLs) in the app sheet | Spoofing / XSS-like | scheme allow-list; plain-text rendering |
| Unbounded memory/disk growth | DoS | LRU-bounded memory map, 30-day file prune |

## Verification Commands

(`workflow.nyquist_validation` is `false` in `.planning/config.json`, so no Nyquist matrix; these are the runnable checks.)

**Web (server PR), after `cd web && npm ci`:**
- Targeted tests: `cd web && npx vitest run src/lib/server/pois src/routes/api/v1/pois` (existing patterns: `src/lib/server/nominatim.test.ts`, `src/routes/api/v1/map/style-sources/server.test.ts`; vitest `include` is `src/**/*.{test,spec}.{js,ts}`, `web/vite.config.ts`).
- Types: `cd web && npm run check` (`svelte-kit sync && svelte-check`).
- Required server tests (all with fake timers + a tmp cache dir): validation 400s; query builder (array-OR, two-object `water`, AND `bus-stop`, quoted values); partition + tile-bounds filter + dedupe; cache hit makes zero upstream calls; **two concurrent requests -> exactly one upstream fetch**; **new store instance over the same dir hits cache (restart)**; **429 -> 30 s backoff: no upstream during, stale served, cold = `unavailable`, resumes after `advanceTimersByTime(30_000)`**; 200 + `remark` error and HTML body are not cached; never more than one upstream request in flight; `User-Agent` header present.
- Web client: manual check on a real trail map with a dense city tile and an empty tile (memory: audit against real dependencies); run `npx vitest run src/lib/vendor` if a unit test is added for tile iteration.
- Docs: edit `docs/src/content/docs/run/environment-configuration.md` (Geocoding & Routing table, around lines 105-111) and `docs/src/content/docs/run/installation/docker.mdx` (compose snippet and the volumes list); do not `npm run build` the docs.

**App (feature/app):**
- `cd app && dart run build_runner build --delete-conflicting-outputs` (new entity/providers), `cd app && flutter gen-l10n` (after ARB edits), `cd app && flutter analyze`, `cd app && flutter test test/<new files>` then `flutter test` for the suite. Never `flutter build` / install.
- Pure-logic tests to write: tile math (edges, negatives, antimeridian-free range), selection JSON round trip and malformed input, TTL decision, curated field formatter and URL/phone sanitizer, composite-SVG builder parses for all 27 categories, response parsing incl. `unavailable`/`stale`, POI fetch driver with a fake repository (zoom gate, offline gate, tile cap, generation cancel). Widget test with the harness in `test/components/map/map_test_harness.dart` for the picker's disabled state when offline.
- Human-verify on device (user builds): overlays visible below trail/route and survive theme change; POI taps open the sheet; Android and iOS `belowLayerId` behaviour; icon sizing; offline toggle hides layers and keeps selection; route-planner POI tap does not add an anchor unless the sheet button is used.

## Sources

### Primary (HIGH confidence): repo files read this session
- `web/src/lib/server/overpass.ts`, `web/src/routes/api/v1/overpass/interpreter/+server.ts`, `web/src/lib/server/{nominatim,valhalla,url,http,region_archive}.ts`, `nominatim.test.ts`
- `web/src/lib/vendor/maplibre-layer-manager/{overpass-layer,layers,maplibre-layer-manager,types}.ts`, `web/src/lib/util/maplibre_util.ts` (`createOverpassPopup`)
- `web/src/hooks.server.ts`, `web/src/lib/util/{authorization_util,api_util}.ts`, `web/src/lib/server/api_errors.ts`, `web/package.json`, `web/Dockerfile`, `web/server.js`, `web/watcher.js`, `docker-compose.yml`, docs `environment-configuration.md`, `docker.mdx`, `from-source.mdx`
- `app/pubspec.yaml`, `app/lib/objectbox-model.json`, `app/lib/entities/*`, `app/lib/store/search_history_store.dart`, `app/lib/provider/{local_settings,online_status,api,map_style_json,map_style_sources,route_anchor}_provider.dart`, `app/lib/components/{map,base}/*`, `app/lib/routes/{map,trail_detail_map,route_planner}_screen.dart`, `app/l10n.yaml`, `crowdin.yml`
- Plugin source in the pub cache: `maplibre_platform_interface-0.3.7`, `maplibre_android-0.3.7`, `maplibre_ios-0.3.7`, `flutter_svg-2.2.4` README
- Git: `git diff` between `feature/app` and `origin/app-merge/4-web-app-support`; commits `b15584134`, `65e23ccda`

### Secondary (MEDIUM confidence)
- wiki.openstreetmap.org/wiki/Overpass_API (usage policy, fetched this session)
- dev.overpass-api.de/overpass-doc/en/preface/commons.html (429/504, slots, timeouts)
- wiki.openstreetmap.org/wiki/Overpass_API/Overpass_QL (first ~100k of ~149k characters read)
- overpass.ch/guides/overpass-api-errors (200-with-remark behaviour, via search summary)
- Live probes 2026-11-04: Waymarked tile (256 px PNG, `expires` +3 h), Overpass `/api/status` (one `502 at capacity` body, then no connection)

### Tertiary (LOW confidence, flagged)
- Waymarked Trails Help/Legal text via a search-result summary (primary page is JS-rendered)
- maplibre-native `addLayerBelow` missing-id behaviour (training knowledge)
- OSM tag key conventions for the curated sheet; CSS colour hex values

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH, no new packages; versions read from manifests (and they differ from CLAUDE.md).
- Architecture (server): HIGH for repo facts and queue patterns; MEDIUM for tuning constants.
- Architecture (app/native): MEDIUM, three behaviours unverifiable without a device.
- Pitfalls: HIGH for items traced to source lines; MEDIUM for platform-density and Android anchoring.

**Research date:** 2026-11-04
**Valid until:** ~30 days (plugin versions and Overpass policy are stable; re-check if `maplibre` or ObjectBox are bumped)
