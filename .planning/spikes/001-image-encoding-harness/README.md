---
spike: 001
name: image-encoding-ladder
type: comparison
validates: "Given a 12/24 MP camera JPEG on a Raspberry Pi 5, when one CGO-free worker decodes, auto-orients, resizes (CatmullRom) to a 2048/1280/800/400 long-edge ladder and encodes, then a photo finishes in a few seconds with peak RSS well under 512 MB"
verdict: WINNER WebP (chosen in spike 003); stdlib JPEG VALIDATED; jpegli PARTIAL
related: [002, 003]
tags: [images, go, cgo-free, webp, jpegli, arm64, performance]
---

# Spike 001: Image encoding ladder (001a stdlib JPEG · 001b WebP · 001c jpegli)

Shared harness for 001a–c and 002. One binary, `CGO_ENABLED=0`, `-tags nodynamic` (so no system
libwebp is picked up — mirrors the `FROM scratch` image).

## What This Validates

Given a 12 or 24 MP camera photo, when the upload pipeline from
`.planning/notes/image-pipeline-overhaul.md` runs in pure Go on Pi-class hardware, then:
- per-photo wall time is acceptable for a background queue,
- output bytes at **equal perceived quality** justify any extra encode cost,
- the encoder can be linked into the PocketBase binary without side effects.

## Research

| Approach | Library | Pros | Cons | Status |
|---|---|---|---|---|
| stdlib JPEG | `image/jpeg` | zero deps, fast, battle-tested | baseline-only, no progressive, weaker quantisation | measured (001a) |
| WebP | `gen2brain/webp` v0.6.4 | ~half size for graphics; Flutter decodes natively | single-threaded, slow; WASM→Go via `wasm2go` | measured (001b) |
| jpegli | `gen2brain/jpegli` v0.4.2 | ~14% smaller JPEG, universal compatibility | **global registry hijack**, see below | measured (001c) |
| libvips | govips / bimg | fastest | needs cgo → breaks `FROM scratch` | excluded by constraint |

Notes from reading the libraries:
- `gen2brain/webp` ≥ v0.6 no longer uses wazero: libwebp is compiled to WASM and **transpiled to
  Go** with `ncruces/wasm2go`. It first tries a system libwebp via purego; `-tags nodynamic` disables
  that. No cold-compile cost.
- `gen2brain/jpegli` uses wazero on amd64 and `wasm2go` on arm64 (always). Benchmarks on x86 used
  `-tags wasm2go` to match the Pi code path.
- jpegli's `EncodingOptions` zero values **disable** optimized Huffman coding and adaptive
  quantisation and select 4:4:4 chroma. Pass every field explicitly (see `main.go`).
- jpegli's `DecodingOptions.ScaleTarget` decodes JPEGs at N/8 scale (DCT scaling); tested as a
  decode-time optimisation.

## How to Run

```sh
cd .planning/spikes/001-image-encoding-harness
# fixtures/ is gitignored: 12mp.jpg (Pexels, 3024x4032), 24mp.jpg (Wikimedia, CC BY-SA 4.0,
# "A rock cairn, taken from the hiking trail Reykjavegur, Iceland 11", 6000x4000), then:
go build -tags nodynamic,wasm2go -o bin/imgbench . && ./bin/imgbench fixtures   # route.webp
./bin/imgbench matrix -runs 2                                                   # local table
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags nodynamic -ldflags="-s -w" -o bin/imgbench-linux-arm64 .
# on the Pi, from a dir holding imgbench + fixtures/ + run-on-pi.sh:
./run-on-pi.sh   # hard-capped: MemoryMax=600M, no swap, CPUWeight/IOWeight=1, nice 19, oom_score_adj=1000
```

## What to Expect

A markdown table per configuration (input × decoder × encoder × GOMAXPROCS): stage timings, peak
RSS, KB per ladder width, SSIM per width. Raw results: `results/pi5-q80.{md,jsonl}`,
`results/x86-q80.{md,jsonl}`, `results/x86-quality-sweep.txt`.

## Investigation Trail

1. **First run crashed** inside `image.DecodeConfig`. Importing jpegli registers it as the global
   `"jpeg"` format, and its `DecodeConfig` reads only the first 1024 bytes. The 24 MP fixture's SOF is
   at byte 39 180 (13 KB EXIF, 12 KB Photoshop IRB, 3 KB ICC, 10 KB XMP) → WASM `unreachable` trap,
   re-panicked, process dies. Harness switched to calling `jpeg.DecodeConfig` / `jpeg.Decode` directly.
