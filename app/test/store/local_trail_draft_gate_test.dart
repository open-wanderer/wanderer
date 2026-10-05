import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guards for the Store-touching draft invariants. There is no
/// ObjectBox test harness for plain `flutter test` (see the header of
/// `test/store/local_trail_store_test.dart`), so these slice the
/// comment-stripped function bodies, like
/// `test/store/local_trail_scoping_gate_test.dart`.
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
        .where((line) => !RegExp(r'^\s*//').hasMatch(line))
        .join('\n');
  }

  String sliceBody(String source, String declaration) {
    final start = source.indexOf(declaration);
    expect(
      start,
      greaterThanOrEqualTo(0),
      reason: '$declaration was renamed or removed; re-point this gate.',
    );
    final end = source.indexOf('\n}\n', start);
    expect(
      end,
      greaterThan(start),
      reason: 'no closing brace for $declaration',
    );
    return source.substring(start, end);
  }

  String readStore() => readCodeOnly('lib/store/local_trail_store.dart');

  group('draft rows (store-side invariants)', () {
    test('selectDrainCandidates excludes drafts in its query', () {
      final body = sliceBody(
        readStore(),
        'List<TrailEntity> selectDrainCandidates(',
      );

      expect(
        body.contains(
          'TrailEntity_.dbSyncState.notEquals(TrailSyncState.draft.index)',
        ),
        isTrue,
        reason:
            'Dropping the draft exclusion lets drainIfOnline (app start, '
            'resume, connectivity change) upload an unsaved recording, '
            'possibly as public, before the user taps Save.',
      );
    });

    test('updateLocalTrail promotes via syncStateAfterLocalEdit and keeps '
        'the same row', () {
      final body = sliceBody(
        readStore(),
        'LocalUpdateOutcome updateLocalTrail(',
      );

      expect(
        body.contains('syncStateAfterLocalEdit(existing.syncState)'),
        isTrue,
        reason:
            'Without the promotion a saved draft stays draft forever and '
            'is never uploaded.',
      );
      expect(
        body.contains('entity.obxId = existing.obxId'),
        isTrue,
        reason: 'obxId carry-forward is what keeps it one row, not two.',
      );
      expect(
        body.contains('entity.syncState = existing.syncState'),
        isFalse,
        reason: 'A verbatim syncState carry-forward bypasses the promotion.',
      );
    });

    test('saveNewLocalTrail picks draft or pending from asDraft', () {
      final body = sliceBody(readStore(), 'String saveNewLocalTrail(');

      expect(
        body.contains(
          'asDraft ? TrailSyncState.draft : TrailSyncState.pending',
        ),
        isTrue,
        reason:
            'asDraft must default to a pending row so the form first save '
            'keeps uploading, and produce a draft only when requested.',
      );
    });

    test('resetDrainBackoff returns early for a draft before promoting', () {
      final body = sliceBody(readStore(), 'void resetDrainBackoff(');

      final draftCheck = body.indexOf('TrailSyncState.draft');
      final promote = body.indexOf('entity.syncState = TrailSyncState.pending');
      expect(draftCheck, greaterThanOrEqualTo(0), reason: 'no draft guard');
      expect(promote, greaterThanOrEqualTo(0), reason: 'promotion moved');
      expect(
        draftCheck < promote,
        isTrue,
        reason:
            'The chip retry must never promote a draft into the upload '
            'queue; only the edit-form save does that.',
      );
    });

    test('the create form never creates drafts', () {
      final form = readCodeOnly('lib/routes/trail_create_screen.dart');

      expect(
        form.contains('asDraft: true'),
        isFalse,
        reason: 'The form first save must create a pending row.',
      );
    });
  });
}
