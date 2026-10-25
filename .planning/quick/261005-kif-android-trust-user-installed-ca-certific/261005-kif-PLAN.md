---
phase: quick-261005-kif
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - /Users/christianbeutel/Documents/svelte/wanderer/app/lib/services/user_ca_trust.dart
  - /Users/christianbeutel/Documents/svelte/wanderer/app/test/services/user_ca_trust_test.dart
  - /Users/christianbeutel/Documents/svelte/wanderer/app/android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt
  - /Users/christianbeutel/Documents/svelte/wanderer/app/lib/main.dart
autonomous: true
requirements: [UCA-01, UCA-02, UCA-03, UCA-04, UCA-05]

must_haves:
  truths:
    - "UCA-01: On Android, MainActivity answers the method `getUserCaCertificates` on channel `com.openwanderer.wanderer/user_ca` with the DER bytes of every AndroidCAStore certificate whose alias starts with `user:`. Any failure returns an empty list, and the existing onCreate MapLibre code is unchanged."
    - "UCA-02: derToPem turns DER bytes into a standard PEM block: a `-----BEGIN CERTIFICATE-----` line, base64 body lines of at most 64 chars, and a `-----END CERTIFICATE-----` line. The base64 body decodes back to the input bytes."
    - "UCA-03: trustUserCaCertificates adds each returned certificate to SecurityContext.defaultContext. One invalid or duplicate certificate (TlsException or anything else) is skipped with a debugPrint, and the rest are still added."
    - "UCA-04: On non-Android platforms the channel is never invoked. A missing handler (MissingPluginException), a PlatformException, a null result, or a timeout returns 0 without throwing, so app start is never blocked or crashed."
    - "UCA-05: main() awaits trustUserCaCertificates() right after WidgetsFlutterBinding.ensureInitialized() and before TileProxyServer.start and any HTTP client creation. A comment says certificates are only picked up at app launch."
    - "network_security_config.xml is untouched, because it does not affect dart:io."
  artifacts:
    - path: "/Users/christianbeutel/Documents/svelte/wanderer/app/lib/services/user_ca_trust.dart"
      provides: "userCaChannel, derToPem, trustDerCertificates, trustUserCaCertificates"
      exports: ["userCaChannel", "derToPem", "trustDerCertificates", "trustUserCaCertificates"]
      contains: "com.openwanderer.wanderer/user_ca"
    - path: "/Users/christianbeutel/Documents/svelte/wanderer/app/test/services/user_ca_trust_test.dart"
      provides: "Unit tests for DER->PEM, per-cert error isolation, platform gate, and channel-failure swallowing"
      min_lines: 80
    - path: "/Users/christianbeutel/Documents/svelte/wanderer/app/android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt"
      provides: "configureFlutterEngine registering the user_ca MethodChannel"
      contains: "AndroidCAStore"
    - path: "/Users/christianbeutel/Documents/svelte/wanderer/app/lib/main.dart"
      provides: "Startup call before TileProxyServer.start"
      contains: "await trustUserCaCertificates()"
  key_links:
    - from: "app/lib/services/user_ca_trust.dart userCaChannel"
      to: "MainActivity.kt USER_CA_CHANNEL"
      via: "identical channel name string and method name `getUserCaCertificates` on both sides"
      pattern: "com\\.openwanderer\\.wanderer/user_ca"
    - from: "app/lib/main.dart main()"
      to: "trustUserCaCertificates"
      via: "awaited between ensureInitialized() and TileProxyServer.start(store)"
      pattern: "await trustUserCaCertificates\\(\\)"
    - from: "trustDerCertificates"
      to: "SecurityContext.setTrustedCertificatesBytes"
      via: "PEM bytes from derToPem, one try/catch per certificate"
      pattern: "setTrustedCertificatesBytes\\("
---

<objective>
Fix GitHub issue #1255. A user's self-hosted Wanderer sits behind nginx with a certificate signed by their own root CA, and that CA is installed on the Android phone as a user CA. Login shows "Network connection issue" and the request never reaches the server. Root cause: dart:io (BoringSSL) on Android trusts only the system CA store. It ignores user-installed CAs, and network_security_config has no effect on it. Every HTTP path in the app goes through dart:io: Dio (`api_provider.dart`), CachedNetworkImageProvider (flutter_cache_manager -> package:http -> dart:io), flutter_html images, and TileProxyServer's HttpClient.

