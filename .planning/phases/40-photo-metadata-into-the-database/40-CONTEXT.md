# Phase 40: Photo Metadata Into the Database - Context

**Gathered:** 2026-10-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Every content photo stored on this instance gets its GPS position, GPS altitude and capture time
extracted into the database — at upload for new photos, by a background backfill for existing ones —
readable only by people who can edit the record. The "use photo location" button works again for
already-uploaded photos, and all EXIF parsing moves from the browser (`exif-js`) to one Go reader
exposed through a parse-only API.

Stack layer 1 of the image pipeline overhaul (`feat/images-metadata`, trunk `dev`; web + db code,
not app). Stripping metadata from served files is Phase 41; locking originals is Phase 43. This phase
must not change what files are served.

</domain>

<decisions>
## Implementation Decisions

### Which photos
- **D-01:** Extract metadata for `trails.photos`, `waypoints.photos` and `summit_logs.photos` only.
  `users.avatar` and `lists.avatar` are out of scope (no useful location; a stored avatar location is
  a liability).
- **D-02:** Photos on federated records that reference remote URLs are skipped — only files stored on
  this instance get metadata. Never fetch remote originals to read EXIF.
- **D-03:** Store latitude, longitude, GPS altitude and capture time. Nothing else (no camera, lens,
  exposure). Originals stay on disk (owner-only from Phase 43), so further fields remain recoverable.

### Who sees location
- **D-04:** Stored photo metadata is readable only by the record owner and users with edit access via
  a share; not by viewers, followers or the public, regardless of the record's own visibility.
  — **Reversibility:** costly — once exposed to wider audiences the positions cannot be retracted from
  clients that fetched them; widening later is easy, narrowing is not.
- **D-05:** Photo metadata never leaves the instance in ActivityPub objects (no Create/Update payload
  carries it), consistent with D-02 for incoming records.
- **D-06:** No user opt-out of extraction in this phase (data is editor-only and never federated).

### Existing photos
- **D-07:** Automatic low-priority background backfill after upgrade: reads only each stored
  original's EXIF header (no full decode), resumable and idempotent across restarts, bounded so it
  cannot starve a small home server. It must be complete before Phase 41 strips anything — Phase 41
  plans need a way to know that.
- **D-08:** Backfill observability is log lines only (start, periodic counts, finish) in the
  PocketBase log. No admin setting, UI or CLI command.

### Photo location button and EXIF parsing
- **D-09:** New parse-only endpoint: a SvelteKit route under `web/src/routes/api/v1/` proxying a new
  PocketBase route (registered in `registerRoutes` in `db/main.go`, like `/waypoint/cluster`) that runs
  the Go EXIF reader — the same reader the upload hook and backfill use — and returns latitude,
  longitude, altitude and capture time. It parses posted bytes only and stores nothing.
- **D-10:** The endpoint requires an authenticated user. The browser sends only the first 256 KB of
  each file; the server rejects larger bodies.
- **D-11:** For already-uploaded photos the waypoint modal takes coordinates from the stored metadata
  (editor access per D-04); for freshly picked files it calls the D-09 endpoint.
- **D-12:** Remove `exif-js` everywhere: the waypoint modal (`getCoordinatesFromPhoto`), the trail
  editor's "create waypoints from photos" flow (`handleWaypointPhotoSelection`), and the vendored
  `web/src/lib/vendor/exif-js/`. Existing UX stays: the `no-gps-data-in-image` toast when a photo has
  no position.

