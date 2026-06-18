---
quick_id: 260618-ola
slug: move-profile-trail-filtering-server-side
description: Move profile trail filtering server-side — update API route to accept and merge filter/sort, update Flutter provider to send filter state
date: 2026-06-18
status: complete
commits:
  - d84134c2
---

# Quick Task 260618-ola — Summary

Single commit d84134c2 touching 4 files.

app/lib/models/trail.dart — added TrailFilter.toProfileFilterText() building a Meilisearch filter string from the filter model, excluding author, visibility, and geo (those are enforced server-side).

app/lib/provider/profile/profile_trails_provider.dart — build() now watches trailFilterProvider('profile_trail_$handle') so the provider rebuilds and re-fetches when filter changes. _fetchPage() calls filter.toProfileFilterText() and sends it as options.filter plus a sort param.

web/src/routes/api/v1/profile/[handle]/trails/+server.ts — local-actor branch AND-merges client-supplied options.filter with author = ${actor.id} using a Meilisearch array filter.

app/lib/routes/profile_trail_screen.dart — removed applyProfileTrailFilter() and trail_filter_provider import. Screen only watches profileTrailsProvider; filtering/sorting happens in the provider and API.
