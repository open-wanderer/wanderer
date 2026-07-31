---
created: 2026-07-31T00:00:00.000Z
title: Fix the two offline gaps in trail_create_screen (blank map, throwing tag autocomplete)
area: app
resolves_phase: 35
files:
  - app/lib/routes/trail_create_screen.dart
  - app/lib/provider/trail/tag_provider.dart
  - app/lib/components/base/trail_map.dart
---

## Problem

Two live offline bugs in the trail create/edit screen. They are prerequisites for Phase 33
(offline recording), but they are **not** speculative — they affect anyone editing a downloaded
trail while offline today.

### 1. The map renders blank

`app/lib/routes/trail_create_screen.dart:611` mounts `TrailMap` with `offline: trail.isOffline`.
But `isOffline` (`app/lib/models/trail.dart:100`) is the *downloaded-trail* flag, not a
connectivity signal. A trail that is being edited while the device happens to be offline — and,
once Phase 33 lands, a freshly recorded draft — has `isOffline == false`, so `TrailMap` selects
the online style (`trail_map.dart:105`) and the basemap never loads.

### 2. Tag autocomplete throws

`app/lib/provider/trail/tag_provider.dart`'s `searchByName` issues `GET /tag?filter=name~'...'`
with no ObjectBox cache and no error handling, unlike `categoryProvider`
(`app/lib/provider/trail/category_provider.dart:44`), which falls back to its cache. Offline the
Dio exception propagates into the autocomplete widget.

## Solution

1. Change the `offline:` argument at `trail_create_screen.dart:611` to also honour connectivity —
   `trail.isOffline || !ref.watch(onlineStatusProvider)`. Audit the other `TrailMap` mount sites
   for the same pattern and fix them consistently.
2. Wrap `searchByName`'s request so a connection failure yields an empty list instead of throwing.
   Deliberately **not** adding a tag cache — the decision (see the design note) is that offline
   autocomplete shows nothing and the user types free-form tags, which `_resolveTags`
   (`app/lib/provider/trail/trail_save_provider.dart:34`) creates at save/upload time.
   Distinguish a genuine connection failure from a real error using the existing
   `isConnectionFailure` helper in `app/lib/provider/online_status_provider.dart`.

## Verification

- In airplane mode with a downloaded trail open in the create/edit screen, the basemap renders
  from the offline region proxy rather than showing blank.
- In airplane mode, typing in the tag field shows no suggestions and produces no error toast or
  thrown exception; a free-form tag can still be entered and survives to the save payload.
- Online behaviour for both is unchanged.

## Context

See `.planning/notes/offline-recording-deferred-upload-design.md` ("`trail_create_screen` runs
offline with two targeted fixes") for why the tag cache was rejected in favour of swallowing.
