---
phase: quick-260712-pac
plan: 01
subsystem: mobile-app
tags: [riverpod, dart, async-value, flutter, extension-methods]

requires:
  - phase: quick-260712-m9v
    provides: "The listenManual auth-settle check in main.dart's initState that this plan fixes"
provides:
  - "Explicitly-typed AsyncValue<UserEntity?> listener closures in main.dart and router_provider.dart so .isLoading resolves as a static extension call instead of dynamic dispatch"
affects: [mobile-app-startup, mobile-app-routing]

tech-stack:
  added: []
  patterns:
    - "Riverpod listen/listenManual callbacks on authProvider must carry explicit AsyncValue<T> generic + parameter types — Dart's type inference otherwise widens the closure params to dynamic, which breaks AsyncValueExtensions.isLoading (an extension method, not a real instance member) at runtime with NoSuchMethodError instead of a compile error."

key-files:
  created: []
  modified:
    - app/lib/main.dart
    - app/lib/provider/router_provider.dart

key-decisions:
  - "Applied the identical explicit-typing fix to router_provider.dart's routerListenable ref.listen call even though it wasn't the reported crash — same untyped-closure shape, same latent risk, two-line change while already root-causing this bug class."

patterns-established:
  - "Any ref.listen/listenManual(authProvider, ...) callback must use explicit AsyncValue<UserEntity?> generic argument and (AsyncValue<UserEntity?>? prev, AsyncValue<UserEntity?> next) parameter types."

requirements-completed: [QUICK-260712-pac]

duration: 5min
completed: 2026-07-12
---

# Quick Task 260712-pac: Fix NoSuchMethodError in main.dart type inference Summary

**Explicitly typed two Riverpod auth-listener closures (main.dart, router_provider.dart) so `AsyncValue.isLoading` resolves as a static extension call instead of failing at runtime via dynamic dispatch.**

## Performance

- **Duration:** ~5 min
- **Tasks:** 1 completed
- **Files modified:** 2

## Accomplishments
- Fixed the reported runtime crash: `NoSuchMethodError: Class 'AsyncLoading<UserEntity?>' has no instance getter 'isLoading'` thrown from `_MainAppState.initState`'s `ref.listenManual(authProvider, ...)` callback in `app/lib/main.dart`.
- Root cause: the closure's `(prev, next)` parameters were left untyped, so Dart inferred `next` as `dynamic`. `isLoading` is an extension method (`AsyncValueExtensions.isLoading`) defined in Riverpod 3.x, and extension methods only resolve against a *static* receiver type — a `dynamic` receiver skips extension resolution and attempts a real instance-member lookup, which fails since `isLoading` isn't a real member on `AsyncValue`/`AsyncLoading`.
- Fix: added explicit generic type argument (`ref.listenManual<AsyncValue<UserEntity?>>`) and explicit closure parameter types (`AsyncValue<UserEntity?>? prev, AsyncValue<UserEntity?> next`) so `next`'s static type is unambiguous and `.isLoading` resolves at compile time.
- Also fixed the identical untyped-closure pattern in `app/lib/provider/router_provider.dart`'s `routerListenable` (`ref.listen(authProvider, (previous, next) => ...)`), which pre-dated this quick task and carried the same latent crash risk, per the plan's explicit instruction to fix it while in the area.

## Task Commits

1. **Task 1: Explicitly type the two AsyncValue<UserEntity?> listener closures** - `5697a064` (fix)

**Plan metadata:** committed separately by orchestrator (docs commit not part of this agent's scope).

## Files Created/Modified
- `app/lib/main.dart` - Added `user_entity.dart` import; `_authSub = ref.listenManual<AsyncValue<UserEntity?>>(authProvider, (AsyncValue<UserEntity?>? prev, AsyncValue<UserEntity?> next) { ... })` replacing the untyped closure.
- `app/lib/provider/router_provider.dart` - Added `user_entity.dart` import; `ref.listen<AsyncValue<UserEntity?>>(authProvider, (AsyncValue<UserEntity?>? previous, AsyncValue<UserEntity?> next) { ... })` replacing the untyped closure in `routerListenable`.

## Decisions Made
- Fixed `router_provider.dart`'s analogous untyped closure alongside the reported bug in `main.dart`, per the plan's explicit scope note (same shape, same latent risk, trivial two-line change while root-causing this bug class — not new scope).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `flutter analyze lib/main.dart lib/provider/router_provider.dart` reported "No issues found!" after the change. Manual on-device verification of the fixed startup path (no `NoSuchMethodError`, app reaches home/map screen) is deferred to the user per the plan's verification note (no widget-test harness covers `main.dart` startup).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The startup crash fix is code-complete and statically verified (`flutter analyze` clean).
- Manual on-device verification recommended: launch the app and confirm no `NoSuchMethodError` occurs, and that the resume-navigation dialog logic (quick task 260712-m9v) still runs correctly once auth settles.

---
*Phase: quick-260712-pac*
*Completed: 2026-07-12*

## Self-Check: PASSED
- FOUND: app/lib/main.dart
- FOUND: app/lib/provider/router_provider.dart
- FOUND: 5697a064
