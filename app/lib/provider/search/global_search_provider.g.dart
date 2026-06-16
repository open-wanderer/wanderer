// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'global_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(GlobalSearchNotifier)
final globalSearchProvider = GlobalSearchNotifierProvider._();

final class GlobalSearchNotifierProvider
    extends $NotifierProvider<GlobalSearchNotifier, GlobalSearchState> {
  GlobalSearchNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'globalSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$globalSearchNotifierHash();

  @$internal
  @override
  GlobalSearchNotifier create() => GlobalSearchNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(GlobalSearchState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<GlobalSearchState>(value),
    );
  }
}

String _$globalSearchNotifierHash() =>
    r'25178e1b410f9f1e8ce1d76c66b1358c5fdad50c';

abstract class _$GlobalSearchNotifier extends $Notifier<GlobalSearchState> {
  GlobalSearchState build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<GlobalSearchState, GlobalSearchState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<GlobalSearchState, GlobalSearchState>,
              GlobalSearchState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