2. Followed it up with `registrycheck/`: with jpegli linked in, PocketBase's own thumb path
   (`imaging.Decode`, `tools/filesystem/filesystem.go:522`) silently decodes through jpegli —
   **4.76 s vs 0.91 s** for the 24 MP file — and a **truncated JPEG decodes without error**
   (stdlib returns `unexpected EOF`). Any `image.DecodeConfig` on a camera JPEG panics.
3. Equal-q comparisons were misleading: stdlib q80 scores SSIM ≈0.964 where jpegli/WebP q80 score
   ≈0.955. Added SSIM (luma, 8×8 windows, stride 4) against the uncompressed resize and swept q 60–90.
4. jpegli scaled decode looked like an obvious win (decode 2250×1500 instead of 6000×4000). Measured
   **slower** than the stdlib full decode on both machines; only memory improved (~60 MB).
5. Ran the full matrix on the user's Pi 5 (8 GB, 27 production containers, swap 94% used) inside a
   `systemd-run --user --scope` with a hard 600 MB cap and no swap. Free memory and swap identical
   before/after, all containers up with unchanged uptimes, `/tmp/imgbench` removed afterwards.

## Results

### Raspberry Pi 5 (Cortex-A76, 4 cores), q80, ladder 2048/1280/800/400

| Input | Pipeline | 1 core total | 4 cores total | Peak RSS |
|---|---|---|---|---|
| 24 MP JPEG | stdlib decode → stdlib JPEG | 2.7 s | 1.9 s | 187 MB |
| 24 MP JPEG | stdlib decode → jpegli | 3.0 s | 2.3 s | 204 MB |
| 24 MP JPEG | stdlib decode → WebP | 5.7 s | 4.9 s | 258 MB |
| 12 MP JPEG | stdlib decode → stdlib JPEG | 1.4 s | 0.8 s | 135 MB |
| 12 MP JPEG | stdlib decode → jpegli | 1.9 s | 1.3 s | 152 MB |
| 12 MP JPEG | stdlib decode → WebP | 4.6 s | 4.1 s | 190 MB |
| 2294×1102 WebP map snapshot | → stdlib JPEG / jpegli / WebP | 0.7 / 1.0 / 2.4 s | 0.4 / 0.7 / 2.1 s | 67–125 MB |

WebP encoding is single-threaded and ~3.4 s of the 24 MP total regardless of cores. Resize is the
only stage that scales with cores (`imaging` parallelises across `GOMAXPROCS`).

### Bytes at equal quality (1280 px, SSIM ≈ 0.96)

| Input | stdlib JPEG | jpegli | WebP |
|---|---|---|---|
| 24 MP | q75 → 171 KB | q80 → 147 KB (**−14%**) | q80 → 159 KB (−7%) |
| 12 MP | q75 → 163 KB | q80 → 142 KB (**−13%**) | q80 → 144 KB (−12%) |
| Map snapshot | q60 → 44 KB (SSIM .978) | q65 → 34 KB (.973) | q80 → 34 KB (.975) |

### Verdicts

- **001a stdlib JPEG — VALIDATED.** Fastest, smallest memory, no side effects. Use q75 to match the
  others' q80 quality. Worst case 2.7 s per 24 MP photo on one Pi 5 core.
- **001b WebP — WINNER (chosen in spike 003, q80).** Originally assessed as PARTIAL: Works CGO-free and wins on graphics, but costs ~2× the CPU of stdlib for
  only 7–12% fewer bytes on photos, and visibly blocks smooth sky gradients at q80 (see spike 003).
  Worth it only for non-photographic images (map snapshots) if at all.
- **001c jpegli — PARTIAL.** Best bytes at equal quality for +0.3–0.5 s per photo, but **must not be
  linked as-is**: its `init()` hijacks the global `"jpeg"` decoder (5× slower PocketBase thumbs,
  `DecodeConfig` panics on camera JPEGs, truncated uploads accepted). Usable only via a fork/vendored
  copy without the `image.RegisterFormat` call, plus a recover() around every call.
- **jpegli scaled decode — INVALIDATED** as a speed optimisation (slower than stdlib under wasm2go).

### Surprises

- The Pi 5 is about as fast per core as a 2016 i7-6820HQ for this workload.
- Pi 4 not measured; expect roughly 2–3× the Pi 5 timings (unverified estimate).
- The 24 MP fixture carries an ICC profile; nothing in the pure-Go stack converts to sRGB. Wide-gamut
  phone photos will look slightly off after stripping — open question for layer 2.
