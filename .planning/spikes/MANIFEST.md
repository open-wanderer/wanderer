# Spike Manifest

## Idea

Replace PocketBase's lazy `?thumb=` resizing with an upload-time, pure-Go image pipeline for Wanderer:
extract GPS/capture time into the DB, auto-orient, strip metadata, and write a fixed long-edge ladder
(2048/1280/800/400) in a bounded background queue that a Raspberry Pi can run without hurting the
rest of the host. See `.planning/notes/image-pipeline-overhaul.md`. These spikes gate stack layer 2
(`feat/images-variants`): which encoder, what quality, what it costs on a Pi.

## Requirements

- Must stay `CGO_ENABLED=0` and ship in the existing `FROM scratch` image (build with `-tags nodynamic`
  so gen2brain libraries never probe for system libraries).
- Must be safe on a shared home server: bounded workers, memory well under 512 MB per worker, and no
  crash on any user upload (every decoder/encoder call behind `recover()`).
- Must not change the behaviour of PocketBase's own image code (no global `image.RegisterFormat`
  side effects from linked libraries).
- **Variants are WebP q80** (`gen2brain/webp`, Method 4) for every ladder width — user decision after
  spike 003. Not jpegli, not stdlib JPEG.
- Benchmarks on shared hosts run inside a hard cgroup cap (`systemd-run --user --scope`,
  `MemoryMax`, `MemorySwapMax=0`, lowest CPU/IO weight, `oom_score_adj=1000`) — the user's Pi runs
  production containers.

## Spikes

| # | Name | Type | Validates | Verdict | Tags |
|---|------|------|-----------|---------|------|
| 001a | ladder-stdlib-jpeg | comparison | 24 MP → ladder on Pi 5 in seconds, CGO-free | ✓ VALIDATED (2.7 s / 1 core, 187 MB) | images, go, arm64 |
| 001b | ladder-webp-wasm2go | comparison | Same with gen2brain/webp | ✓ WINNER (chosen in 003; 5.7 s / 1 core, 258 MB, −7–12% bytes) | images, webp |
| 001c | ladder-jpegli-wasm2go | comparison | Same with gen2brain/jpegli | ⚠ PARTIAL (3.0 s, −13–14% bytes, registry hijack landmine) | images, jpegli |
| 002 | peak-memory-and-binary-size | standard | ≤ 512 MB per worker, small binary growth | ✓ VALIDATED (≤ 260 MB, +1.2–2.9 MB) | images, memory |
| 003 | quality-side-by-side | standard | Pick format/quality by eye at equal SSIM | ✓ VALIDATED (WebP q80 chosen) | images, quality |

Harness, fixtures (gitignored) and raw results for 001/002: `001-image-encoding-harness/`.
