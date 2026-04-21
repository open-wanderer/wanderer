// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_list_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ServerDirectory)
final serverDirectoryProvider = ServerDirectoryProvider._();

final class ServerDirectoryProvider
    extends $AsyncNotifierProvider<ServerDirectory, List<ServerInstance>> {
  ServerDirectoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'serverDirectoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$serverDirectoryHash();

  @$internal
  @override
  ServerDirectory create() => ServerDirectory();
}

String _$serverDirectoryHash() => r'b0ab9cf88ffe63a2f37017926a647955282d4886';

abstract class _$ServerDirectory extends $AsyncNotifier<List<ServerInstance>> {
  FutureOr<List<ServerInstance>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<List<ServerInstance>>, List<ServerInstance>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<List<ServerInstance>>,
                List<ServerInstance>
              >,
              AsyncValue<List<ServerInstance>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
