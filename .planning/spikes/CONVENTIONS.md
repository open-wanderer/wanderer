# Spike Conventions

Patterns and stack choices established across spike sessions. New spikes follow these unless the question requires otherwise.

## Stack

- Go spikes live in their own module under `.planning/spikes/NNN-*/` (`go 1.26`, run with
  `GOTOOLCHAIN=go1.26.1`), never inside `db/`.
- Build exactly like production: `CGO_ENABLED=0`, `-tags nodynamic`; cross-compile
  `GOOS=linux GOARCH=arm64 -ldflags="-s -w"` for the Pi.

## Structure

- One harness binary with subcommands (`bench` = one config, JSON line; `matrix` = each config in a
  child process so peak RSS is clean). Fixtures, binaries and generated output are gitignored;
  results (`results/*.md|jsonl|txt`) are committed.
- Visual comparisons: static `index.html` + generated `manifest.js`, opened via `file://`, with a
  `generate.sh` that rebuilds the assets.

## Patterns

- Running on the user's home server (`user@homeserver.lan`, Pi 5, production containers): always
  inside `systemd-run --user --scope -p MemoryMax=… -p MemorySwapMax=0 -p CPUWeight=1 -p IOWeight=1`,
  `nice -n 19`, `oom_score_adj=1000`, a `MemAvailable` guard, working dir under `/tmp/<spike>`,
  check `docker ps` before/after and delete the directory when done.
- Compare encoders at equal quality (SSIM against an uncompressed reference), never at equal `q`.

## Tools & Libraries

- `gen2brain/webp` v0.6.4 (wasm2go, CGO-free) — chosen encoder.
- `disintegration/imaging` v1.6.2 (CatmullRom; parallelises across `GOMAXPROCS`).
- `rwcarlsen/goexif` for orientation, GPS, DateTimeOriginal.
- Avoid importing `gen2brain/jpegli` into anything that shares a process with PocketBase: its
  `init()` replaces the global `"jpeg"` decoder.
