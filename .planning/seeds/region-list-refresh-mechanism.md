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
