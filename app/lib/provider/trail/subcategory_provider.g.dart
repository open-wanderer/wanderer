// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subcategory_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SubcategoryNotifier)
final subcategoryProvider = SubcategoryNotifierProvider._();

final class SubcategoryNotifierProvider
    extends $NotifierProvider<SubcategoryNotifier, List<Subcategory>> {
  SubcategoryNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'subcategoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$subcategoryNotifierHash();

  @$internal
  @override
  SubcategoryNotifier create() => SubcategoryNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<Subcategory> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<Subcategory>>(value),
    );
  }
}

String _$subcategoryNotifierHash() =>
    r'98d0d5d9294c57105b898ecf9e22046a962a7b3c';

abstract class _$SubcategoryNotifier extends $Notifier<List<Subcategory>> {
  List<Subcategory> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<Subcategory>, List<Subcategory>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<Subcategory>, List<Subcategory>>,
              List<Subcategory>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
