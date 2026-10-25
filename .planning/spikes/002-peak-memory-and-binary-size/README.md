---
spike: 002
name: peak-memory-and-binary-size
type: standard
validates: "Given a 24 MP photo processed by one worker, when the pipeline runs, then peak RSS stays well under 512 MB, and the encoders add little to the linux/arm64 binary"
verdict: VALIDATED
related: [001]
tags: [images, memory, arm64, binary-size]
---

# Spike 002: Peak memory and binary size

## What This Validates

Given a 1 GB-class Pi must not swap, when one worker processes a 24 MP JPEG, then peak RSS stays
well under 512 MB. Given the backend ships as a `FROM scratch` binary, then the encoders don't bloat it.

## How to Run

Peak RSS comes from the spike 001 matrix (each configuration in its own process, `getrusage` max
RSS). Binary size: `sizecheck/` in the 001 harness, built with
`CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags nodynamic -ldflags="-s -w"`.

## Investigation Trail

- Measured peak RSS per configuration on the Pi 5 inside a 600 MB hard cap (`GOMEMLIMIT=500MiB`
  unset-in-practice: no run came close).
- jpegli scaled decode lowers peak RSS by ~60 MB on 24 MP input but is slower (spike 001), so it is
  not worth the jpegli registry risk for memory alone.
- `GOMAXPROCS` barely changes peak memory (resize workers share the source buffer).

## Results

| Pipeline (24 MP input, Pi 5) | Peak RSS |
|---|---|
| stdlib decode → stdlib JPEG | 184–187 MB |
| stdlib decode → jpegli | 191–204 MB |
| stdlib decode → WebP | 256–258 MB |
| jpegli scaled decode → stdlib JPEG | 126–142 MB |

| linux/arm64 binary, `-s -w` | Size |
|---|---|
| imaging + stdlib JPEG | 2.00 MB |
| + gen2brain/webp | 3.69 MB (+1.7 MB) |
| + gen2brain/jpegli | 3.19 MB (+1.2 MB) |
| + both | 4.88 MB (+2.9 MB) |
| current wanderer `db` binary | 31.81 MB |

**VALIDATED.** One worker peaks at ≤ 260 MB on 24 MP input; two concurrent workers would still fit
in 512 MB. Binary growth is under 10% even with both encoders. The real build should still set a
`GOMEMLIMIT` and cap input megapixels (decompression bombs: a 20 MB JPEG can declare 100+ MP).
