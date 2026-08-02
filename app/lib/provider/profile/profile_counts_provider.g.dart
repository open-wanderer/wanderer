// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_counts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(profileCounts)
final profileCountsProvider = ProfileCountsFamily._();

final class ProfileCountsProvider
    extends
        $FunctionalProvider<
          AsyncValue<ProfileCounts>,
          ProfileCounts,
          FutureOr<ProfileCounts>
        >
    with $FutureModifier<ProfileCounts>, $FutureProvider<ProfileCounts> {
  ProfileCountsProvider._({
    required ProfileCountsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileCountsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileCountsHash();

  @override
  String toString() {
    return r'profileCountsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ProfileCounts> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ProfileCounts> create(Ref ref) {
    final argument = this.argument as String;
    return profileCounts(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ProfileCountsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileCountsHash() => r'b130213f3ba84a9768e5fa4f82bed20ae52cd4fb';

final class ProfileCountsFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ProfileCounts>, String> {
  ProfileCountsFamily._()
    : super(
        retry: null,
        name: r'profileCountsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileCountsProvider call(String actorId) =>
      ProfileCountsProvider._(argument: actorId, from: this);

  @override
  String toString() => r'profileCountsProvider';
}
