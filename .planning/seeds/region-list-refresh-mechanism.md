---
title: Automate refreshing the seeded region catalog
trigger_condition: When admins report the seeded region list is stale, or when the upstream provider extract tree changes often enough that patch-by-migration becomes a burden
planted_date: 2026-07-24
---

## Idea

The streamlined region-definition design (`.planning/notes/streamlined-region-definition.md`) seeds the `regions` catalog as a **static snapshot** via migration, and treats updating the "official" list as a deliberate patch/release. This was chosen for simplicity and to keep the milestone scoped.

If that snapshot goes stale in practice — new provider regions appear, boundaries change — revisit whether the backend should **periodically fetch/refresh** the catalog from the provider index instead of shipping a frozen snapshot.

## Why deferred

- A static snapshot is good enough for launch; refresh adds a network dependency, staleness/versioning logic, and a merge story (what happens to an `enabled` region that disappears upstream?).
- This is the natural pull-forward of the same "remote/updatable manifest" instinct (REGN-F04) that has already been deferred once — cheap to do later, not worth doing now.

## When it triggers, decide

- Fetch cadence and trigger (cron vs. on-demand).
- Merge semantics: preserve `enabled` flags across refreshes; handle removed/renamed/re-parented regions without silently disabling an admin's active regions.
- Whether refresh is opt-in per instance.

## Manual refresh procedure (until this is automated) — added 2026-07-24

The provider source was settled as CoMaps (`data/hierarchy.txt` + `data/borders/*.poly`) — see `.planning/notes/streamlined-region-definition.md` for full detail. Until this seed triggers and the above is designed, refreshing the seeded catalog is a **manual, maintainer-run** flow:

1. Re-run the `db/commands/seed_regions.go` custom command (Cobra + `*pocketbase.PocketBase`, same pattern as `db/commands/dedup.go`) against a fresh local copy of CoMaps' `hierarchy.txt`/`.poly` files.
2. The command re-parses the hierarchy, re-derives `path`/`depth`/`sort_order`/`bbox` per leaf, and writes a new committed JSON seed file.
3. Review the diff against the previously committed seed file — this is where "merge semantics" (a region renamed/split/removed upstream) gets caught by a human, not automated logic.
4. Commit the updated seed file and ship a new PocketBase migration bulk-inserting/upserting from it, same as any other schema change — a normal release, not a live-database side effect.

This procedure never touches a running production database automatically — exactly the deliberate-patch model this seed's "why deferred" section describes, just made concrete now that the provider and tool shape are both known.

## Step 4 is harder than "bulk-insert again" — added 2026-07-26

Confirmed while investigating a separate performance fix (splitting `regions.polygon` into its own `region_polygons` collection, migration `1785092688`): the *initial* seeding migration (`1785000000_create_regions_collection.go`) is guarded by `if count, _ := app.CountRecords("regions"); count > 0 { return nil }`. On every real instance (already seeded after first boot), simply regenerating `regions_seed.json.gz` and shipping it does **nothing** — that migration no-ops. Step 4 above ("ship a new migration bulk-inserting/upserting from it") is doing a lot of work in one sentence; a real refresh migration cannot just replay `1785000000`'s bulk-insert logic. It needs to **reconcile**, keyed by `path` (the provably-unique key used throughout this catalog):

- **Existing leaf, still in the new seed** → update hierarchy/name/bbox, but **preserve `enabled`** (an admin's toggle must survive a refresh) and preserve its `region_archives` row (build state) and `region_polygons` row (now a 4th thing to reconcile, alongside `regions`/`region_archives` — upsert its polygon rather than re-insert).
- **New leaf, not previously seen** → insert fresh, `enabled=false` (CATALOG-03 default).
- **Leaf that disappeared upstream** (CoMaps renamed/merged/split it) → no policy decided yet: leave it orphaned (same shape as the `munich` test-archive orphan Phase 29 already left in place) or explicitly prune it plus its `region_archives`/`region_polygons` rows. Still open — matches "handle removed/renamed/re-parented regions" above, just now concretely a 3-collection reconciliation problem instead of a 2-collection one.
