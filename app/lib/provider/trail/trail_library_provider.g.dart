// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'trail_library_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The set of trail ids currently in the signed-in account's downloaded
/// library -- the single named id-set view other providers/utils inject into
/// `applyTrailFilter`'s `downloadedIds` parameter for the `offlineOnly` chip.
///
/// Account scoping is INHERITED from [trailLibraryProvider] and must never
/// be re-implemented here: that provider already returns `const []` for a
/// signed-out account, which is what keeps one account's downloads from
/// leaking into another's filter through this view.
///
/// Empty ids are dropped -- an unsynced local capture has a blanked server
/// id, and is covered instead by `applyTrailFilter`'s separate `isLocal`
/// branch.

@ProviderFor(downloadedTrailIds)
final downloadedTrailIdsProvider = DownloadedTrailIdsProvider._();

/// The set of trail ids currently in the signed-in account's downloaded
/// library -- the single named id-set view other providers/utils inject into
/// `applyTrailFilter`'s `downloadedIds` parameter for the `offlineOnly` chip.
///
/// Account scoping is INHERITED from [trailLibraryProvider] and must never
/// be re-implemented here: that provider already returns `const []` for a
/// signed-out account, which is what keeps one account's downloads from
/// leaking into another's filter through this view.
///
/// Empty ids are dropped -- an unsynced local capture has a blanked server
/// id, and is covered instead by `applyTrailFilter`'s separate `isLocal`
/// branch.

final class DownloadedTrailIdsProvider
    extends $FunctionalProvider<Set<String>, Set<String>, Set<String>>
    with $Provider<Set<String>> {
  /// The set of trail ids currently in the signed-in account's downloaded
  /// library -- the single named id-set view other providers/utils inject into
  /// `applyTrailFilter`'s `downloadedIds` parameter for the `offlineOnly` chip.
  ///
  /// Account scoping is INHERITED from [trailLibraryProvider] and must never
  /// be re-implemented here: that provider already returns `const []` for a
  /// signed-out account, which is what keeps one account's downloads from
  /// leaking into another's filter through this view.
  ///
  /// Empty ids are dropped -- an unsynced local capture has a blanked server
  /// id, and is covered instead by `applyTrailFilter`'s separate `isLocal`
  /// branch.
  DownloadedTrailIdsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'downloadedTrailIdsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$downloadedTrailIdsHash();

  @$internal
  @override
  $ProviderElement<Set<String>> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Set<String> create(Ref ref) {
    return downloadedTrailIds(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Set<String> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Set<String>>(value),
    );
  }
}

String _$downloadedTrailIdsHash() =>
    r'3925626a46baf9a8739e0ce1c0170dd6158d3c2e';

@ProviderFor(downloadedTrailIdsForAuthor)
final downloadedTrailIdsForAuthorProvider =
    DownloadedTrailIdsForAuthorFamily._();

final class DownloadedTrailIdsForAuthorProvider
    extends $FunctionalProvider<Set<String>, Set<String>, Set<String>>
    with $Provider<Set<String>> {
  DownloadedTrailIdsForAuthorProvider._({
    required DownloadedTrailIdsForAuthorFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'downloadedTrailIdsForAuthorProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$downloadedTrailIdsForAuthorHash();

  @override
  String toString() {
    return r'downloadedTrailIdsForAuthorProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<Set<String>> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Set<String> create(Ref ref) {
    final argument = this.argument as String;
    return downloadedTrailIdsForAuthor(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Set<String> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Set<String>>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is DownloadedTrailIdsForAuthorProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$downloadedTrailIdsForAuthorHash() =>
    r'350ded4af0827e6dfea67ddab2801f84ebe8ad4e';

final class DownloadedTrailIdsForAuthorFamily extends $Family
    with $FunctionalFamilyOverride<Set<String>, String> {
  DownloadedTrailIdsForAuthorFamily._()
    : super(
        retry: null,
        name: r'downloadedTrailIdsForAuthorProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  DownloadedTrailIdsForAuthorProvider call(String authorActorId) =>
      DownloadedTrailIdsForAuthorProvider._(
        argument: authorActorId,
        from: this,
      );

  @override
  String toString() => r'downloadedTrailIdsForAuthorProvider';
}

@ProviderFor(TrailLibraryNotifier)
final trailLibraryProvider = TrailLibraryNotifierProvider._();

final class TrailLibraryNotifierProvider
    extends $NotifierProvider<TrailLibraryNotifier, List<Trail>> {
  TrailLibraryNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'trailLibraryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$trailLibraryNotifierHash();

  @$internal
  @override
  TrailLibraryNotifier create() => TrailLibraryNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<Trail> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<Trail>>(value),
    );
  }
}

String _$trailLibraryNotifierHash() =>
    r'07b315f4acd377015a6a4f2a29833e86d6f55ff9';

abstract class _$TrailLibraryNotifier extends $Notifier<List<Trail>> {
  List<Trail> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<Trail>, List<Trail>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<Trail>, List<Trail>>,
              List<Trail>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
