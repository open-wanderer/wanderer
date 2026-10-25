import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for the write-draft-before-clear ordering in
/// `_saveRecordedTrack`.
///
/// `navigation_screen.dart` needs a live ObjectBox store, tracelet, a router
/// and a map controller, so it cannot be mounted in `flutter test`. The
/// ordering itself is behaviour-tested in
/// `test/store/local_trail_store_test.dart` via `persistDraftThenClearSession`
/// with fakes; this gate only pins that the screen actually goes through it.
void main() {
  String codeOnly(String path) {
    expect(
      Directory('lib').existsSync(),
      isTrue,
      reason:
          'This test must be run with `flutter test`\'s working directory '
          'set to "app/" (e.g. "cd app && flutter test").',
    );
    return File(path)
        .readAsStringSync()
        .split('\n')
        .where((line) => !RegExp(r'^\s*//').hasMatch(line))
        .join('\n');
  }

  String saveBody() {
    final source = codeOnly('lib/routes/navigation_screen.dart');
    final start = source.indexOf('Future<void> _saveRecordedTrack(');
    expect(
      start,
      greaterThanOrEqualTo(0),
      reason: '_saveRecordedTrack was renamed; re-point this gate.',
    );
    final end = source.indexOf('\n  }\n', start);
    expect(end, greaterThan(start), reason: 'no closing brace found');
    return source.substring(start, end);
  }

  group('_saveRecordedTrack durability ordering', () {
    test('goes through persistDraftThenClearSession, writeDraft first', () {
      final body = saveBody();

      expect(body.contains('persistDraftThenClearSession('), isTrue);
      expect(
        body.indexOf('writeDraft:') < body.indexOf('clearSession:'),
        isTrue,
        reason: 'The draft must be written before the session is released.',
      );
    });

    test('active_nav.clear runs once, only inside clearSession, after the '
        'persist timer is cancelled', () {
      final body = saveBody();
      final clearSessionIdx = body.indexOf('clearSession:');

      expect(
        'active_nav.clear(_store)'.allMatches(body).length,
        1,
        reason:
            'A second clear outside the closure re-opens the data-loss '
            'window the draft write is meant to close.',
      );
      expect(
        body.indexOf('active_nav.clear(_store)') > clearSessionIdx,
        isTrue,
      );
      expect(
        body.indexOf('_persistTimer?.cancel()') > clearSessionIdx,
        isTrue,
        reason:
            'Cancelling the persist timer before the draft write succeeds '
            'would stop persisting the session after a failed write.',
      );
    });

    test('the draft is written fresh-owner, as a draft, before the clear', () {
      final body = saveBody();
      final write = body.indexOf('writeDraft:');
      final clear = body.indexOf('clearSession:');
      final between = body.substring(write, clear);

      expect(
        between.contains('currentAccountId(_store)'),
        isTrue,
        reason: 'The owner id must be read fresh at the point of the write.',
      );
      expect(between.contains('asDraft: true'), isTrue);
    });

    test('the form opens on the draft row, not on the in-memory trail', () {
      final body = saveBody();

      expect(body.contains('syncState: TrailSyncState.draft'), isTrue);
      expect(body.contains('localId: localId'), isTrue);
      expect(body.contains('pendingImportedTrail = draftTrail'), isTrue);
      expect(
        body.contains('extra: draftTrail'),
        isTrue,
        reason: 'Without the localId the form would createLocal a duplicate.',
      );
      expect(body.contains('extra: trail)'), isFalse);
      expect(
        body.indexOf('persistDraftThenClearSession(') <
            body.indexOf("pushReplacement('/trail/create/edit'"),
        isTrue,
      );
    });

    test('the create form still seeds its localId from the passed trail', () {
      final form = codeOnly('lib/routes/trail_create_screen.dart');

      expect(
        form.contains('_localId = widget.trail.localId;'),
        isTrue,
        reason:
            'This link makes the form save take updateLocal on the draft '
            'row instead of creating a second row.',
      );
    });
  });
}
