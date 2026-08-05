# Quick Task 260805-h1e: Profile trail map screen - Context

**Gathered:** 2026-08-05
**Status:** Ready for planning

<domain>
## Task Boundary

Add a `profile_trail_map_screen` to the Flutter app (`app/lib`), reachable via an action
button in `profile_trail_screen.dart`. It shows one profile's trails on a fullscreen map.

Contents:
1. Fullscreen map, same construction as `map_screen.dart`. Initial camera fit = bounding
   box of that profile's trails, via a new provider hitting `/api/v1/trail/bounding-box`.
2. Clustered search results scoped to that profile's author only. Result cap stays at
   100 (`hitsPerPage`) — deliberately unchanged, easy to revisit later.
3. Tapping a trail draws its polyline, same as `map_screen.dart`.
4. Transparent AppBar: back button + location search bar (same as `route_planner_screen.dart`).
5. `TrailQuickFilterBar` beneath the AppBar.
6. Same `DraggableScrollableSheet` as `map_screen.dart`.
7. Same "search this area" affordance as `map_screen.dart`.
8. Same behaviour: tapping a trail hides the sheet and shows a `TrailListItem`.

Out of scope: changing the 100-result cap; raising Meilisearch `maxTotalHits`; any
line/heatmap tile pipeline; web or Go changes beyond the one endpoint below.

</domain>

<decisions>
## Implementation Decisions

### Initial camera fit — source of the bounding box

**Decision:** Extend `web/src/routes/api/v1/trail/bounding-box/+server.ts` with an
**optional `handle` query param**. When present, resolve handle → actor exactly as
`web/src/routes/api/v1/profile/[handle]/trails/+server.ts` already does, and AND
`author = {actor.id}` into all four `multiSearch` queries. When absent, behaviour is
unchanged.

**Why this and not the alternatives** (all three were investigated and rejected):

- The endpoint today takes **no parameters at all**. It calls
  `withTrailPreferenceMeiliFilter(event, undefined)`, which contributes *only*
  hidden-category exclusions — no author clause, no visibility clause. Despite its
  swagger description ("user's trails"), it returns the bbox of everything the caller
  may see.
- The Meilisearch **tenant token** (`db/routes/search_token.go:13-33`) scopes `trails`
  to `public = true OR author = {me} OR shares = {me}`. So the current endpoint's result
  spans *every public trail on the instance* — near-world on any populated instance, and
  useless as a profile fit.
- **`trails_bounding_box` (PocketBase view) was considered and rejected.** It is NOT
  deprecated — migration `1778583800_persist_trail_bounds.go` recently upgraded it from
  start-point bounds to true per-trail bounds. It is per-actor (row `id` = actor id),
  includes shared trails, and is a single indexed read. But: (a) its `viewRule` is
  `@request.auth.id = user`, so only your own row is readable; and (b) the view query has
  **no public/private predicate** — it aggregates private trails too. Relaxing the rule
  to read another actor's row would leak the existence of unpublished trails and would
  bypass the tenant-token guarantee entirely, since that path never touches Meilisearch.
  It is currently referenced only in the `Collection` enum (`web/src/lib/util/api_util.ts:44`)
  and has no call sites.

**Security note (load-bearing):** because every read goes through the tenant token,
adding `author = X` is purely a *narrowing* concern, never an authorization one. This
screen cannot leak another user's private trails even if the client-side filter is wrong.
On your **own** profile, private trails DO appear — correct, matches the list screen, and
governed by the existing public/private toggles in the shared filter (below).

### Provider scoping

**Decision:** Convert **both** `mapTrailSearchProvider` and `mapClusterSearchProvider`
to `.family`, keyed so that `map_screen.dart` passes the global/null key and the profile
map passes the profile's actor id.

