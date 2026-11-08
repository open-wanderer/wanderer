// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_trail_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The not-yet-uploaded trail identified by [localId], or null when no
/// such row belongs to the signed-in account (or its cached GPX no
/// longer parses).
///
/// Synchronous on purpose. The row is already on this device: there is
/// no request to await, no `AsyncError` to be auto-retried, and no
/// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
/// failure mode the own-trails list deliberately avoids. `Trail?` is
/// also a truer type than `AsyncValue<Trail>`: "this local id has no
/// row" is an ordinary outcome, not an error.
///
/// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
/// and its ObjectBox fallback filters on
/// `savedByUserIds.containsElement(userId)`, which a local capture never
/// sets and MUST never set -- writing it would give an unsynced
/// trail the downloaded badge and make the two states indistinguishable.

@ProviderFor(localTrail)
final localTrailProvider = LocalTrailFamily._();

/// The not-yet-uploaded trail identified by [localId], or null when no
/// such row belongs to the signed-in account (or its cached GPX no
/// longer parses).
///
/// Synchronous on purpose. The row is already on this device: there is
/// no request to await, no `AsyncError` to be auto-retried, and no
/// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
/// failure mode the own-trails list deliberately avoids. `Trail?` is
/// also a truer type than `AsyncValue<Trail>`: "this local id has no
/// row" is an ordinary outcome, not an error.
///
/// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
/// and its ObjectBox fallback filters on
/// `savedByUserIds.containsElement(userId)`, which a local capture never
/// sets and MUST never set -- writing it would give an unsynced
/// trail the downloaded badge and make the two states indistinguishable.

