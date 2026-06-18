// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list_filter_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ListFilterNotifier)
final listFilterProvider = ListFilterNotifierFamily._();

final class ListFilterNotifierProvider
    extends $AsyncNotifierProvider<ListFilterNotifier, ListFilter> {
  ListFilterNotifierProvider._({
    required ListFilterNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'listFilterProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$listFilterNotifierHash();

  @override
  String toString() {
    return r'listFilterProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ListFilterNotifier create() => ListFilterNotifier();

  @override
  bool operator ==(Object other) {
    return other is ListFilterNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$listFilterNotifierHash() =>
    r'94efc1a2395a5d280962710468bec242743a2965';

final class ListFilterNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ListFilterNotifier,
          AsyncValue<ListFilter>,
          ListFilter,
          FutureOr<ListFilter>,
          String
        > {
  ListFilterNotifierFamily._()
    : super(
        retry: null,
        name: r'listFilterProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ListFilterNotifierProvider call(String filterId) =>
      ListFilterNotifierProvider._(argument: filterId, from: this);

  @override
  String toString() => r'listFilterProvider';
}

abstract class _$ListFilterNotifier extends $AsyncNotifier<ListFilter> {
  late final _$args = ref.$arg as String;
  String get filterId => _$args;

  FutureOr<ListFilter> build(String filterId);
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
    element.handleCreate(ref, () => build(_$args));
  }
}
