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

  /// Class methods are indented two spaces, so they end at the next
  /// `\n  }\n` rather than the top-level `\n}\n`.
  String sliceMethod(String source, String declaration) {
    final start = source.indexOf(declaration);
    expect(
      start,
      greaterThanOrEqualTo(0),
      reason: '$declaration was renamed or removed; re-point this gate.',
    );
    final end = source.indexOf('\n  }\n', start);
    expect(
      end,
      greaterThan(start),
      reason: 'no closing brace for $declaration',
    );
    return source.substring(start, end);
  }

  int countOf(String haystack, String needle) =>
      needle.allMatches(haystack).length;

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

  group('detail-screen draft actions', () {
    const syncProvider = 'lib/provider/trail/trail_sync_provider.dart';

    test('promoteDraftToPending is owner-scoped and checks isOwnDraft before '
        'promoting', () {
      final body = sliceBody(readStore(), 'bool promoteDraftToPending(');

      expect(
        body.contains('TrailEntity_.owner.equals(accountId)'),
        isTrue,
        reason:
            "Without the owner scope another account's draft sharing a "
            'localId could be promoted into this account upload queue.',
      );
      final guard = body.indexOf('isOwnDraft(');
      final promote = body.indexOf('syncStateAfterLocalEdit(entity.syncState)');
      expect(guard, greaterThanOrEqualTo(0), reason: 'no isOwnDraft guard');
      expect(promote, greaterThanOrEqualTo(0), reason: 'promotion moved');
      expect(
        guard < promote,
        isTrue,
        reason: 'The draft check must run before the promotion write.',
      );
      expect(
        body.contains('entity!.syncState = syncStateAfterLocalEdit(') ||
            body.contains('entity.syncState = syncStateAfterLocalEdit('),
        isTrue,
        reason: 'The promotion must reuse syncStateAfterLocalEdit semantics.',
      );
      expect(body.contains('return false'), isTrue, reason: 'no no-op path');
      expect(body.contains('return true'), isTrue, reason: 'no success path');
      expect(
        body.contains('entity.syncAttempts = 0'),
        isTrue,
        reason: 'A promoted draft must start with a fresh attempt budget.',
      );
      expect(
        body.contains('entity.syncNextAttemptAt = null'),
        isTrue,
        reason: 'A promoted draft must be due immediately, not backed off.',
      );
    });

    test(
      'isOwnDraftRow is owner-scoped, uses isOwnDraft and never toModel',
      () {
        final body = sliceBody(readStore(), 'bool isOwnDraftRow(');

        expect(
          body.contains('TrailEntity_.owner.equals(accountId)'),
          isTrue,
          reason: 'Discard must never match another account row.',
        );
        expect(body.contains('isOwnDraft('), isTrue, reason: 'one predicate');
        expect(
          body.contains('toModel('),
          isFalse,
          reason:
              'A destructive gate must read the raw row; a GPX that stops '
              'parsing must not flip it.',
        );
      },
    );

    test('draft-to-pending has exactly two paths: the form save and '
        'promoteDraftToPending', () {
      var edits = 0;
      final promoteFiles = <String, int>{};
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File) continue;
        final path = entity.path;
        if (!path.endsWith('.dart') ||
            path.endsWith('.g.dart') ||
            path.endsWith('.freezed.dart')) {
          continue;
        }
        final code = readCodeOnly(path);
        final edit = countOf(code, 'syncStateAfterLocalEdit(');
        if (edit > 0) {
          expect(
            path,
            'lib/store/local_trail_store.dart',
            reason:
                'syncStateAfterLocalEdit must only be used by the store, '
                'otherwise a third draft-to-pending path exists.',
          );
          edits += edit;
        }
        final promote = countOf(code, 'promoteDraftToPending(');
        if (promote > 0) promoteFiles[path] = promote;
      }
      expect(
        edits,
        3,
        reason:
            'Declaration + updateLocalTrail + promoteDraftToPending. A '
            'fourth use is a new way to push a draft into the upload queue.',
      );
      expect(
        promoteFiles.keys.toSet(),
        {'lib/store/local_trail_store.dart', syncProvider},
        reason: 'promoteDraftToPending may only be called from TrailSync.',
      );
      expect(
        promoteFiles[syncProvider],
        1,
        reason: 'Exactly one call site: uploadDraft.',
      );
    });

    test('uploadDraft reads the account fresh, promotes, then drains and '
        'invalidates the providers', () {
      final body = sliceMethod(
        readCodeOnly(syncProvider),
        'Future<bool> uploadDraft(',
      );

      final account = body.indexOf('currentAccountId(store)');
      final promote = body.indexOf('promoteDraftToPending(');
      final drain = body.indexOf('drainIfOnline()');
      expect(account, greaterThanOrEqualTo(0), reason: 'no fresh account');
      expect(promote, greaterThan(account), reason: 'promote before account');
      expect(drain, greaterThan(promote), reason: 'drain before promote');
      expect(
        body.contains('ref.invalidate(localTrailProvider(localId))'),
        isTrue,
        reason: 'The detail screen would keep showing the draft.',
      );
      expect(
        body.contains('ref.invalidate(trailLibraryProvider)'),
        isTrue,
        reason: 'The library list would keep showing a stale row.',
      );
      expect(
        body.contains('profileTrailsProvider('),
        isTrue,
        reason: 'The profile trail list would keep showing a stale row.',
      );
    });

    test('discardDraft re-reads the row as an own draft before deleting', () {
      final body = sliceMethod(
        readCodeOnly(syncProvider),
        'Future<bool> discardDraft(',
      );

      final account = body.indexOf('currentAccountId(store)');
      final guard = body.indexOf('isOwnDraftRow(');
      final delete = body.indexOf('deleteUnsynced(localId)');
      expect(account, greaterThanOrEqualTo(0), reason: 'no fresh account');
      expect(guard, greaterThan(account), reason: 'guard before account');
      expect(
        delete,
        greaterThan(guard),
        reason:
            'Deleting before the draft re-check lets Discard destroy a row '
            'that was uploaded or promoted in the meantime.',
      );
      expect(
        body.contains('UnsyncedDeleteResult.deleted'),
        isTrue,
        reason: 'Only a real deletion may report success (and pop the screen).',
      );
    });

    test('_drainOne re-reads the local row after a failed drain', () {
      final body = sliceMethod(
        readCodeOnly(syncProvider),
        'Future<void> _drainOne(',
      );

      final catchAt = body.indexOf('} catch (e, st) {');
      expect(catchAt, greaterThanOrEqualTo(0), reason: 'catch block moved');
      expect(
        body.indexOf('invalidate(localTrailProvider(localId))', catchAt),
        greaterThan(catchAt),
        reason:
            'Without it a mounted detail screen keeps a stale pending state '
            'with an Upload now button after the drain failed.',
      );
      expect(
        body.indexOf('invalidate(localTrailProvider(localId))'),
        lessThan(catchAt),
        reason: 'The success-path invalidation (retired row redirect) is gone.',
      );
    });
  });
}