After this plan: at startup on Android, MainActivity exposes the user-installed CA certificates over a MethodChannel. Dart converts each one from DER to PEM and adds it to `SecurityContext.defaultContext`, so every dart:io client trusts them in addition to the system roots. A failure anywhere in that path is logged and ignored. It never blocks or crashes app start.

The design is user-approved and locked. Do not revisit it. network_security_config.xml stays as is.

Purpose: Self-hosted instances using a private root CA must be reachable from the Android app.
Output: New `user_ca_trust.dart` service plus unit tests, a MethodChannel in MainActivity, and the startup call in main().

Out of scope (do NOT implement): picking up certificates installed while the app is running (the user accepted launch-only pickup), iOS changes, changes to network_security_config.xml, and any `badCertificateCallback` or verification bypass.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@/Users/christianbeutel/Documents/svelte/wanderer/CLAUDE.md
@/Users/christianbeutel/Documents/svelte/wanderer/app/android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/main.dart
@/Users/christianbeutel/Documents/svelte/wanderer/app/test/services/tile_proxy_identity_test.dart

Repo facts (verified during planning, base commit `6cd505a05` on `feature/app`):
- The app has no existing MethodChannel usage. This is the first one, so there is no in-repo pattern to copy beyond the standard Flutter API.
- `analysis_options.yaml` includes only `package:flutter_lints/flutter.yaml` plus riverpod_lint. `avoid_catches_without_on_clauses` is NOT enabled.
- `dangling_library_doc_comments` IS active. The file-level convention is a `///` doc comment block followed by a bare `library;` directive, then the imports (see `lib/services/tile_proxy_identity.dart` lines 1-8).
- `flutter analyze` is NOT clean repo-wide and exits 1 on infos. There are 5 pre-existing infos in `lib/` (actor_entity, category_entity, navigation_stats_provider, local_photo_store, util/local/id) and more in `vendor/`. They are out of scope and must not be fixed. Analyze gates in this plan therefore target the touched files explicitly.
- No code in `app/lib` sets a custom SecurityContext or badCertificateCallback, so every dart:io client uses `SecurityContext.defaultContext`.
- `openssl` (OpenSSL 3.2.1) is on the PATH. This command was verified to work: `openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -keyout /dev/null -subj "/CN=wanderer-test-user-ca" -days 36500 -outform DER | base64` (about 550 base64 chars).
- Test style: plain `flutter_test`, `group`/`test`, and a header comment explaining what is tested and why (see tile_proxy_identity_test.dart).

Working rules from the user (non-negotiable):
- Work directly on branch `feature/app`. Commit locally and do NOT push.
- Commit messages are a single headline line. No body and no Co-Authored-By line. This user preference overrides any default attribution instructions.
- Do NOT run `flutter build`, `./gradlew`, or `adb install`. The user builds and installs. Verify only with `flutter analyze` (on the touched files) and `flutter test`.
- Comments explain why, matching the existing MainActivity.kt and main.dart comment style (see the MapLibre.setConnected and _raiseAmbientTileCacheSize comments).
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Dart user-CA trust service with DER->PEM conversion and per-cert isolation, test-first</name>
  <files>/Users/christianbeutel/Documents/svelte/wanderer/app/lib/services/user_ca_trust.dart, /Users/christianbeutel/Documents/svelte/wanderer/app/test/services/user_ca_trust_test.dart</files>
  <behavior>
    - derToPem(bytes) starts with `-----BEGIN CERTIFICATE-----\n` and ends with `-----END CERTIFICATE-----\n`.
    - For a 130-byte input (176 base64 chars), body lines are 64, 64, and 48 chars long. Every body line except the last is exactly 64 chars, and none is longer.
    - Joining the body lines and running base64Decode returns exactly the input bytes.
    - trustDerCertificates([validDer], SecurityContext(withTrustedRoots: false)) returns 1.
    - trustDerCertificates([garbage, validDer], ctx) returns 1 and does not throw. The invalid first cert does not abort the second. Garbage is e.g. Uint8List.fromList([1, 2, 3, 4]).
    - trustDerCertificates([], ctx) returns 0, and trustDerCertificates([garbage], ctx) returns 0 without throwing.
    - trustUserCaCertificates(isAndroid: false, ...) returns 0, and the mock channel handler is invoked 0 times.
    - Called with no isAndroid argument on the macOS test host, trustUserCaCertificates returns 0 and the handler is invoked 0 times. This proves the default platform gate.
    - With isAndroid: true and a mock handler that returns [garbage, validDer] (as Uint8List) for method `getUserCaCertificates`, it returns 1 into an injected fresh SecurityContext(withTrustedRoots: false). The handler is called exactly once with that method name.
    - With isAndroid: true and a handler that throws PlatformException, it completes normally with 0.
    - With isAndroid: true and no handler registered (MissingPluginException), it completes normally with 0.
    - With isAndroid: true and a handler that returns null, it returns 0.
    - With isAndroid: true and a handler whose list mixes a non-Uint8List element (e.g. the String 'x') with validDer, it returns 1. Non-bytes elements are skipped and do not throw.
  </behavior>
  <action>
