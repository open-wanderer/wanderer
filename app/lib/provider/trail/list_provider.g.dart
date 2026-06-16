// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'list_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ListNotifier)
final listProvider = ListNotifierFamily._();

final class ListNotifierProvider
    extends $AsyncNotifierProvider<ListNotifier, WandererList> {
  ListNotifierProvider._({
    required ListNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'listProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$listNotifierHash();

  @override
  String toString() {
    return r'listProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ListNotifier create() => ListNotifier();

  @override
  bool operator ==(Object other) {
    return other is ListNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$listNotifierHash() => r'17f924ea84b15c7c98fffff0f776df403d70207d';

final class ListNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ListNotifier,
          AsyncValue<WandererList>,
          WandererList,
          FutureOr<WandererList>,
          String
        > {
  ListNotifierFamily._()
    : super(
        retry: null,
        name: r'listProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ListNotifierProvider call(String id) =>
      ListNotifierProvider._(argument: id, from: this);

  @override
  String toString() => r'listProvider';
}

abstract class _$ListNotifier extends $AsyncNotifier<WandererList> {
  late final _$args = ref.$arg as String;
  String get id => _$args;

  FutureOr<WandererList> build(String id);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<WandererList>, WandererList>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<WandererList>, WandererList>,
              AsyncValue<WandererList>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
