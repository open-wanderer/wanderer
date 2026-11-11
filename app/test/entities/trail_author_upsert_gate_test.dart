import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guards for the actor-upsert invariant, which cannot be
/// exercised behaviourally because there is no ObjectBox test harness for
/// plain `flutter test` (see `test/store/local_trail_store_test.dart`'s file
/// header) -- same rationale as `test/store/local_trail_scoping_gate_test.dart`
/// and `test/store/local_trail_retirement_gate_test.dart`.
///
/// The invariant: `ActorEntity.id` is `@Unique(onConflict: replace)`, so
/// putting a fresh `ActorEntity` (`obxId == 0`) deletes the existing actor row
/// and inserts a new one under a new ObjectBox id -- orphaning the
/// `TrailEntity.author` ToOne of every OTHER trail by that same author, whose
/// library cards then render "Unknown"/"UN". Every writer must therefore route
/// the actor through `actorEntityForUpsert`, which reuses the existing row's
/// ObjectBox id so the put UPDATES instead of re-minting.
void main() {
  final libDir = Directory('lib');

  String readCodeOnly(String path) {
    expect(
      libDir.existsSync(),
      isTrue,
      reason:
          'This test must be run with `flutter test`\'s working directory '
          'set to "app/" (e.g. "cd app && flutter test").',
    );
    final source = File(path).readAsStringSync();
    return source
        .split('\n')
        .where((line) => !RegExp(r'^\s*(//|///)').hasMatch(line))
        .join('\n');
  }

  group('TrailEntity.fromModel routes the author through the upsert', () {
    test('accepts a Store so the author relation can reuse the existing '
        'ActorEntity row id', () {
      final codeOnly = readCodeOnly('lib/entities/trail_entity.dart');

      expect(
        codeOnly.contains(
          'factory TrailEntity.fromModel(Trail trail, {Store? store})',
        ),
        isTrue,
        reason:
            'fromModel lost its Store parameter. Without it the author '
            'relation can only be built from a fresh ActorEntity (obxId 0), '
            'and every put re-mints the actor row -- orphaning the author '
            'ToOne of every other trail by that author.',
      );
    });

    test('calls actorEntityForUpsert when a Store is available, never a bare '
        'ActorEntity.fromModel unconditionally', () {
      final codeOnly = readCodeOnly('lib/entities/trail_entity.dart');

      expect(
        codeOnly.contains('actorEntityForUpsert(store, actor)'),
        isTrue,
        reason:
            'The author relation is no longer built via actorEntityForUpsert. '
            'A bare ActorEntity.fromModel has obxId 0, which makes the '
            'unique-replace conflict strategy delete and re-insert the actor '
            'row on every write.',
      );
    });

    test('builds the category relation the same way -- CategoryEntity carries '
        'the identical unique-replace id', () {
      final codeOnly = readCodeOnly('lib/entities/trail_entity.dart');

      expect(
        codeOnly.contains('categoryEntityForUpsert(store, category)'),
        isTrue,
        reason:
            'The category relation regressed to a bare CategoryEntity'
            '.fromModel. CategoryEntity.id is @Unique(onConflict: replace) '
            'exactly like ActorEntity.id, so the same re-mint orphans '
            'TrailEntity.category on every other trail in that category.',
      );
    });
  });

  group('the category catalog refresh does not re-mint every row', () {
    test('categoryNotifier upserts by record id instead of removeAll + '
        'putMany', () {
      final codeOnly = readCodeOnly('lib/provider/trail/category_provider.dart');

      expect(
        codeOnly.contains('box.removeAll()'),
        isFalse,
        reason:
            'The category cache is being wiped and re-inserted wholesale '
            'again. Every row gets a new ObjectBox id, so TrailEntity'
            '.category on EVERY stored trail is orphaned by what is '
            'otherwise a routine refresh.',
      );
      expect(
        codeOnly.contains('categoryEntityForUpsert(store, category)'),
        isTrue,
        reason:
            'The category refresh no longer routes rows through '
            'categoryEntityForUpsert, so puts re-mint ObjectBox ids instead '
            'of updating the existing rows in place.',
      );
    });
  });

  group('every persisting caller passes its Store to fromModel', () {
    // A caller that omits `store:` silently falls back to the re-minting
    // path, so this is checked per call site rather than by grepping for the
    // helper anywhere in the file.
    const persistingCallSites = <String>[
      'lib/store/local_trail_store.dart',
      'lib/services/trail_download_service.dart',
    ];

    for (final path in persistingCallSites) {
      test('$path never calls TrailEntity.fromModel without store:', () {
        final codeOnly = readCodeOnly(path);
        final calls = RegExp(
          r'TrailEntity\.fromModel\(([^)]*)\)',
        ).allMatches(codeOnly);

        expect(
          calls,
          isNotEmpty,
          reason:
              'No TrailEntity.fromModel call found in $path. Re-point this '
              'gate rather than deleting it.',
        );

        for (final call in calls) {
          expect(
            call.group(1),
            contains('store:'),
            reason:
                'TrailEntity.fromModel(${call.group(1)}) in $path persists '
                'its result but omits `store:`, so the author relation is '
                'built from a fresh ActorEntity and the put re-mints the '
                'actor row -- orphaning every other trail by that author.',
          );
        }
      });
    }
  });
}
