// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'search_history_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The signed-in account's search history, most recently used first.
///
/// Every write re-reads `currentAccountId(store)` rather than remembering
/// the id `build()` saw, so an entry can never land in another account's
/// history.

@ProviderFor(SearchHistoryNotifier)
final searchHistoryProvider = SearchHistoryNotifierProvider._();

/// The signed-in account's search history, most recently used first.
///
/// Every write re-reads `currentAccountId(store)` rather than remembering
/// the id `build()` saw, so an entry can never land in another account's
/// history.
final class SearchHistoryNotifierProvider
    extends $NotifierProvider<SearchHistoryNotifier, List<SearchHistoryItem>> {
  /// The signed-in account's search history, most recently used first.
  ///
  /// Every write re-reads `currentAccountId(store)` rather than remembering
  /// the id `build()` saw, so an entry can never land in another account's
  /// history.
  SearchHistoryNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'searchHistoryProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$searchHistoryNotifierHash();

  @$internal
  @override
  SearchHistoryNotifier create() => SearchHistoryNotifier();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<SearchHistoryItem> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<SearchHistoryItem>>(value),
    );
  }
}

String _$searchHistoryNotifierHash() =>
    r'02f96d5d58005be81db0e71409f44129f2846c83';

/// The signed-in account's search history, most recently used first.
///
/// Every write re-reads `currentAccountId(store)` rather than remembering
/// the id `build()` saw, so an entry can never land in another account's
/// history.

abstract class _$SearchHistoryNotifier
    extends $Notifier<List<SearchHistoryItem>> {
  List<SearchHistoryItem> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<List<SearchHistoryItem>, List<SearchHistoryItem>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<SearchHistoryItem>, List<SearchHistoryItem>>,
              List<SearchHistoryItem>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
