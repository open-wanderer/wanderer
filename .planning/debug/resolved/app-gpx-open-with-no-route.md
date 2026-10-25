---
status: resolved
trigger: "GitHub issue #1438: [App] Opening a GPX via \"Open with\" (ACTION_VIEW) shows GoException: no routes for location: content://…"
created: 2026-10-25
updated: 2026-10-25
---

# Debug: app-gpx-open-with-no-route

## Symptoms

<!-- DATA_START -->
- **Expected:** Opening a GPX file into the app via Android "Open with" (ACTION_VIEW intent) runs the same import flow as + -> Import file (which works with the same file).
- **Actual:** App starts and shows the router error page: "Page Not Found / GoException: no routes for location: content://media/external/downloads/126256 / [Home]". Nothing is imported.
- **Errors:** `GoException: no routes for location: content://media/external/downloads/126256`
- **Timeline:** Reported 2026-10-24 on Google Play open test, app-v0.1.0-beta.20, instance v0.21.1-app. Unknown whether it ever worked.
- **Reproduction:** 1) Export a route from Mapy.com as GPX (lands in Downloads). 2) Open the file and pick wanderer in the chooser (or open from Downloads app / file manager). 3) App shows the error screen.
- **Reporter's hypothesis:** AndroidManifest.xml declares an ACTION_VIEW intent filter for trail MIME types (next to ACTION_SEND handled by receive_sharing_intent). With Flutter's default deep linking enabled, the VIEW intent's data URI is pushed to GoRouter as a route location. Suggested fixes: `flutter_deeplinking_enabled=false` if deep links aren't otherwise needed, or have router redirect/onException swallow content:// and file:// locations and hand the URI to the existing share-import path.
<!-- DATA_END -->

Project constraints (from memory): app code lives in `app/` on branch `feature/app`; never run `flutter build`/`adb install` (user builds and installs) — hand off after `flutter analyze` + tests; GPX parsing only via `parseGpxSafely`; commit messages headline-only, no body, no Co-Authored-By; never push without permission.

## Current Focus

- hypothesis: Flutter's FlutterActivity deep linking is on by default (manifest has no flutter_deeplinking_enabled meta-data); for the ACTION_VIEW intent it hands intent.getData().toString() ("content://media/...") to the engine as the initial route (cold start) or via pushRouteInformation (onNewIntent, warm start). GoRouter(initialLocation: '/', overridePlatformDefaultLocation: false) adopts that platform route as its initial location, finds no GoRoute for it, and renders the default error page. receive_sharing_intent separately delivers the same intent through its own channel, so the import path itself is probably fine.
- bug_class: Bohrbug (deterministic; every VIEW intent reproduces it)
- status of hypothesis: CONFIRMED by source reading (engine, go_router, receive_sharing_intent); no device attached (adb devices empty), so no on-device repro possible from here.
- reasoning_checkpoint:
    hypothesis: "The unmatched content:// error page is caused by GoRouter adopting Flutter's platform route (the VIEW intent's data URI) as a location, because GoRouter is built with overridePlatformDefaultLocation: false and has no policy for scheme-bearing locations."
    confirming_evidence:
      - "FlutterActivityLaunchConfigs.deepLinkEnabled() defaults to true; FlutterActivityAndFragmentDelegate.maybeGetInitialRouteFromIntent returns intent.getData().toString() with no scheme check, and onNewIntent forwards it via pushRouteInformation"
      - "go_router 17.2.1 router.dart _effectiveInitialLocation('/') returns WidgetsBinding.platformDispatcher.defaultRouteName whenever it is not '/', and configuration.dart:355 throws exactly 'no routes for location: $uri'"
      - "router_provider.dart builds GoRouter with initialLocation '/' only: no overridePlatformDefaultLocation, onEnter, onException or errorBuilder"
    falsification_test: "Widget test: set platformDispatcher.defaultRouteNameTestValue to content://media/external/downloads/126256 and mount a GoRouter without the fix; error page must appear. With the fix the '/' route must render. If the unfixed router renders '/' the hypothesis is wrong."
    fix_rationale: "Make the router refuse to treat scheme-bearing URIs as app locations: overridePlatformDefaultLocation: true (cold start) plus an onEnter guard that blocks any location with a scheme (warm start via pushRouteInformation). Inbound files are owned by receive_sharing_intent, which already receives the same intent, so the router has nothing to do with them. Covers Android content:// and file://, and iOS file:// open-in and the ShareMedia-<bundle id>:// share-extension scheme, in one Dart place that flutter test can verify."
    blind_spots: "No device attached: cannot confirm on-device that (1) intent.type is non-null for the reporter's VIEW intent (receive_sharing_intent ignores VIEW with null type as a media file and reports it as a URL), (2) the import sheet then opens. iOS behaviour inferred from source, not run."
    candidate_causes:
      - "code: GoRouter adopts platform route as location (confirmed)"
      - "config: AndroidManifest has no flutter_deeplinking_enabled=false although the app owns VIEW intents via receive_sharing_intent (contributing: this is why the URI reaches Dart at all)"
      - "environment: file manager sends VIEW without explicit MIME type (not the cause of the error page; relevant only to what happens after the fix)"
    and_gate: "No single other condition is needed: the error page appears for any VIEW intent on the unfixed router. Two layers (config lets URI through, code has no policy) but fixing either alone removes the symptom; fixing the code layer also covers iOS."
