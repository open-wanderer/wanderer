# Spike Wrap-Up Summary

**Date:** 2026-10-06
**Spikes processed:** 3
**Feature areas:** Image variant pipeline, Benchmarking on shared hosts
**Skill output:** `./.claude/skills/spike-findings-wanderer/` (local only — `.claude/` is gitignored)

## Processed Spikes
| # | Name | Type | Verdict | Feature Area |
|---|------|------|---------|--------------|
| 001 | image-encoding-harness (001a stdlib JPEG · 001b WebP · 001c jpegli) | comparison | WebP chosen; stdlib JPEG validated; jpegli partial | Image variant pipeline, Benchmarking |
| 002 | peak-memory-and-binary-size | standard | VALIDATED | Image variant pipeline, Benchmarking |
| 003 | quality-side-by-side | standard | VALIDATED (WebP q80 chosen) | Image variant pipeline |

## Key Findings

- **Decision:** variants are WebP q80 (`gen2brain/webp` v0.6.4, `-tags nodynamic`), ladder
  2048/1280/800/400 on the long edge, never upscaled, cascade-resized with `imaging.CatmullRom`.
- **Cost on a Pi 5:** 5.7 s (1 core) / 4.9 s (4 cores) and 258 MB peak per 24 MP photo; 4.6 s and
  190 MB per 12 MP. WebP encode is single-threaded. +1.7 MB binary. Pi 4 unmeasured (~2–3× est.).
- **Landmine:** `gen2brain/jpegli` registers itself as the global `"jpeg"` decoder — 5× slower
  PocketBase thumbs, `image.DecodeConfig` panics on camera JPEGs, truncated JPEGs accepted. Never link it.
- **Method:** compare encoders at equal SSIM, not equal `q`; decode via stdlib functions directly,
  not the `image` registry.
- **Open for layer 2:** ICC → sRGB conversion (pure Go ignores embedded profiles), megapixel cap
  against decompression bombs, `recover()` + `GOMEMLIMIT` around the worker.
- **Safety:** the Pi 5 benchmark ran inside a hard 600 MB / no-swap cgroup; free memory, swap and all
  27 container uptimes were unchanged.
