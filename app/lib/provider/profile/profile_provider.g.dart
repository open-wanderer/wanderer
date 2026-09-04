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

String _$profileNotifierHash() => r'2b8d7e60cbc91d2745a6ae7c7f44b39b7719f372';

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
/// Reads handle from authProvider.preferredUsername.
/// Cache refreshed on pull-to-refresh AND on any auth change: watching
/// `authProvider` instead of reading it once means this provider rebuilds —
/// and re-fetches — the moment auth resolves to a different session, rather
/// than holding the first account's `Actor` forever. Still no app-resume
/// invalidation.
///
/// Writes every successful fetch through to `UserEntity.actor` and falls back
/// to that cached actor when the fetch fails, so the own-profile screen renders
/// its real layout offline instead of an error page. Only the genuinely
/// network-bound sections (counts, lists, feed) degrade.

@ProviderFor(OwnProfile)
final ownProfileProvider = OwnProfileProvider._();

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername.
/// Cache refreshed on pull-to-refresh AND on any auth change: watching
/// `authProvider` instead of reading it once means this provider rebuilds —
/// and re-fetches — the moment auth resolves to a different session, rather
/// than holding the first account's `Actor` forever. Still no app-resume
/// invalidation.
///
/// Writes every successful fetch through to `UserEntity.actor` and falls back
/// to that cached actor when the fetch fails, so the own-profile screen renders
/// its real layout offline instead of an error page. Only the genuinely
/// network-bound sections (counts, lists, feed) degrade.
final class OwnProfileProvider
    extends $AsyncNotifierProvider<OwnProfile, Actor> {
  /// keepAlive provider — fetches the current user's own profile Actor.
  /// Reads handle from authProvider.preferredUsername.
  /// Cache refreshed on pull-to-refresh AND on any auth change: watching
  /// `authProvider` instead of reading it once means this provider rebuilds —
  /// and re-fetches — the moment auth resolves to a different session, rather
  /// than holding the first account's `Actor` forever. Still no app-resume
  /// invalidation.
  ///
  /// Writes every successful fetch through to `UserEntity.actor` and falls back
  /// to that cached actor when the fetch fails, so the own-profile screen renders
  /// its real layout offline instead of an error page. Only the genuinely
  /// network-bound sections (counts, lists, feed) degrade.
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

String _$ownProfileHash() => r'b1d7fa0d60c2b0b159cd8afed1193d609645a438';

/// keepAlive provider — fetches the current user's own profile Actor.
/// Reads handle from authProvider.preferredUsername.
/// Cache refreshed on pull-to-refresh AND on any auth change: watching
/// `authProvider` instead of reading it once means this provider rebuilds —
/// and re-fetches — the moment auth resolves to a different session, rather
/// than holding the first account's `Actor` forever. Still no app-resume
/// invalidation.
///
/// Writes every successful fetch through to `UserEntity.actor` and falls back
/// to that cached actor when the fetch fails, so the own-profile screen renders
/// its real layout offline instead of an error page. Only the genuinely
/// network-bound sections (counts, lists, feed) degrade.

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
