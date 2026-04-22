// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_selection_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ServerSelection)
final serverSelectionProvider = ServerSelectionProvider._();

final class ServerSelectionProvider
    extends $AsyncNotifierProvider<ServerSelection, ServerState> {
  ServerSelectionProvider._()
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
  String debugGetCreateSourceHash() => _$serverSelectionHash();

  @$internal
  @override
  ServerSelection create() => ServerSelection();
}

String _$serverSelectionHash() => r'd3b0f5e17aec17c883d5fdef3a668fe32f9ff3d1';

abstract class _$ServerSelection extends $AsyncNotifier<ServerState> {
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
