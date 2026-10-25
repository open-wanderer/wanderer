# Phase 40: Photo Metadata Into the Database - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-06
**Phase:** 40-photo-metadata-into-the-database
**Areas discussed:** Which photos, Who sees location, Existing photos, Photo location button

**Todo folded:** `2026-10-06-photo-location-reads-stripped-thumb` (instead of keeping it as stack layer 0).

---

## Which photos

| Option | Description | Selected |
|--------|-------------|----------|
| Content photos only | Trail, waypoint and summit-log photos; skip avatars and list covers | ✓ |
| Every image field | Also avatars and list covers | |
| Trails + waypoints only | Skip summit logs | |

| Option | Description | Selected |
|--------|-------------|----------|
| Skip remote photos | Only files stored on this instance get metadata | ✓ |
| Accept via federation | Store metadata the origin shares over ActivityPub | |

| Option | Description | Selected |
|--------|-------------|----------|
| Add GPS altitude | Keep altitude alongside position and time | ✓ |
| Only position + time | The minimum | |
| Also camera/exposure | Camera model, lens, exposure | |

| Option | Description | Selected |
|--------|-------------|----------|
| Store as local + offset | Camera wall-clock time plus offset when recorded | |
| Normalize to UTC | One comparable instant; needs a zone guess without offset | |
| You decide | Leave to research and planning | ✓ |

**User's choice:** content photos only, skip remote, add altitude, capture-time representation at Claude's discretion.

---

## Who sees location

| Option | Description | Selected |
|--------|-------------|----------|
| Owner + editors | Only people who can edit the record | ✓ |
| Anyone who can see the record | Follows the record's visibility | |
| Owner only | Not even editors | |

| Option | Description | Selected |
|--------|-------------|----------|
| Never | Metadata never in ActivityPub objects | ✓ |
| Only capture time | Federate time but not position | |

| Option | Description | Selected |
|--------|-------------|----------|
| Not in this phase | No opt-out; privacy setting deferred | ✓ |
| Per-user setting now | Toggle that skips extraction | |

---

## Existing photos

| Option | Description | Selected |
|--------|-------------|----------|
| Background backfill on upgrade | Low-priority, header-only, resumable; done before Phase 41 strips | ✓ |
| Admin command only | Admins run it when they choose | |
| Not at all | Only new uploads | |

| Option | Description | Selected |
|--------|-------------|----------|
| Log progress only | Start/finish/periodic counts in the PocketBase log | ✓ |
| Also an admin command | CLI to rerun/force | |
| You decide | Leave to planning | |

---

## Photo location button

| Option | Description | Selected |
|--------|-------------|----------|
| DB for uploaded, file for new | Stored metadata for uploaded photos; browser EXIF for new picks | |
| Server for everything | Always ask the backend; new picks upload first | |

**User's choice (free text):** "Create an API route that provides the go exif reader via the sveltekit api + a pocketbase route".
**Follow-ups:** (1) the endpoint only parses posted image bytes — uploaded photos use the stored metadata; (2) remove `exif-js` everywhere, including the trail editor's photo-to-waypoint flow.

| Option | Description | Selected |
|--------|-------------|----------|
| Logged-in, header slice | Auth required; client sends first 256 KB; larger bodies rejected | ✓ |
| Logged-in, whole file | Full upload just to read the header | |
| Public, header slice | No auth | |

---

## Claude's Discretion

- Capture-time representation (local wall-clock + offset was the recommended direction).
- Storage shape, Go EXIF library, backfill pacing, endpoint naming.

## Deferred Ideas

- Per-user "don't keep photo locations" privacy setting.
- Features using stored positions/altitude (photos on map / elevation profile / summit-log matching).
- Camera/lens/exposure metadata.
- Flutter app using the parse-only endpoint instead of its local EXIF reader.
