// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subcategory_preference_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SubcategoryPreferenceNotifier)
final subcategoryPreferenceProvider = SubcategoryPreferenceNotifierProvider._();

final class SubcategoryPreferenceNotifierProvider
    extends
        $AsyncNotifierProvider<
          SubcategoryPreferenceNotifier,
          List<SubcategoryPreference>
        > {
  SubcategoryPreferenceNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'subcategoryPreferenceProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$subcategoryPreferenceNotifierHash();

  @$internal
  @override
  SubcategoryPreferenceNotifier create() => SubcategoryPreferenceNotifier();
}

String _$subcategoryPreferenceNotifierHash() =>
    r'1d8114eb80f944655ce2e9d6b44ae61f206a2ff1';

abstract class _$SubcategoryPreferenceNotifier
    extends $AsyncNotifier<List<SubcategoryPreference>> {
  FutureOr<List<SubcategoryPreference>> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<List<SubcategoryPreference>>,
              List<SubcategoryPreference>
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<List<SubcategoryPreference>>,
                List<SubcategoryPreference>
              >,
              AsyncValue<List<SubcategoryPreference>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
