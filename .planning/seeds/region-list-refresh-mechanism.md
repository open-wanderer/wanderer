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
