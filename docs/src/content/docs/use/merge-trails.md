---
title: Merge trails
description: Link repeated or duplicate trail recordings into a single target trail.
---

<span class="-tracking-[0.075em]">wanderer</span> can link multiple trails into a single target trail. This is useful when the same route was recorded multiple times, imported from multiple services, or uploaded separately even though it belongs to the same underlying trail.

When a trail is merged, the source trail is converted into a summit log entry on the selected target trail. Depending on the chosen options, additional data such as photos, comments, tags and likes can also be merged.

## When to Use Trail Merging

Trail merging is helpful when:

- you recorded the same route multiple times and want one canonical trail page
- a route was uploaded as a new trail, even though it was actually supposed to be a further iteration of an existing trail

If the route already exists and you simply want to log another outing, using a [summit log](/use/summit-logs) directly is usually the better choice.

## Manual Merge

You can merge trails manually from the trail actions menu:

- select multiple trails and choose **Link**
- or open a single trail and choose **Merge with similar trail**

Before the merge is executed, <span class="-tracking-[0.075em]">wanderer</span> asks the backend for a suggested target trail. The backend uses the same target selection strategy in all merge modes and prefers trails that preserve the most useful information.

The target suggestion currently considers:

- existing summit logs
- external references from integrations
- content richness such as comments, photos, waypoints and descriptions
- how centrally the trail geometry fits within the candidate set
- trail age as a deterministic fallback

Warnings are shown before the merge if the selected trails differ noticeably in geometry or location.

## Automatic Matching for Similar Trails

The **Merge with similar trail** action searches for strong geometric matches of the currently open trail. Only trails with a sufficiently similar forward direction are considered. Out-and-back reversals are not treated as the same trail.

## Maintenance Page

The maintenance page groups potentially repeated or duplicate trails so that you can review them in batches:

- open **Settings → Repeated trails / duplicates**
- inspect each group on the map
- choose the target trail directly in the list
- merge the group once you are satisfied

This page is especially useful after large imports or when you want to consolidate older data.

## Integrations

Integrations can optionally auto-merge imported trails, but only when the backend finds exactly one clear target candidate. This keeps imports conservative and avoids accidentally merging different routes.

External references from integrations are preserved during merges, so future imports can still recognize already-linked trails correctly.

## What Happens During a Merge

At a high level, the backend:

1. determines or receives a target trail
2. creates a new summit log from the source trail on the target trail
3. optionally merges existing summit logs, comments, likes, tags and photos
4. reassigns external references to the target trail
5. optionally deletes the source trail

The merge itself runs transactionally so partially completed merges are avoided.