- next_action: AWAITING USER on-device check (user builds and installs; never run flutter build / adb install). Then archive: move this file to .planning/debug/resolved/, commit with a headline-only message (no body, no Co-Authored-By), append to .planning/debug/knowledge-base.md. Do not push.

## Evidence

- timestamp: 2026-10-25
  checked: app/android/app/src/main/AndroidManifest.xml
  found: MainActivity has MAIN/LAUNCHER, SEND (mime types) and VIEW (mime types only, no scheme/host) filters. No `flutter_deeplinking_enabled` meta-data. `wanderer://` scheme VIEW filter lives on a different activity, com.linusu.flutter_web_auth_2.CallbackActivity.
  implication: Flutter deep linking is at its default for MainActivity; OAuth callback is not on MainActivity.

- timestamp: 2026-10-25
  checked: /opt/flutter/engine/.../FlutterActivityLaunchConfigs.java deepLinkEnabled(), FlutterActivityAndFragmentDelegate.maybeGetInitialRouteFromIntent / onNewIntent
  found: deepLinkEnabled returns true when the meta-data key is absent. When true, `intent.getData().toString()` becomes the initial route (cold start) and is sent via navigationChannel.pushRouteInformation on onNewIntent (warm start). No scheme/host check.
  implication: Any ACTION_VIEW intent with a data URI, including content://, is forwarded to the Dart router as a location.

- timestamp: 2026-10-25
  checked: app/lib/provider/router_provider.dart GoRouter(...) and go_router-17.2.1 router.dart `_effectiveInitialLocation`
  found: GoRouter built with initialLocation '/', overridePlatformDefaultLocation left false, no onEnter, no onException, no errorBuilder. `_effectiveInitialLocation('/')` returns the platform default route name whenever it is not '/'. Unmatched location -> GoException('no routes for location: $uri') (configuration.dart:355) -> default error screen.
  implication: Exactly reproduces the reported message and page. Cold start path confirmed by source.