### Claude's Discretion
- How capture time is represented (user said "you decide"; the recommended direction was keeping the
  camera's wall-clock time plus the UTC offset when EXIF provides one, rather than guessing a zone).
- Storage shape (fields on the record vs. a separate collection keyed by record + filename), provided
  D-04's access rule is enforceable and metadata follows photo add/replace/delete.
- Go EXIF library (spikes used `github.com/rwcarlsen/goexif`), backfill pacing, endpoint path names.

### Folded Todos
- **Fix "use photo location" on already-uploaded photos** (`.planning/todos/pending/2026-10-06-photo-location-reads-stripped-thumb.md`):
  since #1268 the waypoint modal passes a 600x0 thumbnail URL to `exif-js`, and thumbnails carry no
  EXIF. Folded instead of shipping a separate layer-0 fix: D-11/D-12 replace that path entirely.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Image pipeline overhaul
- `.planning/notes/image-pipeline-overhaul.md` — why, locked decisions (home-server target, private originals, strip all EXIF), stack layout (this phase = layer 1)
- `.planning/ROADMAP.md` § "Image Pipeline Overhaul (Phases 40–44)" — phase goals and dependencies

### Spike findings
- `.planning/spikes/MANIFEST.md` — requirements carried from spikes (CGO-free, `recover()` around parsing, no `image.RegisterFormat` side effects)
- `.claude/skills/spike-findings-wanderer/references/image-variant-pipeline.md` — EXIF orientation/GPS/time extraction with `goexif`, decode-via-stdlib rule, never link `gen2brain/jpegli`
- `.planning/spikes/001-image-encoding-harness/main.go` — `exifSummary`/`exifOrientation`: working goexif extraction of orientation, `LatLong()`, `DateTime()`

### Folded todo
- `.planning/todos/pending/2026-10-06-photo-location-reads-stripped-thumb.md` — the regression this phase fixes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `db/routes/waypoint_cluster.go` + `web/src/routes/api/v1/waypoint/cluster/+server.ts`: the exact
  PocketBase-route + SvelteKit-proxy pair to copy for the D-09 endpoint (zod-validated body,
  `event.locals.pb.send`, `handleError`).
- `web/src/lib/components/photo_card.svelte` / `trail/photo_picker.svelte`: already carry
  `showExifControls` / `onexif` plumbing the button uses.

### Established Patterns
- Custom PocketBase routes are registered in `registerRoutes` (`db/main.go`); record hooks live in
  `db/hooks/{trails,waypoint,summit_logs}.go` and are bound in `setupEventHandlers`.
- Uploads reach PocketBase through SvelteKit form/file routes:
  `web/src/routes/api/v1/{trail,summit-log}/form/…`, `{trail,summit-log,waypoint}/[id]/file/+server.ts`.
  Metadata extraction belongs in Go hooks so every upload path (web, app, API tokens) is covered.
- Go tests that run migrations need `GOTOOLCHAIN=go1.26.1` (Go 1.27 breaks PocketBase tests).

### Integration Points
- Waypoint modal: `web/src/lib/components/waypoint/waypoint_modal.svelte` (`getCoordinatesFromPhoto`).
- Trail editor: `web/src/routes/trail/edit/[id]/+page.svelte` (`handleWaypointPhotoSelection`, feeds
  `/api/v1/waypoint/cluster`).
- ActivityPub serialization of trails/waypoints/summit logs in `db/federation/` must not pick up the
  new metadata (D-05).
- The Flutter app reads EXIF locally (`app/lib/util/exif.dart`, on `feature/app` only) and is not
  touched by this phase.

</code_context>

<specifics>
## Specific Ideas

- One EXIF implementation for the whole system: upload hook, backfill and the parse-only endpoint all
  call the same Go reader.
- 256 KB header slice is enough for camera JPEGs with large EXIF/IRB/ICC/XMP blocks (the spike's 24 MP
  fixture has its SOF at byte 39 180).

</specifics>

<deferred>
## Deferred Ideas

- Per-user privacy setting "don't keep photo locations" (skip extraction for a user's uploads).
- Using stored positions/altitude for new features (photos on the map, photos on the elevation
  profile, matching against summit logs) — each its own phase.
- Camera/lens/exposure metadata.
- Flutter app switching from its local EXIF reader to the D-09 endpoint (Phase 44 or later).

</deferred>

---

*Phase: 40-photo-metadata-into-the-database*
*Context gathered: 2026-10-06*
