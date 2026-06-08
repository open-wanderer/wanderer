// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Auto-dispose family provider — fetches any user's profile by handle.
/// Call site: `ref.watch(profileProvider(handle))`
/// Auto-disposes when no longer watched; re-fetches on each navigation.

@ProviderFor(ProfileNotifier)
final profileProvider = ProfileNotifierFamily._();

/// Auto-dispose family provider — fetches any user's profile by handle.
/// Call site: `ref.watch(profileProvider(handle))`
/// Auto-disposes when no longer watched; re-fetches on each navigation.
final class ProfileNotifierProvider
    extends $AsyncNotifierProvider<ProfileNotifier, Actor> {
  /// Auto-dispose family provider — fetches any user's profile by handle.
  /// Call site: `ref.watch(profileProvider(handle))`
  /// Auto-disposes when no longer watched; re-fetches on each navigation.
  ProfileNotifierProvider._({
    required ProfileNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileNotifierHash();

  @override
  String toString() {
    return r'profileProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileNotifier create() => ProfileNotifier();

  @override
  bool operator ==(Object other) {
    return other is ProfileNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileNotifierHash() => r'a9d771b94f111b48a5acab5709afbf435b6b283d';

/// Auto-dispose family provider — fetches any user's profile by handle.
/// Call site: `ref.watch(profileProvider(handle))`
/// Auto-disposes when no longer watched; re-fetches on each navigation.

final class ProfileNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileNotifier,
          AsyncValue<Actor>,
          Actor,
          FutureOr<Actor>,
          String
        > {
  ProfileNotifierFamily._()
    : super(
        retry: null,
        name: r'profileProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Auto-dispose family provider — fetches any user's profile by handle.
  /// Call site: `ref.watch(profileProvider(handle))`
  /// Auto-disposes when no longer watched; re-fetches on each navigation.

  ProfileNotifierProvider call(String handle) =>
      ProfileNotifierProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileProvider';
}

/// Auto-dispose family provider — fetches any user's profile by handle.
/// Call site: `ref.watch(profileProvider(handle))`
/// Auto-disposes when no longer watched; re-fetches on each navigation.

abstract class _$ProfileNotifier extends $AsyncNotifier<Actor> {
  late final _$args = ref.$arg as String;
  String get handle => _$args;

  FutureOr<Actor> build(String handle);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Actor>, Actor>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Actor>, Actor>,
              AsyncValue<Actor>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername (per D-02).
/// Cache refreshed only on pull-to-refresh (Phase 2); no app-resume invalidation (D-03).

@ProviderFor(OwnProfile)
final ownProfileProvider = OwnProfileProvider._();

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername (per D-02).
/// Cache refreshed only on pull-to-refresh (Phase 2); no app-resume invalidation (D-03).
final class OwnProfileProvider
    extends $AsyncNotifierProvider<OwnProfile, Actor> {
  /// keepAlive provider — fetches the current user's own profile Actor.
  /// Reads handle from authProvider.preferredUsername (per D-02).
  /// Cache refreshed only on pull-to-refresh (Phase 2); no app-resume invalidation (D-03).
  OwnProfileProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'ownProfileProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$ownProfileHash();

  @$internal
  @override
  OwnProfile create() => OwnProfile();
}

String _$ownProfileHash() => r'545e771d72c533a8121a7fcde8f3863740882abf';

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername (per D-02).
/// Cache refreshed only on pull-to-refresh (Phase 2); no app-resume invalidation (D-03).

abstract class _$OwnProfile extends $AsyncNotifier<Actor> {
  FutureOr<Actor> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Actor>, Actor>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Actor>, Actor>,
              AsyncValue<Actor>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
