// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ListSearchNotifier)
final listSearchProvider = ListSearchNotifierProvider._();

final class ListSearchNotifierProvider
    extends $AsyncNotifierProvider<ListSearchNotifier, ListSearchState> {
  ListSearchNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'listSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$listSearchNotifierHash();

  @$internal
  @override
  ListSearchNotifier create() => ListSearchNotifier();
}

String _$listSearchNotifierHash() =>
    r'86678adc874c409735e65d003da63d1b76b725a7';

abstract class _$ListSearchNotifier extends $AsyncNotifier<ListSearchState> {
  FutureOr<ListSearchState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ListSearchState>, ListSearchState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ListSearchState>, ListSearchState>,
              AsyncValue<ListSearchState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
