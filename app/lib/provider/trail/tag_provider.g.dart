// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tag_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(TagNotifier)
final tagProvider = TagNotifierProvider._();

final class TagNotifierProvider
    extends $AsyncNotifierProvider<TagNotifier, List<Tag>> {
  TagNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'tagProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$tagNotifierHash();

  @$internal
  @override
  TagNotifier create() => TagNotifier();
}

String _$tagNotifierHash() => r'a4748c5317e1a7cba3c9665bd5641d196e81f61f';

abstract class _$TagNotifier extends $AsyncNotifier<List<Tag>> {
  FutureOr<List<Tag>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Tag>>, List<Tag>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Tag>>, List<Tag>>,
              AsyncValue<List<Tag>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
