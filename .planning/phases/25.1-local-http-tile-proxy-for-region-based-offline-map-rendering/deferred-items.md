# Deferred Items — Phase 25.1

## 25.1-02: Pre-existing generated-file drift (out of scope, not committed)

Running `dart run build_runner build --delete-conflicting-outputs` for Task 2's
`tile_proxy_provider.g.dart` codegen also regenerated
`app/lib/provider/region/tile_repository_provider.g.dart` — its committed
version was stale relative to `tile_repository_provider.dart`'s own
already-committed doc comments (a doc-comment-only diff, no behavior change).
This file is not in this plan's `files_modified` list, so the regenerated
output was left uncommitted in the working tree rather than folded into a
25.1-02 task commit (SCOPE BOUNDARY — only auto-fix issues directly caused by
the current task's own changes). A future plan touching
`tile_repository_provider.dart`/`.g.dart` should pick this up naturally via
its own codegen step.

**RESOLVED (2026-07-24, commit 9e164aab):** the offline-render fix's own
`build_runner` step regenerated this file, so the doc-comment drift is now
committed and no longer outstanding.

## 25.1 UAT fix: iOS offline map rendering (needs maplibre_ios fork)

The on-device UAT fix that made offline tiles render (commit 9e164aab) relies
on `MapLibre.setConnected(true)` in Android `MainActivity` — MapLibre Native
otherwise suppresses ALL online-file-source HTTP requests (including to the
loopback proxy at `http://127.0.0.1`) when its `ConnectivityReceiver` reports
no network in airplane mode.

iOS has the identical connectivity gate, but the `maplibre_ios` 0.3.5 FFI
package exposes **no** `setConnected` / `NetworkStatus` / reachability hook
(only `MLNMapView` and `MLNOfflinePack` are bound). The Android fix works only
because the `maplibre_android` plugin re-exports the native SDK as a Gradle
`api` dependency, letting the app call `MapLibre.setConnected` directly; iOS
has no equivalent seam.

**Path to fix (deferred — needs an iOS build environment, currently blocked):**
fork `maplibre_ios` (or the `maplibre` package) to expose the mbgl
`NetworkStatus::Set(Online)` override to Dart/native, then call it at startup
mirroring `MainActivity`. Upstream's own recommendation for this class of
problem is likewise to fork
(github.com/maplibre/flutter-maplibre-gl/discussions/490). Until then, offline
map tiles will not render on iOS even though the ATS loopback exception (T2)
and all proxy/style code are already in place.
