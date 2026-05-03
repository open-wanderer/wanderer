// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_selection_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ServerSelectionNotifier)
final serverSelectionProvider = ServerSelectionNotifierProvider._();

final class ServerSelectionNotifierProvider
    extends $AsyncNotifierProvider<ServerSelectionNotifier, ServerState> {
  ServerSelectionNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'serverSelectionProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$serverSelectionNotifierHash();

  @$internal
  @override
  ServerSelectionNotifier create() => ServerSelectionNotifier();
}

String _$serverSelectionNotifierHash() =>
    r'fed88e170f18eccdb25c076c038d896040d98fed';

abstract class _$ServerSelectionNotifier extends $AsyncNotifier<ServerState> {
  FutureOr<ServerState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ServerState>, ServerState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ServerState>, ServerState>,
              AsyncValue<ServerState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