**Why:** clustering is not client-side. `mapClusterSearchProvider` hits a separate
`POST /search/trails/cluster` endpoint and feeds the native circle/count layers;
`mapTrailSearchProvider` separately powers the sheet list and the tapped-trail metadata
lookup (the cluster endpoint's `attributesToRetrieve` is only id/_geo/bounding_box_diagonal).
Both are currently `@Riverpod(keepAlive: true)` **singletons** with no family key, and both
hardwire `trailFilterProvider('map')`. Scoping one without the other yields author-scoped
pins with unscoped sheet contents, or vice versa.

Parallel duplicate providers were rejected — they would drift on the next search change.
Cost accepted: touches `map_screen.dart` call sites and both `.g.dart` regenerations; the
family key must also reach the two internal `ref.listen(trailFilterProvider(...))` calls
inside each provider's `build()`.

### Camera state

**Decision:** The profile map keeps **local camera state**. It never reads or writes
`mapCameraProvider`.

**Why:** `mapCameraProvider` is a global `keepAlive` singleton with no key, and
`map_screen.dart` writes to it on every `MapEventCameraIdle`. Sharing it would mean
opening a profile map silently relocates the user's main map, and the main map's saved
camera would fight the bbox fit on open. Consequence, accepted: the profile map opens at
the bbox fit every time and does not remember position between visits.

### Filter identity

**Decision:** The profile map's `TrailQuickFilterBar` shares the list screen's filter id,
`profile_trail_{handle}`.

**Why:** list and map are two views of one filtered set — a filter set in the list is
already applied when the map opens. Accepted consequence: the map's filter bar mutates
the list's filter state, so popping back shows a correspondingly changed list. Judged
correct rather than surprising.

Note this is a *third* filter id in play: `map_screen.dart` uses `'map'`, and both
scoped providers currently hardwire `trailFilterProvider('map')` in `build()` — so the
filter id must become part of (or travel with) the family key, not just the author id.

### Federated (non-local) actors

**Decision:** When the resolved actor is non-local, **proxy the bounding-box request to
the remote instance**. If that request fails for any reason — non-200, timeout, the
remote not supporting the `handle` param, malformed payload — fall back to the **world
view** (the map's existing default camera), not to a locally-computed bbox.

**Stated tradeoff, accepted by the user:** the proxy only succeeds against remote
instances running this same new param, so on a mixed-version federation the fallback is
the common path rather than the exception; it also adds a cross-instance network hop on
screen open. The planner must therefore treat the fallback as a first-class path
(bounded timeout, no user-visible error, silent degrade to world view) rather than an
edge case.

Note the clustered search itself is unaffected: remote trails are federated into this
instance's Meilisearch index, so `author = {actor.id}` returns pins normally once the
user pans or hits "search this area". Only the initial fit degrades.

### Revisions from research (supersede assumptions above)

- **Do NOT set `TrailFilter.author`** to carry the author clause. It would alter the
  visibility branch at `app/lib/models/trail.dart:323-329`, on a filter object shared
  with the list screen. Append `author = {id}` inside each provider's `_executeSearch`
  from the family key instead. The two endpoints differ in shape: `/search/trails` takes
  an array (`filter`), `/search/trails/cluster` takes a string (`filterText`).
- **`keepAlive` must stay** on both providers. `/map` sits in a plain `ShellRoute`
  (`router_provider.dart:142`), so `MapScreen` unmounts on every tab switch and depends
  on it. The profile map instead invalidates its own two family instances in `dispose()`,
  which also satisfies the locked "opens at the bbox fit every time" decision.
- **Family key shape:** `build({String? authorId, required String filterId})` — named
  params, no defaults, so a transposed key is a compile error.
- **`fitBounds` exists** on `ml.MapController` (two in-tree precedents); no center+zoom
  math needed. Never pass `Duration.zero` (the Android binding throws). Guard zero-extent
  bboxes per `route_planner_screen.dart:240-243`.
- **`TrailBoundingBox` (app/lib/models/trail.dart) is unusable as written** — camelCase
  fields against a snake_case payload, no `has_trails`, no `fromJson` factory. It has
  zero call sites in `app/lib`, so it can be corrected freely.
- **Known compile break:** `test/provider/trail/map_search_deletion_test.dart` (11 call
  sites). There is no test covering `map_screen.dart`, so its ~15 rewritten call sites
  have no CI gate — the plan should account for that gap.
- **Screen strategy:** copy `map_screen.dart` rather than extract a shared widget,
  pulling out only the two pieces that would genuinely drift (the unclustered-marker loop
  and the sheet opacity/padding math). ~40% of the layout differs because
  `/profile/:handle` is outside the bottom-nav shell, making every
  `kBottomNavigationBarHeight` offset wrong there.

### Claude's Discretion

- **handle → actorId resolution on the Dart side.** `ProfileTrailScreen` is keyed by
  *handle*; the Meilisearch `author` field is the **actor record id**
  (`db/util/meilisearch.go:70`), and `TrailFilter.author` is an `ActorSearchResult`
  requiring `id`. Planner to pick the resolution path (existing profile provider vs.
  passing the actor id through the route).
- **Route path and navigation entry.** Placement of the action button in
  `profile_trail_screen.dart`'s AppBar, and the go_router path under `/profile/:handle`.
- **Own-profile local/unsynced trails.** `profile_trails_provider.dart` merges unsynced
  local trails and renders an offline banner; the map path has no such merge. Default
  assumption: the map shows network results only and reuses `map_screen.dart`'s existing
  offline sheet state. Planner may confirm or revise.
- Exact family key shape (record of authorId + filterId vs. a small value class).

</decisions>

<specifics>
## Specific Ideas

Components to mirror, named by the user:
- `app/lib/routes/map_screen.dart` — map construction, `DraggableScrollableSheet`,
  "search this area", trail-tap → hide sheet + show `TrailListItem`, polyline draw.
- `app/lib/routes/route_planner_screen.dart` — the location search bar in the AppBar.
- `app/lib/components/trail/trail_quick_filter_bar.dart` — filter bar.
- `app/lib/components/trail/trail_list_item.dart` — selected-trail card.
- `app/lib/routes/list_detail_map_screen.dart` — an existing smaller precedent for a
  scoped map screen, worth reading before writing a new one.

Confirmed facts the planner should not re-derive:
- Meilisearch `author` == actor record id, and is filterable
  (`db/migrations/1742167033_init_meilisearch.go:32`).
- `TrailFilter.toFilterText()` already emits `author = {id}` when `author` is set and
  `actor` is non-null (`app/lib/models/trail.dart:275-279`).
- Server precedent for the author clause: `author = ${actor.id}` in
  `web/src/routes/api/v1/profile/[handle]/trails/+server.ts:63`.

</specifics>

<canonical_refs>
## Canonical References

- `web/src/routes/api/v1/trail/bounding-box/+server.ts` — endpoint to extend.
- `web/src/routes/api/v1/profile/[handle]/trails/+server.ts` — handle→actor resolution
  pattern to copy.
- `db/routes/search_token.go` — tenant token search rules; the permission backstop.
- `db/migrations/1778583800_persist_trail_bounds.go` — `trails_bounding_box` view
  (considered, rejected; see above).

</canonical_refs>
