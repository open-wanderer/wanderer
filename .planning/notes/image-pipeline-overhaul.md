---
title: Image pipeline overhaul — process once at upload, serve stripped variants
date: 2026-10-06
context: /gsd-explore session after PR #1334 (header srcset + serve-smaller-thumb hook) exposed the limits of PocketBase's lazy ?thumb pipeline
---

## Why

PR #1268 swapped originals for PocketBase `?thumb=` sizes; PR #1334 patched the fallout
(blurry headers, PNG-encoded WebP thumbs, upscaled thumbs). The review of #1334 showed the
underlying pipeline is the problem, not the individual sizes:

- PocketBase generates thumbs lazily on first request (CPU spike per new size × photo), re-encodes
  JPEG at q95 and **every other format as PNG**, resizes with `imaging.Linear`, and never guards
  against upscaling for `Wx0`.
- Only registered sizes are honoured; anything else silently serves the original. The Flutter app's
  `?thumb=1200x0` gets originals today.
- **Originals are served publicly with EXIF intact, including GPS** — a privacy leak for hiking photos
  that may start at someone's home.
- The waypoint "use photo location" button reads EXIF from a 600x0 thumb since #1268, and thumbs
  carry no EXIF, so it no longer finds coordinates on already-uploaded photos.
- HEIC is accepted on upload but PocketBase doesn't thumb it, so most browsers can't display it.

## Decisions (2026-10-06)

| Question | Decision |
|---|---|
| Hardware to optimise for | **Small home server (Pi / NAS).** CPU is scarce; process once, in the background, bounded; never on-demand bursts. |
| Originals | **Keep byte-for-byte, stop serving publicly.** Owner-only download. Public views use variants. |
| EXIF in served files | **Strip everything.** GPS + capture time extracted into DB fields at upload, owner controls display. |
| Delivery | **One GitHub stack (`gh stack`, trunk `dev`)**, each layer independently reviewable and useful. #1334 merges first as the short-term fix. |

## Industry baseline (research summary)

- Mastodon, Pixelfed, Immich, Discourse, PhotoPrism: **eager, upload-time** generation of a small fixed
  ladder + placeholder; strip metadata. On-demand (Ghost, Nextcloud, imgproxy, Cloudinary) only works
  well behind a CDN — PhotoPrism explicitly advises against it on a Pi.
- Format: WebP is the safe common denominator (browsers + Flutter native decode). AVIF ~95% browser
  support but very slow to encode on ARM, Flutter support only via platform fallback. jpegli has no
  pure-Go port. Prefer `<picture>`/single format over `Accept` negotiation (needs `Vary`, caches poorly
  without a CDN).
- Ladder ≈ 400 / 800 / 1280 / 2048 px, WebP q75–82 or JPEG q80–85.
- Serve with content-unique URLs + `Cache-Control: public, max-age=31536000, immutable`; store
  width/height (no CLS) and a ThumbHash or dominant colour (placeholder).
- Lightbox: show the cached card image immediately (PhotoSwipe `msrc`), then the 2048 variant;
  original only on explicit download.
- Go options: libvips (govips/bimg) needs cgo → breaks our `CGO_ENABLED=0` + `FROM scratch` build;
  pure Go (`imaging`/`x/image` with CatmullRom, `gen2brain/webp` via WASM/wazero) keeps it; imgproxy
  sidecar is an opt-in for bigger hosts.

## Target architecture

1. **Upload hook → background queue** (1 worker default, admin-configurable). Per image: decode
   (JPEG/PNG/WebP; HEIC TBD), extract GPS + DateTimeOriginal to DB, auto-orient, convert to sRGB,
   write stripped variants 400/800/1280/2048 (never upscaled), store width/height/thumbhash.
2. **Serving route** for variants with immutable caching; SvelteKit proxy forwards the headers.
   Missing variant → enqueue + serve nearest existing size (never block on encoding).
3. **Web:** one shared responsive image component (srcset/sizes/width/height/thumbhash, `lazy`
   by default, `fetchpriority="high"` on the LCP header); progressive lightbox; all `?thumb=` call
   sites migrated.
4. **Originals locked down:** owner-only download; ActivityPub attachments reference a stripped
   variant (1280 or 2048), never the original.
5. **App:** pick the nearest ladder width for display width × DPR; ThumbHash placeholders. The app
   already disk-caches via `cached_network_image` (PERFORMANCE-PLAN P5).
6. **Backfill command** for existing photos, low priority, resumable.

## Stack layout (trunk `dev`)

| Layer | Branch (proposed) | Scope |
|---|---|---|
| 0 | `fix/photo-location-original` | Quick fix for the #1268 regression (todo `2026-10-06-photo-location-reads-stripped-thumb`). |
| 1 | `feat/images-metadata` | Extract GPS + capture time into DB at upload; "use photo location" reads DB, not EXIF. |
| 2 | `feat/images-variants` | Queue, ladder generation, storage layout, serving route, backfill command. No UI change. |
| 3 | `feat/images-web` | Responsive image component, progressive lightbox, migrate every `?thumb=` site. |
| 4 | `feat/images-originals-private` | Owner-only originals, federation uses variants. |
| 5 | `feat/images-app` | Flutter nearest-width + ThumbHash. Lands after `feature/app` merges, or as a separate follow-up if it hasn't. |

**Gate before layer 2:** the arm64 encoding spike (format + library + ladder cost on a Pi).

## Open decisions

- Variant storage: beside PocketBase's `thumbs_<file>/` dirs under the record path (deleted with the
  record?) vs. a separate `image_variants` collection — affects backups, deletion and federation
  cleanup. Verify what PocketBase deletes on record/file removal before choosing.
- ~~Format~~ **Decided 2026-10-06 (spikes 001–003): WebP q80 via `gen2brain/webp` (wasm2go, `-tags nodynamic`).**
  Pi 5: 5.7 s per 24 MP photo on one core, 258 MB peak. Never link `gen2brain/jpegli` (hijacks the
  global `"jpeg"` decoder). See `.planning/spikes/MANIFEST.md`.
- HEIC decode (libheif needs cgo; HEVC licensing) vs. converting client-side before upload.
- Exact ladder/quality, and whether `1200x0` requests from released app builds get mapped onto the
  ladder (they currently receive originals; layer 4 changes that).
- What happens to registered PocketBase thumb sizes and the #1334 serve-smaller hook once variants
  exist — keep as fallback for legacy records until backfill completes, then retire.
