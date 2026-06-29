// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'category_preference_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(CategoryPreferenceNotifier)
final categoryPreferenceProvider = CategoryPreferenceNotifierProvider._();

final class CategoryPreferenceNotifierProvider
    extends
        $AsyncNotifierProvider<
          CategoryPreferenceNotifier,
          List<CategoryPreference>
        > {
  CategoryPreferenceNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'categoryPreferenceProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$categoryPreferenceNotifierHash();

  @$internal
  @override
  CategoryPreferenceNotifier create() => CategoryPreferenceNotifier();
}

String _$categoryPreferenceNotifierHash() =>
    r'819a895d5d5e92ac4c78a8c064de2a3eaa374b2d';

abstract class _$CategoryPreferenceNotifier
    extends $AsyncNotifier<List<CategoryPreference>> {
  FutureOr<List<CategoryPreference>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<List<CategoryPreference>>,
              List<CategoryPreference>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<List<CategoryPreference>>,
                List<CategoryPreference>
              >,
              AsyncValue<List<CategoryPreference>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