final class LocalTrailProvider
    extends $FunctionalProvider<Trail?, Trail?, Trail?>
    with $Provider<Trail?> {
  /// The not-yet-uploaded trail identified by [localId], or null when no
  /// such row belongs to the signed-in account (or its cached GPX no
  /// longer parses).
  ///
  /// Synchronous on purpose. The row is already on this device: there is
  /// no request to await, no `AsyncError` to be auto-retried, and no
  /// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
  /// failure mode the own-trails list deliberately avoids. `Trail?` is
  /// also a truer type than `AsyncValue<Trail>`: "this local id has no
  /// row" is an ordinary outcome, not an error.
  ///
  /// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
  /// and its ObjectBox fallback filters on
  /// `savedByUserIds.containsElement(userId)`, which a local capture never
  /// sets and MUST never set -- writing it would give an unsynced
  /// trail the downloaded badge and make the two states indistinguishable.
  LocalTrailProvider._({
    required LocalTrailFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'localTrailProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$localTrailHash();

  @override
  String toString() {
    return r'localTrailProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<Trail?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Trail? create(Ref ref) {
    final argument = this.argument as String;
    return localTrail(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Trail? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Trail?>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is LocalTrailProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$localTrailHash() => r'24b9ef25f4746748d51e6f0fe83c10d3e4e4b135';

/// The not-yet-uploaded trail identified by [localId], or null when no
/// such row belongs to the signed-in account (or its cached GPX no
/// longer parses).
///
/// Synchronous on purpose. The row is already on this device: there is
/// no request to await, no `AsyncError` to be auto-retried, and no
/// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
/// failure mode the own-trails list deliberately avoids. `Trail?` is
/// also a truer type than `AsyncValue<Trail>`: "this local id has no
/// row" is an ordinary outcome, not an error.
///
/// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
/// and its ObjectBox fallback filters on
/// `savedByUserIds.containsElement(userId)`, which a local capture never
/// sets and MUST never set -- writing it would give an unsynced
/// trail the downloaded badge and make the two states indistinguishable.

final class LocalTrailFamily extends $Family
    with $FunctionalFamilyOverride<Trail?, String> {
  LocalTrailFamily._()
    : super(
        retry: null,
        name: r'localTrailProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// The not-yet-uploaded trail identified by [localId], or null when no
  /// such row belongs to the signed-in account (or its cached GPX no
  /// longer parses).
  ///
  /// Synchronous on purpose. The row is already on this device: there is
  /// no request to await, no `AsyncError` to be auto-retried, and no
  /// `AsyncLoading` for a `.when()` to render as a spinner -- exactly the
  /// failure mode the own-trails list deliberately avoids. `Trail?` is
  /// also a truer type than `AsyncValue<Trail>`: "this local id has no
  /// row" is an ordinary outcome, not an error.
  ///
  /// NOT a variant of `trailProvider`. That one is keyed on the SERVER id
  /// and its ObjectBox fallback filters on
  /// `savedByUserIds.containsElement(userId)`, which a local capture never
  /// sets and MUST never set -- writing it would give an unsynced
  /// trail the downloaded badge and make the two states indistinguishable.

  LocalTrailProvider call(String localId) =>
      LocalTrailProvider._(argument: localId, from: this);

  @override
  String toString() => r'localTrailProvider';
}

/// Whether [localId] identifies a live, not-yet-uploaded capture owned by
/// the signed-in account.
///
/// The single UI-facing handle on `isOwnLiveCapture`: it exists for
/// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
/// download-family guard, and `library_screen.dart`'s Remove tile -- and
/// there must be exactly one predicate serving both, never two.
///
/// [localId] is nullable on purpose so both call sites can watch this
/// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
/// rather than writing a conditional `ref.watch`.

@ProviderFor(ownLiveCapture)
final ownLiveCaptureProvider = OwnLiveCaptureFamily._();

/// Whether [localId] identifies a live, not-yet-uploaded capture owned by
/// the signed-in account.
///
/// The single UI-facing handle on `isOwnLiveCapture`: it exists for
/// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
/// download-family guard, and `library_screen.dart`'s Remove tile -- and
/// there must be exactly one predicate serving both, never two.
///
/// [localId] is nullable on purpose so both call sites can watch this
/// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
/// rather than writing a conditional `ref.watch`.

final class OwnLiveCaptureProvider extends $FunctionalProvider<bool, bool, bool>
    with $Provider<bool> {
  /// Whether [localId] identifies a live, not-yet-uploaded capture owned by
  /// the signed-in account.
  ///
  /// The single UI-facing handle on `isOwnLiveCapture`: it exists for
  /// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
  /// download-family guard, and `library_screen.dart`'s Remove tile -- and
  /// there must be exactly one predicate serving both, never two.
  ///
  /// [localId] is nullable on purpose so both call sites can watch this
  /// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
  /// rather than writing a conditional `ref.watch`.
  OwnLiveCaptureProvider._({
    required OwnLiveCaptureFamily super.from,
    required String? super.argument,
  }) : super(
         retry: null,
         name: r'ownLiveCaptureProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$ownLiveCaptureHash();

  @override
  String toString() {
    return r'ownLiveCaptureProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  bool create(Ref ref) {
    final argument = this.argument as String?;
    return ownLiveCapture(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is OwnLiveCaptureProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$ownLiveCaptureHash() => r'031de3045552e1a668e534cb9572a51322b1ee46';

/// Whether [localId] identifies a live, not-yet-uploaded capture owned by
/// the signed-in account.
///
/// The single UI-facing handle on `isOwnLiveCapture`: it exists for
/// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
/// download-family guard, and `library_screen.dart`'s Remove tile -- and
/// there must be exactly one predicate serving both, never two.
///
/// [localId] is nullable on purpose so both call sites can watch this
/// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
/// rather than writing a conditional `ref.watch`.

final class OwnLiveCaptureFamily extends $Family
    with $FunctionalFamilyOverride<bool, String?> {
  OwnLiveCaptureFamily._()
    : super(
        retry: null,
        name: r'ownLiveCaptureProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Whether [localId] identifies a live, not-yet-uploaded capture owned by
  /// the signed-in account.
  ///
  /// The single UI-facing handle on `isOwnLiveCapture`: it exists for
  /// exactly two call sites -- `trail_dropdown.dart`'s `_allowDelete` and its
  /// download-family guard, and `library_screen.dart`'s Remove tile -- and
  /// there must be exactly one predicate serving both, never two.
  ///
  /// [localId] is nullable on purpose so both call sites can watch this
  /// unconditionally with `ref.watch(ownLiveCaptureProvider(trail.localId))`
  /// rather than writing a conditional `ref.watch`.

  OwnLiveCaptureProvider call(String? localId) =>
      OwnLiveCaptureProvider._(argument: localId, from: this);

  @override
  String toString() => r'ownLiveCaptureProvider';
}