- timestamp: 2026-10-25
  checked: receive_sharing_intent-1.9.0 ReceiveSharingIntentPlugin.kt + FileDirectory.kt
  found: handleIntent accepts ACTION_VIEW when intent.type != null (copies content:// URI into cacheDir via ContentResolver and returns the cache path as SharedMediaFile.path). onNewIntent also handled. So the import plumbing in main.dart (getInitialMedia / getMediaStream -> _maybeHandleShare -> importTrailFile) already receives VIEW intents independently of the router.
  implication: The router error is the visible defect; share path is likely already correct for the typed-VIEW case.

- timestamp: 2026-10-25
  checked: flutter_web_auth_2-5.0.3 CallbackActivity.kt; ios Info.plist; grep for defaultRouteName / initialRoute / SystemNavigator / pushRouteInformation / notification routing in app/lib and app/android
  found: CallbackActivity is a plain android.app.Activity that hands the URL to the plugin and calls finishAndRemoveTask(); it never touches FlutterActivity's deep-link path. No code in app/lib or android reads defaultRouteName, sets an initial route, or routes from notification taps. MainActivity has no http(s)/app-link filters. iOS Info.plist declares CFBundleDocumentTypes (trail files, "Open in"), URL schemes `ShareMedia-$(PRODUCT_BUNDLE_IDENTIFIER)` and `wanderer`, and no FlutterDeepLinkingEnabled key (default on), so the same class exists latently on iOS (file:// open-in; the ShareMedia hand-off has an empty path and normalises to '/').
  implication: Nothing in the app relies on Flutter handing platform URIs to the router, so ignoring them in the router is safe. The Android manifest flag would also be safe but is untestable here and does not cover iOS.

- timestamp: 2026-10-25
  checked: test/provider/router_inbound_uri_test.dart run against a copy with overridePlatformDefaultLocation and onEnter removed (unfixed config)
  found: RED. All 7 cold-start tests fail (home route not rendered: error page for content://, file:// and SAF URIs; ShareMedia URI lands on '/' only because its empty path normalises). All 7 warm-start tests fail (content:// pushed via the flutter/navigation channel replaces the current screen with the error page).
  implication: Reproduces the reported behaviour at both seams the engine uses, with real go_router and the real binding.

- timestamp: 2026-10-25
  checked: same tests with the fix, plus single-half mutations
  found: GREEN 20/20. Removing only overridePlatformDefaultLocation: all cold-start tests fail. Removing only onEnter: cold-start tests pass, warm-start tests fail. Both halves are required.
  implication: The fix is two-part and each part is covered by a test that fails without it.

- timestamp: 2026-10-25
  checked: adb devices
  found: no device attached.
  implication: Not verified on a real device. Not verified: (1) the reporter's VIEW intent carries a non-null type, (2) the import sheet opens afterwards.

## Eliminated

- hypothesis: receive_sharing_intent ignores ACTION_VIEW, so the import would not run even without the router error
  evidence: ReceiveSharingIntentPlugin.handleIntent accepts ACTION_VIEW when intent.type != null, FileDirectory.getDataColumn copies a content:// URI (authority "media") into cacheDir via ContentResolver and returns the file path; onNewIntent is handled too.
  timestamp: 2026-10-25

- hypothesis: disabling Flutter deep linking would break the OAuth callback
  evidence: the wanderer:// callback is caught by flutter_web_auth_2's own CallbackActivity (plain Activity), which does not use FlutterActivity's initial-route / pushRouteInformation path. Not applied anyway, see Resolution.
  timestamp: 2026-10-25

## Resolution

root_cause: Flutter's Android embedding forwards the data URI of every ACTION_VIEW intent to the Dart router as a deep link (initial platform route on a cold start, pushRouteInformation from onNewIntent when the singleTask activity is running); flutter_deeplinking_enabled is unset so it defaults to true. GoRouter was built with overridePlatformDefaultLocation: false, so it adopted "content://media/external/downloads/126256" as its initial location instead of '/', found no route for it and rendered its default "Page Not Found" screen (GoException: no routes for location). The file itself is delivered separately by receive_sharing_intent, so nothing imported because the user was stuck on the error page, not because the import path failed.
fix: Added lib/util/inbound_uri_guard.dart (isInboundUri = uri.hasScheme; ignoreInboundUris onEnter guard returning Block.stop() for such URIs). router_provider.dart now builds GoRouter with overridePlatformDefaultLocation: true (cold start: ignore the platform route) and onEnter: ignoreInboundUris (warm start: block pushed inbound URIs, user stays on the current screen). Router-level rather than the manifest flag so it also covers iOS file open-in and the ShareMedia hand-off, and so flutter test can verify it. AndroidManifest.xml and iOS Info.plist untouched.
verification:
  target_test: { result: pass, detail: "test/provider/router_inbound_uri_test.dart 20/20; RED against unfixed config, GREEN with fix" }
  mutation_check: { result: skipped, reason_if_skipped: "Stryker is JS-only, no Dart mutation tool configured; manual single-half mutation done instead (remove overridePlatformDefaultLocation: cold-start tests fail; remove onEnter: warm-start tests fail)", mutant_killed: true }
  no_op_deletion: { result: pass, deletion_justified_by_rca: false }
  adjacent_tests: { result: pass, suites_run: [test/routes/trail_detail_screen_retired_redirect_test.dart, test/routes/profile_trail_screen_navigation_test.dart, test/routes/profile_trail_map_navigation_test.dart, test/actions/import_trail_file_test.dart] }
  revert_and_reconfirm: { result: pass, bug_returned_on_revert: true, fixed_on_reapply: true, detail: "git stash of router_provider.dart makes the wiring test fail; pop restores 20/20. Behaviour tests were reverted by config copy, not by git, because the unfixed behaviour lives in the test's mirror of Router.build()." }
  flutter_analyze: no new issues (13 pre-existing infos elsewhere, none in touched files)
  guardrail_verdict: accepted
  not_verified: "On-device. No adb device. (1) whether the reporter's intent has a non-null type; if null, receive_sharing_intent reports it as a url-type media with a content:// string path and importTrailFile would show its import-error toast instead of the sheet. (2) iOS."
files_changed:
  - app/lib/util/inbound_uri_guard.dart (new)
  - app/lib/provider/router_provider.dart
  - app/test/provider/router_inbound_uri_test.dart (new)
suggested_commit_headline: "fix(app): keep Open with file URIs out of the router"
