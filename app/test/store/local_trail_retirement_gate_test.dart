import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for `retireUploadedLocalTrail`'s invariants.
///
/// `flutter test` cannot open an ObjectBox `Store` (Phase 31 established
/// there is no ObjectBox test harness for plain `flutter test`, restated in
/// `local_trail_store_test.dart`'s own header), and there is no PocketBase in
/// the test environment. So the transaction this function runs in has NO
/// behavioural surface here at all -- not a hard-to-reach one, an ABSENT one.
/// `local_trail_store_test.dart` covers the pure half of this plan
/// (`shouldDeleteUploadedRow`, `resolveLocalSaveModeForRow`) with real
/// assertions on real inputs; this file pins only the structural invariants a
/// reader cannot see from the call site -- the cascade, the delete-or-demote
/// choice, and (once Task 2 lands) the drain's ordering. The end-to-end
/// behaviour is covered by the device re-test in `36-14-PLAN.md`'s
/// `<verification>` (UAT Test 5).
///
/// The technique mirrors `test/components/trail/trail_dropdown_delete_gate_test.dart`:
/// slice the function body out of the source file and assert on substrings
/// and their relative order, rather than asserting "the source contains
/// retireUploadedLocalTrail" and leaving it at that -- a call-site presence
/// check would pass for a call placed in the `catch` block just as easily as
/// the right place.
void main() {
  final libDir = Directory('lib');

  /// [retireUploadedLocalTrail]'s body, isolated from the rest of the file.
  ///
  /// Comment-stripped first so a `reason:` string quoting one of the
  /// asserted substrings (e.g. a doc comment mentioning
  /// `shouldDeleteUploadedRow(`) can never make an assertion pass on prose
  /// instead of code.
  String retirementBody() {
    expect(
      libDir.existsSync(),
      isTrue,
      reason:
          'This test must be run with `flutter test`\'s working directory '
          'set to `app/` (e.g. "cd app && flutter test").',
    );

    final source = File('lib/store/local_trail_store.dart').readAsStringSync();
    final codeOnly = source
        .split('\n')
        .where((line) => !RegExp(r'^\s*//').hasMatch(line))
        .join('\n');

    final sigStart = codeOnly.indexOf(
      'void retireUploadedLocalTrail(Store store, String localId) {',
    );
    expect(
      sigStart,
      isNot(-1),
      reason:
          'retireUploadedLocalTrail was renamed or its signature changed. '
          'Re-point this gate rather than deleting it -- the invariant '
          'still matters.',
    );

    final bodyEnd = codeOnly.indexOf('\n}', sigStart);
    expect(
      bodyEnd,
      isNot(-1),
      reason: 'Could not find the end of retireUploadedLocalTrail\'s body.',
    );

    return codeOnly.substring(sigStart, bodyEnd);
  }

  group('retireUploadedLocalTrail -- the write transaction', () {
    test('runs inside one write transaction', () {
      final body = retirementBody();

      expect(
        body.contains('runInTransaction(TxMode.write'),
        isTrue,
        reason:
            'Splitting the lookup and the removes across two transactions '
            'reintroduces a window in which a crash leaves a half-retired '
            'row -- a trail present neither as a resumable local capture '
            'nor as a clean server entry.',
      );
    });

    test('is keyed on TrailEntity_.localId, not the server id', () {
      final body = retirementBody();

      expect(
        body.contains('TrailEntity_.localId.equals(localId)'),
        isTrue,
        reason:
            'The local id is the only handle the drain holds for the '
            'whole resume sequence (D-05); keying on anything else breaks '
            'retirement for a row the drain is mid-sequence with.',
      );
    });

    test('gates the delete on shouldDeleteUploadedRow', () {
      final body = retirementBody();

      expect(
        body.contains('shouldDeleteUploadedRow('),
        isTrue,
        reason:
            'An unconditional delete destroys the offline library entry '
            'of any account that downloaded the trail during the upload '
            'window -- shouldDeleteUploadedRow is what prevents that.',
      );
    });

    test('removes the waypoint children via waypointBox.remove(', () {
      final body = retirementBody();

      expect(
        body.contains('waypointBox.remove('),
        isTrue,
        reason:
            'TrailLibraryNotifier.deleteTrail removes the trail row alone '
            'and leaks its waypoints -- "simplifying" this into a call to '
            'that one leaks a set of waypoints per upload.',
      );
    });

    test('the demote branch actually clears entity.owner', () {
      final body = retirementBody();

      expect(
        body.contains('entity.owner = null'),
        isTrue,
        reason:
            'Without giving up ownership, a demoted row still matches '
            'readOwnLocalTrails\' owner clause and the orphan class comes '
            'back through the side door.',
      );
    });
  });

  group(
    'the drain -- ordering and absence facts a call-site grep cannot see',
    () {
      /// The comment-stripped contents of every `.dart` file under `lib/`,
      /// keyed by path.
      Map<String, String> allLibSourcesCodeOnly() {
        expect(
          libDir.existsSync(),
          isTrue,
          reason:
              'This test must be run with `flutter test`\'s working '
              'directory set to `app/` (e.g. "cd app && flutter test").',
        );

        final sources = <String, String>{};
        for (final entity in libDir.listSync(recursive: true)) {
          if (entity is! File || !entity.path.endsWith('.dart')) continue;
          final codeOnly = entity
              .readAsStringSync()
              .split('\n')
              .where((line) => !RegExp(r'^\s*//').hasMatch(line))
              .join('\n');
          sources[entity.path] = codeOnly;
        }
        return sources;
      }

      /// `_drainOne`'s body, comment-stripped, isolated from the rest of the
      /// file.
      String drainOneBody() {
        final source = File(
          'lib/provider/trail/trail_sync_provider.dart',
        ).readAsStringSync();
        final codeOnly = source
            .split('\n')
            .where((line) => !RegExp(r'^\s*//').hasMatch(line))
            .join('\n');

        final sigStart = codeOnly.indexOf('Future<void> _drainOne(');
        expect(
          sigStart,
          isNot(-1),
          reason:
              '_drainOne was renamed or its signature changed. Re-point this '
              'gate rather than deleting it -- the invariant still matters.',
        );

        final bodyEnd = codeOnly.indexOf('\n  }', sigStart);
        expect(
          bodyEnd,
          isNot(-1),
          reason: 'Could not find the end of _drainOne\'s body.',
        );

        return codeOnly.substring(sigStart, bodyEnd);
      }

      test('markTrailSynced is gone from app/lib entirely', () {
        final sources = allLibSourcesCodeOnly();

        for (final entry in sources.entries) {
          expect(
            entry.value.contains('markTrailSynced'),
            isFalse,
            reason:
                'markTrailSynced still appears in ${entry.key}. Its return '
                'means a row survives its own upload as '
                '`owner != null && syncState == synced`, which '
                'readOwnLocalTrails re-emits forever and no delete affordance '
                'in the app can remove (UAT Test 5, blocker).',
          );
        }
      });

      test('retirement happens after the waypoint loop', () {
        final body = drainOneBody();

        final loopIdx = body.indexOf(
          'for (final waypointEntity in entity.waypoints) {',
        );
        final retireIdx = body.indexOf('retireUploadedLocalTrail(');

        expect(loopIdx, isNot(-1));
        expect(retireIdx, isNot(-1));
        expect(
          loopIdx < retireIdx,
          isTrue,
          reason:
              'Retiring before every waypoint is created destroys the '
              'resume state D-05 depends on -- the trail would exist '
              'server-side with missing waypoints and no local row to '
              'retry from.',
        );
      });

      test('retirement happens inside the try, not the catch', () {
        final body = drainOneBody();

        final retireIdx = body.indexOf('retireUploadedLocalTrail(');
        final catchIdx = body.indexOf('} catch (e, st) {');

        expect(retireIdx, isNot(-1));
        expect(catchIdx, isNot(-1));
        expect(
          retireIdx < catchIdx,
          isTrue,
          reason:
              'A retirement reachable from the failure handler deletes the '
              'hiker\'s only copy of a trail whose upload did not complete.',
        );
      });

      test('files are swept after the row is retired', () {
        final body = drainOneBody();

        final retireIdx = body.indexOf('retireUploadedLocalTrail(');
        final sweepIdx = body.indexOf('_deletePhotoDirBestEffort(localId)');

        expect(retireIdx, isNot(-1));
        expect(sweepIdx, isNot(-1));
        expect(
          retireIdx < sweepIdx,
          isTrue,
          reason:
              '`unsyncedLocalIds` cannot see a retired row, so the startup '
              'sweep reclaims a directory orphaned by a crash between the '
              'two, whereas the reverse order leaves a live row pointing at '
              'files that are gone.',
        );
      });
    },
  );

  group('deleteUnsynced -- a row with a server id gets a real server '
      'DELETE too (CR-04)', () {
    /// `TrailSync.deleteUnsynced`'s body, comment-stripped, isolated from
    /// the rest of the file. Same technique as [drainOneBody] above --
    /// `deleteUnsynced` awaits `apiProvider.delete` and mutates a live
    /// `Store`, so like the drain it has no behavioural surface `flutter
    /// test` can reach; the decision it acts on (`trailHasServerId`) is
    /// pure and already covered with real assertions in
    /// `local_trail_store_test.dart`, so this pins only the ordering and
    /// presence facts a call-site grep cannot see.
    String deleteUnsyncedBody() {
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to `app/` (e.g. "cd app && flutter test").',
      );

      final source = File(
        'lib/provider/trail/trail_sync_provider.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      final sigStart = codeOnly.indexOf(
        'Future<UnsyncedDeleteResult> deleteUnsynced(String localId) async {',
      );
      expect(
        sigStart,
        isNot(-1),
        reason:
            'deleteUnsynced was renamed or its signature changed. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final bodyEnd = codeOnly.indexOf('\n  }', sigStart);
      expect(
        bodyEnd,
        isNot(-1),
        reason: 'Could not find the end of deleteUnsynced\'s body.',
      );

      return codeOnly.substring(sigStart, bodyEnd);
    }

    test('decides on readLocalTrailServerId(, never readLocalTrail( -- a '
        'delete decision must not depend on the row\'s cached GPX still '
        'parsing (CR-02)', () {
      final body = deleteUnsyncedBody();

      expect(
        body.contains('readLocalTrailServerId('),
        isTrue,
        reason:
            'A `failed`/`pending`/`uploading` row can already carry a '
            'real server id -- `writeServerTrailId` stamps it the '
            'instant `PUT /trail/form` is accepted, well before the row '
            'is retired. The decision must read the raw entity id '
            'column, not go through `toModel()`, which returns null on '
            'ANY parse failure (including unparseable cached GPX) and '
            'would then skip the server DELETE, stranding a '
            'possibly-public trail on the server with no device left '
            'pointing at it.',
      );
      expect(
        body.contains('readLocalTrail('),
        isFalse,
        reason:
            'readLocalTrail() -> toModel() can silently return null on '
            'an unparseable cached GPX, which previously made this '
            'method treat a row with a real server id as if it had '
            'none (CR-02). The decision must never route through it.',
      );
    });

    test('validates the server id through recordIdDirSegment( before it '
        'reaches the Dio path (WR-17)', () {
      final body = deleteUnsyncedBody();

      expect(
        body.contains('recordIdDirSegment('),
        isTrue,
        reason:
            'The server id originates in a response body from a server '
            'that may be federated or compromised. Interpolating it '
            'straight into a Dio path without validation lets a '
            'malicious id (e.g. containing `../`) steer the request at '
            'a different endpoint under the same authenticated session.',
      );
    });

    test('the server DELETE runs BEFORE the local row is removed', () {
      final body = deleteUnsyncedBody();

      final serverDeleteIdx = body.indexOf('apiProvider).delete(');
      final localDeleteIdx = body.indexOf('deleteLocalTrailRow(');

      expect(serverDeleteIdx, isNot(-1));
      expect(localDeleteIdx, isNot(-1));
      expect(
        serverDeleteIdx < localDeleteIdx,
        isTrue,
        reason:
            'The local row is the device\'s only handle on the trail. If '
            'the server DELETE ran after (or the local delete were not '
            'gated on it succeeding), a network failure would strand the '
            'local row deleted and the server copy alive -- exactly the '
            'shape CR-04 exists to prevent.',
      );
    });

    test('deleteLocalTrailRow( is called with accountId: -- every write on '
        'the delete path is owner-scoped (WR-10)', () {
      final body = deleteUnsyncedBody();

      final localDeleteIdx = body.indexOf('deleteLocalTrailRow(');
      expect(localDeleteIdx, isNot(-1));

      final callEnd = body.indexOf(';', localDeleteIdx);
      expect(callEnd, isNot(-1));
      final call = body.substring(localDeleteIdx, callEnd);
      expect(
        call.contains('accountId:'),
        isTrue,
        reason:
            'Without an owner clause, account B could delete account '
            'A\'s device-only row through a stale localId (WR-10).',
      );
    });

    test('a failed server DELETE is classified via resolveServerDeleteOutcome( '
        'between the DELETE call and the local delete -- the failure is '
        'classified, not swallowed unconditionally (WR-15)', () {
      final body = deleteUnsyncedBody();

      final serverDeleteIdx = body.indexOf('apiProvider).delete(');
      final localDeleteIdx = body.indexOf('deleteLocalTrailRow(');
      expect(serverDeleteIdx, isNot(-1));
      expect(localDeleteIdx, isNot(-1));

      final slice = body.substring(serverDeleteIdx, localDeleteIdx);
      expect(
        slice.contains('resolveServerDeleteOutcome('),
        isTrue,
        reason:
            'A `catch` that swallows every DELETE failure and proceeds '
            'unconditionally to the local delete treats a 401/403/500 '
            'exactly like a 404, silently destroying the local row '
            'while the server copy survives. The failure must be '
            'classified via resolveServerDeleteOutcome so a 404 (the '
            'server copy is already gone) is the only case that '
            'proceeds automatically.',
      );
    });
  });
}
