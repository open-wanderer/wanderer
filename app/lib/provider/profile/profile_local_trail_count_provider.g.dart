// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_local_trail_count_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// How many of [handle]'s trails are on THIS device: the signed-in account's
/// not-yet-uploaded captures plus the downloaded trails it authored itself.
///
/// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
/// profile shows offline is the number of trails `/profile/<handle>/trails`
/// can actually render offline — no second, differently-scoped query that
/// could disagree with the list it links to.
///
/// Null — not zero — for anyone else's handle and for a signed-out store:
/// "this device holds none of their trails" is not an answer worth showing,
/// while a genuine zero for the signed-in hiker is.
///
/// The own-handle
/// test and the actor id are both re-derived here from fresh
/// `authProvider`/`currentAccountId` reads rather than passed in, which is
/// The "never from a cached value" — a stale actor id would pair this
/// account with the previous one's trails.

@ProviderFor(profileLocalTrailCount)
final profileLocalTrailCountProvider = ProfileLocalTrailCountFamily._();

/// How many of [handle]'s trails are on THIS device: the signed-in account's
/// not-yet-uploaded captures plus the downloaded trails it authored itself.
///
/// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
/// profile shows offline is the number of trails `/profile/<handle>/trails`
/// can actually render offline — no second, differently-scoped query that
/// could disagree with the list it links to.
///
/// Null — not zero — for anyone else's handle and for a signed-out store:
/// "this device holds none of their trails" is not an answer worth showing,
/// while a genuine zero for the signed-in hiker is.
///
/// The own-handle
/// test and the actor id are both re-derived here from fresh
/// `authProvider`/`currentAccountId` reads rather than passed in, which is
/// The "never from a cached value" — a stale actor id would pair this
/// account with the previous one's trails.

final class ProfileLocalTrailCountProvider
    extends $FunctionalProvider<int?, int?, int?>
    with $Provider<int?> {
  /// How many of [handle]'s trails are on THIS device: the signed-in account's
  /// not-yet-uploaded captures plus the downloaded trails it authored itself.
  ///
  /// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
  /// profile shows offline is the number of trails `/profile/<handle>/trails`
  /// can actually render offline — no second, differently-scoped query that
  /// could disagree with the list it links to.
  ///
  /// Null — not zero — for anyone else's handle and for a signed-out store:
  /// "this device holds none of their trails" is not an answer worth showing,
  /// while a genuine zero for the signed-in hiker is.
  ///
  /// The own-handle
  /// test and the actor id are both re-derived here from fresh
  /// `authProvider`/`currentAccountId` reads rather than passed in, which is
  /// The "never from a cached value" — a stale actor id would pair this
  /// account with the previous one's trails.
  ProfileLocalTrailCountProvider._({
    required ProfileLocalTrailCountFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'profileLocalTrailCountProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileLocalTrailCountHash();

  @override
  String toString() {
    return r'profileLocalTrailCountProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<int?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  int? create(Ref ref) {
    final argument = this.argument as String;
    return profileLocalTrailCount(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(int? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<int?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is ProfileLocalTrailCountProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileLocalTrailCountHash() =>
    r'9e26532bf2cd4e869bb7a54e3094a8e17c0c72db';

/// How many of [handle]'s trails are on THIS device: the signed-in account's
/// not-yet-uploaded captures plus the downloaded trails it authored itself.
///
/// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
/// profile shows offline is the number of trails `/profile/<handle>/trails`
/// can actually render offline — no second, differently-scoped query that
/// could disagree with the list it links to.
///
/// Null — not zero — for anyone else's handle and for a signed-out store:
/// "this device holds none of their trails" is not an answer worth showing,
/// while a genuine zero for the signed-in hiker is.
///
/// The own-handle
/// test and the actor id are both re-derived here from fresh
/// `authProvider`/`currentAccountId` reads rather than passed in, which is
/// The "never from a cached value" — a stale actor id would pair this
/// account with the previous one's trails.

final class ProfileLocalTrailCountFamily extends $Family
    with $FunctionalFamilyOverride<int?, String> {
  ProfileLocalTrailCountFamily._()
    : super(
        retry: null,
        name: r'profileLocalTrailCountProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// How many of [handle]'s trails are on THIS device: the signed-in account's
  /// not-yet-uploaded captures plus the downloaded trails it authored itself.
  ///
  /// Counts exactly the rows `readOwnLocalTrails` returns, so the number the
  /// profile shows offline is the number of trails `/profile/<handle>/trails`
  /// can actually render offline — no second, differently-scoped query that
  /// could disagree with the list it links to.
  ///
  /// Null — not zero — for anyone else's handle and for a signed-out store:
  /// "this device holds none of their trails" is not an answer worth showing,
  /// while a genuine zero for the signed-in hiker is.
  ///
  /// The own-handle
  /// test and the actor id are both re-derived here from fresh
  /// `authProvider`/`currentAccountId` reads rather than passed in, which is
  /// The "never from a cached value" — a stale actor id would pair this
  /// account with the previous one's trails.

  ProfileLocalTrailCountProvider call(String handle) =>
      ProfileLocalTrailCountProvider._(argument: handle, from: this);

  @override
  String toString() => r'profileLocalTrailCountProvider';
}
