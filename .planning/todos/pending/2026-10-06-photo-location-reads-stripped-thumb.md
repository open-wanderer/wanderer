---
created: 2026-10-06T00:00:00.000Z
title: Fix "use photo location" on already-uploaded photos (reads EXIF from a stripped thumb since #1268)
area: web
priority: high
stack_layer: 0 (image pipeline stack, see notes/image-pipeline-overhaul.md)
files:
  - web/src/lib/components/trail/photo_picker.svelte
  - web/src/lib/components/photo_card.svelte
  - web/src/lib/components/waypoint/waypoint_modal.svelte
---

## Problem

`photo_picker.svelte:167` renders existing photos with `getFileURL(parent, photo, "600x0")` and
passes that `src` to `onexif`. `waypoint_modal.svelte:86` runs `EXIF.getData` on it. PocketBase thumbs
carry no EXIF, so for any photo that is already uploaded the button finds no coordinates. Newly
picked (not yet uploaded) photos still work because their preview is a local object URL.

## Fix

Keep the thumbnail for display, but hand the EXIF reader the original file URL
(`getFileURL(parent, photo)`) for uploaded photos. Small, self-contained, shippable now.

## Superseded by

Stack layer 1 (`feat/images-metadata`) stores GPS + capture time in the DB at upload; the button then
reads the DB and this original-URL path goes away before layer 4 makes originals owner-only.
