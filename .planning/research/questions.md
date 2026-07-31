# Research Questions

## Region catalog source for seeded `regions` table (added 2026-07-24) — RESOLVED 2026-07-24

Blocked the seed migration for the streamlined region-definition feature (see `.planning/notes/streamlined-region-definition.md`, "Provider source confirmed" section for full detail). Answered via `/gsd-explore` session:

- **Which provider index to snapshot?** **CoMaps** ([codeberg.org/comaps/comaps](https://codeberg.org/comaps/comaps)) — chosen over OsmAnd/Geofabrik.
- **Does the index publish bbox/polygon, parent/child, and size?** `data/countries.txt` (JSON) publishes parent/child (via a `g` children array) and size (`s`/`sha1_base64`, leaf nodes only) — but sizes are for CoMaps' own `.mwm` format, not directly usable for our PMTiles archive sizes. `data/hierarchy.txt` publishes display names/ISO codes. `data/borders/*.poly` publishes canonical **polygon** boundaries (leaf nodes only, Osmosis `.poly` format, not GeoJSON — needs conversion) — richer than a bare bbox, and directly usable with `pmtiles extract --region`.
- **Licensing:** ODbL (OpenStreetMap-derived), per CoMaps' own `data/copyright.html`. Redistributable as a derivative dataset under the same existing attribution mechanism Wanderer already uses for its other OSM-derived tile data — not a new compliance category, but must stay attributed/share-alike.
- **Granularity:** Country-and-below where CoMaps' extract tree defines sub-divisions (e.g. Germany → Bavaria, Thuringia, ...); country-level where it doesn't (e.g. Luxembourg is itself the leaf). Matches the "cover this place" mental model.
- **Format for seeding:** No single ready-to-seed file — combine `countries.txt` (structure + leaf/group marker) + `hierarchy.txt` (names) + `.poly` files (leaf geometry) via a one-time transform tool. See `.planning/notes/streamlined-region-definition.md`'s "Seeding tool" section for the full `db/commands/seed_regions.go` → committed JSON → migration design.

---

## Offline recording & deferred upload — open decisions (added 2026-07-31)

Surfaced during a `/gsd-explore` session and deliberately left unresolved; they belong in Phase 33's
`discuss-phase`. Full context: `.planning/notes/offline-recording-deferred-upload-design.md`.

### 1. Photo durability for an undrained recording

`image_picker` returns paths into an OS-purgeable cache directory. A recorded trail may sit
undrained for days (a multi-day hike with no signal), so its referenced photo files can vanish
before upload. Does the local-first save need to copy picked photos into an app-owned directory
first, and if so, who owns cleanup after a successful drain — and after the user deletes an
unsynced recording?

Komoot explicitly flags photos as the slow part of tour upload, so this is a real-world pain point,
not a theoretical one.

### 2. Partial-failure semantics for the upload sequence

Uploading a trail is not one request. `TrailSave.createTrail`
(`app/lib/provider/trail/trail_save_provider.dart:46`) runs `PUT /tag` × N → `PUT /trail/form`
(multipart, with photo files) → `PUT /waypoint` × N. If the connection drops after the trail is
created but before all waypoints land — or the auth token expires mid-drain — what happens?

Three candidates:
- **Resume from step** — the entry records which steps succeeded (server trail id, uploaded
  waypoint ids) and the next drain continues rather than re-creating a duplicate trail. Most
  robust, most state.
- **All-or-nothing** — delete the partially-created server trail and retry the whole entry. Simpler
  to reason about, but the delete can itself fail and orphan a trail.
- **Best-effort + surface** — mirror the existing `hadWaypointFailures` precedent already in
  `TrailSaveResult`: upload what you can, mark the entry "uploaded with problems", let the user fix
  it in the normal edit screen. Least machinery, and has precedent in this codebase.

### 3. Account scoping and the logout-with-pending UX

`.planning`-recorded rule: scope by account, never purge user data. So each locally-recorded trail
must carry the account it was recorded under, the drain must filter to the active account, and
logout must leave entries on disk untouched.

Open: does logging out with undrained recordings warrant an explicit confirmation dialog? Komoot's
single most common "my tour is missing" cause is being signed into a different account, which
argues for surfacing it — but it adds a blocking dialog to a routine action.

### 4. (Lower stakes) Where does the sync badge live?

Both incumbents show per-item inline sync state in the normal trail list rather than a separate
"Pending" screen. Confirm the trail card / trail list item can carry that state without
overloading the existing `isOffline` badge, which means something different (downloaded for
offline use, not awaiting upload).
