// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list_filter_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ListFilterNotifier)
final listFilterProvider = ListFilterNotifierProvider._();

final class ListFilterNotifierProvider
    extends $AsyncNotifierProvider<ListFilterNotifier, ListFilter> {
  ListFilterNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'listFilterProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$listFilterNotifierHash();

  @$internal
  @override
  ListFilterNotifier create() => ListFilterNotifier();
}

String _$listFilterNotifierHash() =>
    r'94efc1a2395a5d280962710468bec242743a2965';

abstract class _$ListFilterNotifier extends $AsyncNotifier<ListFilter> {
  FutureOr<ListFilter> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ListFilter>, ListFilter>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ListFilter>, ListFilter>,
              AsyncValue<ListFilter>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