RED: Create `app/test/services/user_ca_trust_test.dart` covering every behavior above. Start with a header comment in the style of tile_proxy_identity_test.dart. It should say the tests cover the pure DER->PEM conversion, per-certificate failure isolation, and the platform and channel guards, using a mocked channel and a fresh SecurityContext so the shared defaultContext is never touched. Call `TestWidgetsFlutterBinding.ensureInitialized()` at the top of main().

Build the valid-certificate fixture once. Run the openssl command from the context section, then paste its base64 output into the test as a `const String` (named e.g. `_testCaDerBase64`) and decode it with `base64Decode`. Do not commit a separate .der/.pem fixture file, and do not shell out at test time.

Mock the channel with `TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(userCaChannel, handler)`. Count invocations and record method names in the handler. In tearDown, reset the handler with `setMockMethodCallHandler(userCaChannel, null)`. Use plain `test()`, not `testWidgets`, because the timeout uses a real timer. Every case that passes a SecurityContext must pass a fresh `SecurityContext(withTrustedRoots: false)`. No test may mutate `SecurityContext.defaultContext`.

Run the tests. They must fail because the source file does not exist yet.

GREEN: Create `app/lib/services/user_ca_trust.dart`. Follow the repo's file-comment convention: a `///` doc block, then a bare `library;` directive, then imports, as in tile_proxy_identity.dart. The doc block explains why the file exists:
- dart:io's BoringSSL on Android trusts only the system CA store, and network_security_config does not apply to it.
- So self-hosted instances behind a user-installed private root CA fail TLS (#1255).
- Native code hands over the user CAs at launch, and they are added to SecurityContext.defaultContext, which every dart:io client (Dio, image caches, the tile proxy) uses.
- A CA installed while the app runs takes effect only after a restart (accepted tradeoff).

Imports: `dart:async`, `dart:convert`, `dart:io`, `dart:typed_data` (drop it if the analyzer reports unnecessary_import), `package:flutter/foundation.dart`, and `package:flutter/services.dart`. The service must contain:

- A top-level `const MethodChannel userCaChannel = MethodChannel('com.openwanderer.wanderer/user_ca');`. Its doc comment says the name must match `USER_CA_CHANNEL` in MainActivity.kt. Also add a private const for the method name `getUserCaCertificates`.
- `@visibleForTesting String derToPem(List<int> der)`. It base64-encodes with `base64Encode` and splits the result into 64-char lines. It returns `-----BEGIN CERTIFICATE-----\n`, then each line followed by `\n`, then `-----END CERTIFICATE-----\n`. The PEM conversion is per the locked design ("convert DER to PEM to be safe"): setTrustedCertificatesBytes documents PEM/PKCS12 input, not raw DER.
- `@visibleForTesting int trustDerCertificates(Iterable<List<int>> ders, SecurityContext context)`. For each DER entry, call `context.setTrustedCertificatesBytes(utf8.encode(derToPem(der)))` inside its own try/catch and count successes.
  - The catch has no `on` clause, so a TlsException from a duplicate or invalid cert, or anything else, is caught. It logs `debugPrint('user CA trust: skipped a certificate: $error')` and continues. Per the locked design, one bad cert must not abort the others.
  - Never log the certificate bytes. Return the count.
- `Future<int> trustUserCaCertificates({bool? isAndroid, MethodChannel channel = userCaChannel, SecurityContext? context, Duration timeout = const Duration(seconds: 2)})`:
  - If `isAndroid ?? Platform.isAndroid` is false, return 0 immediately without touching the channel.
  - Otherwise call `channel.invokeMethod<List<Object?>>(method).timeout(timeout)` inside a try/catch with no `on` clause. That one catch swallows MissingPluginException, PlatformException, TimeoutException, and anything else, logs with debugPrint, and returns 0. Per the locked design, channel errors never block or crash app start. The 2 s timeout is a discretionary addition. It turns "never block app start" into a guarantee even if the native handler somehow never replies.
  - Filter the result with `(raw ?? const <Object?>[]).whereType<Uint8List>().toList()`. Do not use `invokeListMethod<Uint8List>`: its lazy `cast` would throw on a bad element outside the per-cert try.
  - Return 0 if the filtered list is empty. Otherwise call `trustDerCertificates(certs, context ?? SecurityContext.defaultContext)`, debugPrint a one-line summary (e.g. `user CA trust: trusted $added of ${certs.length} user certificate(s)`), and return the count.
  - Give the function a short doc comment saying it is best-effort and never throws. Use explicit return types everywhere.

Run `dart format` on both files. Run the tests until they pass, then run `flutter analyze` on both files and confirm it reports no issues.

Commit both files with exactly this message, a single line with no body: `feat(app): add a dart:io trust helper for user-installed CA certificates`
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/services/user_ca_trust_test.dart && flutter analyze lib/services/user_ca_trust.dart test/services/user_ca_trust_test.dart</automated>
  </verify>
  <done>user_ca_trust_test.dart passes every behavior case listed above. flutter analyze reports "No issues found!" for both files. The service exports userCaChannel, derToPem, trustDerCertificates, and trustUserCaCertificates with the signatures above. One commit exists with the headline-only message.</done>
</task>

<task type="auto">
  <name>Task 2: Register the user_ca MethodChannel in MainActivity and call the trust helper first thing in main()</name>
  <files>/Users/christianbeutel/Documents/svelte/wanderer/app/android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt, /Users/christianbeutel/Documents/svelte/wanderer/app/lib/main.dart</files>
  <action>
**MainActivity.kt** (per the locked design: additive only, existing onCreate MapLibre code left byte-identical):

1. Add these imports alongside the existing ones, keeping all current imports: `android.util.Log`, `io.flutter.embedding.engine.FlutterEngine`, `io.flutter.plugin.common.MethodChannel`, `java.security.KeyStore`.
2. Below onCreate, add `override fun configureFlutterEngine(flutterEngine: FlutterEngine)`.
   - Its FIRST statement must be `super.configureFlutterEngine(flutterEngine)`. The super call registers GeneratedPluginRegistrant, and leaving it out silently breaks every plugin.
   - Then call `MethodChannel(flutterEngine.dartExecutor.binaryMessenger, USER_CA_CHANNEL).setMethodCallHandler { call, result -> ... }`. When `call.method == "getUserCaCertificates"`, reply with `result.success(readUserCaCertificates())`. Otherwise reply with `result.notImplemented()`.
   - FlutterActivity calls configureFlutterEngine before the Dart entrypoint runs, so the handler is registered by the time main() invokes it.
3. Add `private fun readUserCaCertificates(): List<ByteArray>`. Inside an outer `try`:
   - Get `KeyStore.getInstance("AndroidCAStore")` into a local `keyStore` and call `keyStore.load(null)`.
   - Take `keyStore.aliases().toList()`, keep only aliases where `startsWith("user:")`, and map each with `mapNotNull { alias -> runCatching { keyStore.getCertificate(alias)?.encoded }.getOrNull() }`. One unreadable cert is dropped instead of emptying the whole list. This per-cert isolation is discretionary and mirrors the Dart side.
   - `catch (e: Exception)` logs `Log.w(TAG, "Could not read user CA certificates", e)` and returns `emptyList()`. Per the locked design, any failure returns an empty list.
   - StandardMessageCodec encodes `List<ByteArray>` as a Dart `List<Object?>` of `Uint8List`.
4. Add a `companion object` with `private const val USER_CA_CHANNEL = "com.openwanderer.wanderer/user_ca"` and `private const val TAG = "UserCaTrust"`. The channel string must exactly equal the Dart `userCaChannel` name.
5. Above configureFlutterEngine, add a why-comment in the existing MainActivity voice (`//` lines, like the MapLibre comment). It should explain:
   - Dart's TLS stack (BoringSSL in dart:io) reads only the system CA store on Android and ignores network_security_config, so user-installed CAs never reach it (#1255).
   - This hands their DER bytes to Dart at startup, where user_ca_trust.dart adds them to SecurityContext.defaultContext.
   - Only `user:` aliases are read because system CAs are already trusted.
   - Failures return an empty list because a missing CA must never stop the app from starting.

Do not touch onCreate, and do not modify network_security_config.xml.

**main.dart** (per the locked design):

1. Add `import 'package:wanderer/services/user_ca_trust.dart';` to the existing `package:wanderer/services/...` imports.
2. Put `await trustUserCaCertificates();` immediately after `WidgetsFlutterBinding.ensureInitialized();` and before `getApplicationDocumentsDirectory()`. That places it ahead of `TileProxyServer.start(store)` and of any Dio or HttpClient creation (those are built lazily inside ProviderScope after runApp).
3. Put a short `//` why-comment above the call, matching the tone of the `_raiseAmbientTileCacheSize` comment. It should say:
   - On Android, dart:io trusts only system CAs, so self-hosted instances behind a user-installed private root CA fail TLS (#1255).
   - The call must run before the tile proxy or any HTTP client connects.
   - Certificates are read once per launch, so a CA installed while the app runs needs a restart.
   - It is best-effort and never throws.

Run `dart format lib/main.dart`. Run `flutter analyze` on the touched Dart files only. The repo-wide run has pre-existing infos that are out of scope; do not fix them. Then run `flutter test test/services/user_ca_trust_test.dart`. Do NOT run flutter build, gradle, or adb. The Kotlin compiles only when the user builds, so double-check the four imports and the `override` signature by eye.

Commit both files with exactly this message, a single line with no body: `fix(app): trust user-installed CA certificates on Android`
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter analyze lib/main.dart lib/services/user_ca_trust.dart test/services/user_ca_trust_test.dart && flutter test test/services/user_ca_trust_test.dart && K=android/app/src/main/kotlin/com/openwanderer/wanderer/MainActivity.kt && grep -q 'super.configureFlutterEngine(flutterEngine)' $K && grep -q 'import io.flutter.embedding.engine.FlutterEngine' $K && grep -q 'import io.flutter.plugin.common.MethodChannel' $K && grep -q 'import java.security.KeyStore' $K && grep -q 'import android.util.Log' $K && grep -q '"AndroidCAStore"' $K && grep -q 'startsWith("user:")' $K && grep -q '"getUserCaCertificates"' $K && grep -q '"com.openwanderer.wanderer/user_ca"' $K && grep -q 'MapLibre.setConnected(true)' $K && grep -q "'com.openwanderer.wanderer/user_ca'" lib/services/user_ca_trust.dart && grep -q "'getUserCaCertificates'" lib/services/user_ca_trust.dart && test "$(git diff 6cd505a05 -- $K | grep -c '^-[^-]')" = 0 && git diff --quiet 6cd505a05 -- android/app/src/main/res/xml/network_security_config.xml && awk '/ensureInitialized\(\);/{a=NR} /await trustUserCaCertificates\(\);/{b=NR} /TileProxyServer\.start\(/{c=NR} END{exit !(a>0 && a<b && b<c)}' lib/main.dart && echo OK</automated>
  </verify>
  <done>flutter analyze reports no issues for the touched Dart files, and the user_ca_trust tests pass. The MainActivity.kt diff against 6cd505a05 has zero removed lines (additive only, onCreate intact). It contains the super call, the four imports, the AndroidCAStore and `user:` filter, and a channel and method name that match the Dart side. network_security_config.xml is unchanged. In main.dart, `await trustUserCaCertificates();` sits between ensureInitialized() and TileProxyServer.start(, with a why-comment that mentions launch-only pickup. One commit exists with the headline-only message, and nothing is pushed.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Device user CA store -> app TLS trust | Certificates the device user installed become trust anchors for all dart:io TLS in the app |
| Native (Kotlin) -> Dart over MethodChannel | Byte lists cross the platform channel at startup; Dart must tolerate malformed or unexpected payloads |
| App startup path | A failure in the new code sits on the critical path before runApp |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-kif-01 | Spoofing | user_ca_trust.dart adding user CAs to defaultContext | accept | Deliberate, user-approved widening of trust. A user CA can only be installed by the device owner (it requires the device credential), and this is the trust model the reporter needs for a self-hosted private CA. Chrome on Android trusts user CAs the same way. Native Android networking keeps its stricter default because network_security_config is unchanged. |
| T-kif-02 | Tampering | trust scope | mitigate | Only `user:` aliases are read (system CAs are already trusted by dart:io). Certificates are added on top of the system roots, never replacing them. No `badCertificateCallback`, no verification bypass, and hostname verification stays fully on. |
| T-kif-03 | Denial of Service | main() startup | mitigate | The channel call runs under a try/catch with no `on` clause plus a 2 s timeout, and returns 0 on any failure. Each certificate gets its own try/catch. Kotlin catches everything and returns an empty list. Non-Android platforms never invoke the channel. Covered by the Task 1 unit tests. |
| T-kif-04 | Tampering | MethodChannel payload | mitigate | `whereType<Uint8List>()` drops non-bytes elements. An invalid DER produces a TlsException that is caught per certificate. A null result is treated as empty. Covered by the Task 1 unit tests. |
| T-kif-05 | Information Disclosure | debugPrint logging | mitigate | Logs contain counts and exception messages only, never certificate bytes. Kotlin logs only a warning with the exception. |
</threat_model>

<verification>
- `cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test test/services/user_ca_trust_test.dart` passes.
- `flutter analyze lib/main.dart lib/services/user_ca_trust.dart test/services/user_ca_trust_test.dart` reports no issues. The pre-existing infos elsewhere in lib/ and vendor/ are unchanged and out of scope.
- The Task 2 automated gate prints OK. It covers additive-only MainActivity, matching channel and method names, untouched network_security_config.xml, and call ordering in main().
- `git log --format=%B -n 2` shows two single-line messages with no body and no Co-Authored-By line. Nothing is pushed.
- Handoff for the user: build and install on Android, install a private root CA as a user CA (Settings -> Security -> Encryption & credentials -> Install a certificate -> CA certificate), restart the app, and log in to an instance served with a certificate from that CA. Login succeeds, and `adb logcat | grep "user CA trust"` shows `trusted N of N`. The executor does not do this. It goes in the SUMMARY as a user-side check.
</verification>

<success_criteria>
- On Android, every user-installed CA certificate is added to dart:io's default trust at launch, so Dio, image loading, and the tile proxy can reach a self-hosted instance using a private root CA (#1255).
- No path in the new code can throw out of main() or block startup beyond the 2 s timeout.
- The existing MainActivity onCreate MapLibre code and network_security_config.xml are unchanged.
- Unit tests cover DER->PEM conversion, per-certificate failure isolation, the platform gate, and channel-failure handling.
- Two local commits on feature/app with headline-only messages.
</success_criteria>

<output>
Create `/Users/christianbeutel/Documents/svelte/wanderer/.planning/quick/261005-kif-android-trust-user-installed-ca-certific/261005-kif-SUMMARY.md` when done. Include the on-device user-side check from the verification section as a handoff note.
</output>
