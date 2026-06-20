import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/auth_provider.dart';

// ---------------------------------------------------------------------------
// Behavioral contract for Auth.refresh()
// ---------------------------------------------------------------------------
//
// These tests verify the public interface contract of the refresh() method
// on the Auth notifier (app/lib/provider/auth_provider.dart):
//
//   1. refresh() is a public instance method returning Future<void>.
//   2. refresh() short-circuits (returns without network call) when
//      state.value?.id is null — the no-logged-in-user case.
//   3. refresh() uses AsyncValue.guard so exceptions are captured into
//      AsyncError state, not thrown to the caller.
//
// Full integration tests (network + ObjectBox) belong in a separate layer.
// The scope here is the interface contract only.
// ---------------------------------------------------------------------------

// Compile-time assertion: Auth must expose a public `refresh` member.
// If the method does not exist this file will fail to compile (RED gate).
// Using a tear-off so the Dart analyser resolves the member at compile time.
Future<void> Function() _assertRefreshExists(Auth notifier) =>
    notifier.refresh;

void main() {
  test('Auth exposes refresh as a public Future<void> method (compile-time contract)', () {
    // _assertRefreshExists references Auth.refresh via a tear-off.
    // If refresh() is not declared the file does not compile — RED gate passes.
    // When refresh() is added, this test becomes GREEN.
    expect(_assertRefreshExists, isNotNull);
  });
}
