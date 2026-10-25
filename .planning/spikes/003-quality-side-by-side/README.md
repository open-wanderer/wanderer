---
spike: 003
name: quality-side-by-side
type: standard
validates: "Given the variants from spike 001 at equal SSIM, when viewed side by side at 1x/2x against a lossless reference, then a format and quality can be picked by eye"
verdict: VALIDATED
related: [001]
tags: [images, quality, webp, jpegli, viewer]
---

# Spike 003: Quality side by side

## What This Validates

SSIM is luma-only and blind to chroma bleeding and WebP's gradient blocking. A human comparison at
the sizes Wanderer actually serves decides the format and quality.

## How to Run

```sh
cd .planning/spikes/003-quality-side-by-side
./generate.sh          # needs the 001 harness binary and fixtures; writes images/ (gitignored) + manifest.js
open index.html        # file:// works, no server needed
```

## What to Expect

Six synchronised panels per photo (lossless reference, stdlib JPEG q75/q85, jpegli q80, WebP q80/q85)
at 2048/1280/800 px and 1×/2×/4× zoom. Drag pans all panels; hold **B** to flip every panel to the
reference. Captions show KB, Δ vs stdlib JPEG q75, and SSIM.

## Investigation Trail

- First headless render opened on flat sky, where only WebP's blocking shows. Centred the initial pan
  on the photo so detail (cloud edges, ridgelines, rock texture) is visible first.
- Early observation (12 MP, 1280 px, 2×): WebP q80 bands the blue sky gradient into visible blocks at
  the same SSIM where jpegli q80 stays smooth.

- At 4× on cloud texture, WebP q80 smooths fine detail that WebP q85 keeps (181 KB vs 145 KB at
  1280 px); jpegli q80 sits in between at 142 KB.

## Results

**VALIDATED — decision (user, 2026-10-06): WebP q80 for all variants.** WebP was preferred as the widely
accepted standard (native in every current browser and in Flutter), over jpegli's slightly better
bytes-per-quality and its registry landmine. q80 accepted knowing it softens fine texture at high zoom;
at 1× the variants are visually indistinguishable from the reference in normal viewing.
