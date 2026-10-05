---
phase: quick-261005-kif
plan: 01
subsystem: app/android-tls
tags: [android, tls, dart-io, method-channel, issue-1255]
requires: []
provides:
  - user-installed Android CAs trusted by all dart:io clients
affects: [app/lib/main.dart, MainActivity.kt]
key-files:
  created:
    - app/lib/services/user_ca_trust.dart
    - app/test/services/user_ca_trust_test.dart
  modified:
    - app/android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt
    - app/lib/main.dart
decisions:
  - "Native hands DER bytes over a MethodChannel; Dart converts to PEM and adds to SecurityContext.defaultContext"
  - "2 s timeout on the channel call so startup can never hang"
metrics:
  tasks: 2
  files: 4
  completed: 2026-10-05
---

# Quick 261005-kif: Android trusts user-installed CA certificates (#1255)

On Android, MainActivity exposes every `user:` AndroidCAStore certificate over the `com.openwanderer.wanderer/user_ca` channel. `main()` awaits `trustUserCaCertificates()` first, which adds each certificate (DER converted to PEM) to `SecurityContext.defaultContext`. Dio, image caches, and the tile proxy then trust self-hosted instances behind a private root CA.

## Commits

- e582380fd: feat(app): add a dart:io trust helper for user-installed CA certificates (Task 1, TDD: RED confirmed via missing-symbol compile failure, then GREEN)
- 3cf73e73d: fix(app): trust user-installed CA certificates on Android (Task 2)

## Verification

- `flutter test test/services/user_ca_trust_test.dart`: 15/15 pass.
- `flutter analyze` on lib/main.dart, lib/services/user_ca_trust.dart, test/services/user_ca_trust_test.dart: no issues.
- Task 2 automated gate printed OK: MainActivity diff is additive only (zero removed lines), channel and method names match, network_security_config.xml unchanged, call ordering in main() correct.
- Kotlin is not compiled by the executor (no gradle per user rule); imports and override signature were checked by eye.

## Deviations from Plan

- Added one test beyond the listed behaviors: a never-replying handler with a 50 ms timeout returns 0 (covers the timeout part of UCA-04).
- Removed the `dart:typed_data` import from the test file, which analyze flagged as unnecessary_import (the plan anticipated this for the service file; the service never had it).
- `dart format` reformatted one unrelated line in main.dart (readSessionNav); reverted so the diff contains only the intended change.

## Known Stubs

None.

## Threat Flags

None beyond the plan's threat model.

## User-side handoff check (not done by the executor)

Build and install on Android, install a private root CA as a user CA (Settings -> Security -> Encryption & credentials -> Install a certificate -> CA certificate), restart the app, and log in to an instance served with a certificate from that CA. Login should succeed, and `adb logcat | grep "user CA trust"` should show `trusted N of N`.

## Self-Check: PASSED

Files exist, both commits (e582380fd, 3cf73e73d) present on feature/app, nothing pushed.
