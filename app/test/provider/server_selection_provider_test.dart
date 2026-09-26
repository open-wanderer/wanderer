import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';

// ---------------------------------------------------------------------------
// setSelectedServer must succeed regardless of whether the servers.json fetch
// that backs `build()` succeeded, failed, or is still in flight — a "Last
// used" entry (quick-260926-ijp) can be tapped in any of those states.
// ---------------------------------------------------------------------------

class _ThrowingServerSelection extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async {
    throw Exception('servers.json fetch failed');
  }
}

class _SucceedingServerSelection extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async {
    return ServerState(const [
      ServerInstance(name: 'W', url: 'https://wanderer.to'),
    ], null);
  }
}

void main() {
  test(
    'setSelectedServer succeeds when the servers.json fetch failed',
    () async {
      final container = ProviderContainer(
        // Riverpod's default AsyncNotifier retry (up to 10 exponential-backoff
        // attempts) would otherwise keep this build in a retrying loading
        // state well past the test's own timeout. This test only cares about
        // setSelectedServer's behavior once the fetch has settled to an
        // error, so retries are disabled here.
        retry: (retryCount, error) => null,
        overrides: [
          serverSelectionProvider.overrideWith(_ThrowingServerSelection.new),
        ],
      );
      addTearDown(container.dispose);
      container.listen(serverSelectionProvider, (_, _) {});

      // Let build() run and settle into an error state before selecting.
      await expectLater(
        container.read(serverSelectionProvider.future),
        throwsException,
      );

      const selected = ServerInstance(url: 'https://self.hosted.example');
      container.read(serverSelectionProvider.notifier).setSelectedServer(selected);

      final state = container.read(serverSelectionProvider);
      expect(state.hasError, isFalse);
      expect(state.value?.availableServers, isEmpty);
      expect(state.value?.selectedServer, selected);
    },
  );

  test('setSelectedServer keeps the fetched availableServers on success', () async {
    final container = ProviderContainer(
      overrides: [
        serverSelectionProvider.overrideWith(_SucceedingServerSelection.new),
      ],
    );
    addTearDown(container.dispose);
    container.listen(serverSelectionProvider, (_, _) {});

    await container.read(serverSelectionProvider.future);

    const selected = ServerInstance(url: 'https://self.hosted.example');
    container.read(serverSelectionProvider.notifier).setSelectedServer(selected);

    final state = container.read(serverSelectionProvider);
    expect(state.hasError, isFalse);
    expect(state.value?.availableServers, hasLength(1));
    expect(state.value?.availableServers.single.name, 'W');
    expect(state.value?.selectedServer, selected);
  });
}
