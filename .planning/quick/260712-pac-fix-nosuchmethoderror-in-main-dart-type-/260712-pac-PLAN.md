---
phase: quick-260712-pac
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/main.dart
  - app/lib/provider/router_provider.dart
autonomous: true
requirements: [QUICK-260712-pac]

must_haves:
  truths:
    - "Launching the app no longer throws 'NoSuchMethodError: Class AsyncLoading<UserEntity?> has no instance getter isLoading' from main.dart's auth listener."
    - "The resume-navigation launch check (quick task 260712-m9v) still runs correctly once auth settles."
  artifacts:
    - path: "app/lib/main.dart"
      provides: "Explicitly-typed listenManual callback so AsyncValue.isLoading resolves statically instead of via dynamic dispatch"
      contains: "AsyncValue<UserEntity?>"
  key_links: []
---

<objective>
Fix a runtime crash introduced by quick task 260712-m9v: `_MainAppState.initState` in `app/lib/main.dart` calls `ref.listenManual(authProvider, (prev, next) { if (_resumeHandled || next.isLoading) return; ... })` with an untyped closure. `isLoading` is defined in Riverpod 3.x as an **extension method** on `AsyncValue` (`AsyncValueExtensions.isLoading`, see `riverpod-3.3.2/lib/src/core/async_value.dart:74`), not a real instance member of `AsyncValue`/`AsyncLoading`/`AsyncData`. Extension methods only resolve at compile time against the receiver's *static* type. When the closure's parameters are left untyped, Dart infers `next` as `dynamic` in this call chain (confirmed at runtime: the live app throws `NoSuchMethodError: Class 'AsyncLoading<UserEntity?>' has no instance getter 'isLoading'`, which can only happen via dynamic dispatch — a properly statically-resolved extension call cannot NoSuchMethodError). A `dynamic` receiver skips extension resolution and does a real runtime member lookup, which fails since `isLoading` isn't a real member.

Fix: give the closure explicit parameter types (and the `listenManual` call an explicit type argument, for defense in depth) so `next`'s static type is unambiguously `AsyncValue<UserEntity?>` and `.isLoading` resolves at compile time.

While in this area: `app/lib/provider/router_provider.dart:53` has the exact same untyped-closure shape (`ref.listen(authProvider, (previous, next) { if (!next.isLoading) ... })`), pre-dating this quick task. It carries the identical latent crash risk (just not yet observed/reported) — apply the same explicit-typing fix there too since it's a two-line, directly-analogous change while root-causing this bug, not new scope.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@app/lib/main.dart
@app/lib/provider/router_provider.dart
@app/lib/provider/auth_provider.dart
</context>

<tasks>

<task type="auto">
  <name>Task 1: Explicitly type the two AsyncValue&lt;UserEntity?&gt; listener closures</name>
  <files>app/lib/main.dart, app/lib/provider/router_provider.dart</files>
  <action>
Add `import 'package:wanderer/entities/user_entity.dart';` to `app/lib/main.dart` if not already present (check first — `active_navigation_entity.dart`/`trail_entity.dart` are already imported there, `user_entity.dart` may not be).

In `app/lib/main.dart`, change:
```dart
_authSub = ref.listenManual(authProvider, (prev, next) {
  if (_resumeHandled || next.isLoading) return;
  ...
}, fireImmediately: true);
```
to explicitly type both the generic argument and the closure parameters:
```dart
_authSub = ref.listenManual<AsyncValue<UserEntity?>>(authProvider, (
  AsyncValue<UserEntity?>? prev,
  AsyncValue<UserEntity?> next,
) {
  if (_resumeHandled || next.isLoading) return;
  ...
}, fireImmediately: true);
```
(keep the existing body unchanged — only the closure signature and generic argument change). `AsyncValue` is already available via the existing `package:flutter_riverpod/flutter_riverpod.dart` import in this file.

In `app/lib/provider/router_provider.dart` at the `ref.listen(authProvider, (previous, next) { if (!next.isLoading) { notifier.notify(); } });` call inside `routerListenable` (~line 53), apply the identical fix: explicit generic argument `ref.listen<AsyncValue<UserEntity?>>(authProvider, (AsyncValue<UserEntity?>? previous, AsyncValue<UserEntity?> next) { ... });`. Add `import 'package:wanderer/entities/user_entity.dart';` to this file if not already present, and confirm `AsyncValue` is in scope (via `flutter_riverpod`/`riverpod_annotation`, already imported).
  </action>
  <verify>
    <automated>cd app &amp;&amp; flutter analyze lib/main.dart lib/provider/router_provider.dart</automated>
  </verify>
  <done>Both listener closures have explicit `AsyncValue&lt;UserEntity?&gt;` parameter types and explicit generic type arguments; `flutter analyze` reports no new errors; app launch no longer throws the `NoSuchMethodError` on `next.isLoading` (verify via a manual run since no widget-test harness covers app startup/main.dart).</done>
</task>

</tasks>

<verification>
- `cd app && flutter analyze lib/main.dart lib/provider/router_provider.dart` reports no new errors.
- Manual: run the app (`flutter run`), confirm no `NoSuchMethodError` on startup, and confirm the app still reaches its normal home/map screen (auth listener no longer throws before `_resumeHandled` logic runs).
</verification>

<success_criteria>
- The reported `NoSuchMethodError: Class 'AsyncLoading<UserEntity?>' has no instance getter 'isLoading'` no longer occurs on app launch.
- The resume-navigation dialog logic (quick task 260712-m9v) is otherwise unchanged in behavior.
- No regressions to the pre-existing `routerListenable` redirect-refresh behavior in `router_provider.dart`.
</success_criteria>

<output>
Create `.planning/quick/260712-pac-fix-nosuchmethoderror-in-main-dart-type-/260712-pac-SUMMARY.md` when done.
</output>
