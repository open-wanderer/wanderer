// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'comment_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(CommentNotifier)
final commentProvider = CommentNotifierFamily._();

final class CommentNotifierProvider
    extends $AsyncNotifierProvider<CommentNotifier, List<Comment>> {
  CommentNotifierProvider._({
    required CommentNotifierFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'commentProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$commentNotifierHash();

  @override
  String toString() {
    return r'commentProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  CommentNotifier create() => CommentNotifier();

  @override
  bool operator ==(Object other) {
    return other is CommentNotifierProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$commentNotifierHash() => r'f93bccd8ce6b99c10eed028ab9ef516533ce6194';

final class CommentNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          CommentNotifier,
          AsyncValue<List<Comment>>,
          List<Comment>,
          FutureOr<List<Comment>>,
          String
        > {
  CommentNotifierFamily._()
    : super(
        retry: null,
        name: r'commentProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  CommentNotifierProvider call(String trailId) =>
      CommentNotifierProvider._(argument: trailId, from: this);

  @override
  String toString() => r'commentProvider';
}

abstract class _$CommentNotifier extends $AsyncNotifier<List<Comment>> {
  late final _$args = ref.$arg as String;
  String get trailId => _$args;

  FutureOr<List<Comment>> build(String trailId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<List<Comment>>, List<Comment>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<Comment>>, List<Comment>>,
              AsyncValue<List<Comment>>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
