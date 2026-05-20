// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'summit_log_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SummitLogListNotifier)
final summitLogListProvider = SummitLogListNotifierFamily._();

final class SummitLogListNotifierProvider
    extends $AsyncNotifierProvider<SummitLogListNotifier, List<SummitLog>> {
  SummitLogListNotifierProvider._({
    required SummitLogListNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'summitLogListProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$summitLogListNotifierHash();

  @override
  String toString() {
    return r'summitLogListProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SummitLogListNotifier create() => SummitLogListNotifier();

  @override
  bool operator ==(Object other) {
    return other is SummitLogListNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$summitLogListNotifierHash() =>
    r'761ee458ca435b688265711c2115eccda8c3c24b';

final class SummitLogListNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          SummitLogListNotifier,
          AsyncValue<List<SummitLog>>,
          List<SummitLog>,
          FutureOr<List<SummitLog>>,
          String
        > {
  SummitLogListNotifierFamily._()
    : super(
        retry: null,
        name: r'summitLogListProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SummitLogListNotifierProvider call(String trailId) =>
      SummitLogListNotifierProvider._(argument: trailId, from: this);

  @override
  String toString() => r'summitLogListProvider';
}

abstract class _$SummitLogListNotifier extends $AsyncNotifier<List<SummitLog>> {
  late final _$args = ref.$arg as String;
  String get trailId => _$args;

  FutureOr<List<SummitLog>> build(String trailId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<SummitLog>>, List<SummitLog>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<SummitLog>>, List<SummitLog>>,
              AsyncValue<List<SummitLog>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
