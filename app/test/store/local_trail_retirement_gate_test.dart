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
///
/// 36-20 (WR-05/T-36-20-01/T-36-20-03): the assertions in this file split
/// into two honest categories, and every group below is labelled with which
/// one it is:
///
/// - EFFECT assertions -- a value is assigned to a variable and later
///   CONSUMED (not merely mentioned), or a call is ordered relative to a
///   real state transition another call performs. These fail when the
///   effect they pin is removed, not merely renamed or relocated to a
///   comment. The retirement-return-value capture-and-consume group and the
///   drain's memo/invalidate ordering group below are effect assertions.
/// - PRESENCE/ABSENCE facts -- a substring is (or is not) in the sliced
///   body, with no claim about what consumes it. `runs inside one write
///   transaction`, `is keyed on TrailEntity_.localId`, `removes the waypoint
///   children`, `markTrailSynced is gone from app/lib entirely` and the
///   `deleteUnsynced`/`writeServerWaypointId` groups below are presence/
///   absence facts -- still real regression pins (each one was written
///   against a genuine incident), but they do not prove a value flows
///   anywhere.
///
/// Neither category is a substitute for the device UAT named in each of this
/// phase's plans' `<verification>` sections -- `flutter test` cannot open an
/// ObjectBox `Store` or reach PocketBase, so no assertion here proves the
/// retirement transaction actually runs correctly against live data. What
/// this file proves is narrower and still real: that the SOURCE, as written,
/// cannot silently regress to a shape a past incident already produced.
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

    // 38.1 WR-03: the return type became a record so the caller can tell the
    // demote branch from the remove branch and reclaim library/<serverId>/.
    // Matched on the record type + name so a further signature change still
    // trips this gate rather than silently skipping the whole group.
    final sigStart = codeOnly.indexOf(
      '({String? serverId, bool rowRemoved}) retireUploadedLocalTrail(',
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

    // EFFECT assertion (36-20): the captured server id must actually be
    // RETURNED from both exits, not merely computed and discarded -- CR-01's
    // whole fix depends on the caller receiving this value.
    test(
      'captures the server id once, before either exit mutates the row, '
      'and returns it from BOTH exits (CR-01)',
      () {
        final body = retirementBody();

        final captureIdx = body.indexOf('final serverId =');
        expect(
          captureIdx,
          isNot(-1),
          reason:
              'Could not find the `final serverId =` capture. Re-point '
              'this gate rather than deleting it -- the invariant still '
              'matters.',
        );

        // 38.1 WR-03: both exits now return a record. The CR-01 invariant is
        // unchanged -- each must still carry the captured serverId -- so the
        // gate matches `serverId: serverId` instead of `return serverId;`.
        final returnCount = RegExp(
          r'serverId:\s*serverId',
        ).allMatches(body).length;
        expect(
          returnCount,
          greaterThanOrEqualTo(2),
          reason:
              'retireUploadedLocalTrail has two exits (demote, delete) and '
              'both must return the captured serverId -- discarding it on '
              'either exit re-opens CR-01: the screen\'s trail.id stays '
              '\'\' forever and every post-upload save fails for a trail '
              'retired down that exit.',
        );

        // 38.1 WR-03: and they must disagree on rowRemoved, or the caller
        // cannot tell which branch ran and the library/<serverId>/ reclaim
        // either never fires or fires on the demote branch and deletes a
        // still-held library entry's photos.
        expect(
          RegExp(r'rowRemoved:\s*false').allMatches(body).length,
          greaterThanOrEqualTo(1),
          reason: 'No exit reports rowRemoved: false -- the demote branch '
              'would trigger the caller\'s library-directory reclaim.',
        );
        expect(
          RegExp(r'rowRemoved:\s*true').allMatches(body).length,
          greaterThanOrEqualTo(1),
          reason: 'No exit reports rowRemoved: true -- WR-03\'s orphaned '
              'library/<serverId>/ is never reclaimed.',
        );

        final ownerNullIdx = body.indexOf('entity.owner = null');
        final removeIdx = body.indexOf('trailBox.remove(');
        expect(ownerNullIdx, isNot(-1));
        expect(removeIdx, isNot(-1));
        expect(
          captureIdx < ownerNullIdx && captureIdx < removeIdx,
          isTrue,
          reason:
              'serverId must be captured BEFORE either exit mutates or '
              'removes the row -- capturing after `entity.owner = null` or '
              '`trailBox.remove(` reads a value the row no longer reliably '
              'carries.',
        );
      },
    );
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
        final sweepIdx = body.indexOf(
          '_deletePhotoDirBestEffort(accountId, localId)',
        );

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

      test('hasKeylessPendingWaypoint( is checked before the in-flight set '
          'is joined -- an invariant break must not record a failed '
          'attempt (WR-04)', () {
        final body = drainOneBody();

        final guardIdx = body.indexOf('hasKeylessPendingWaypoint(');
        final joinIdx = body.indexOf('state = {...state, localId};');

        expect(guardIdx, isNot(-1));
        expect(joinIdx, isNot(-1));
        expect(
          guardIdx < joinIdx,
          isTrue,
          reason:
              'A waypoint that is still a local sentinel with no localKey '
              'to record the server\'s returned id against is a corrupt '
              'row, not a network condition -- no retry will ever fix it. '
              'Checking this only after the row joins the in-flight set '
              'and enters the try lets four fast passes (a lifecycle or '
              'connectivity flurry produces these within seconds) park an '
              'otherwise-healthy trail as `failed`, after which isDrainDue '
              'never picks it up again.',
        );
      });

      test('the body contains no StateError( -- the keyless-waypoint '
          'throw is gone, so it can no longer reach recordDrainFailure '
          '(WR-04)', () {
        final body = drainOneBody();

        expect(
          body.contains('StateError('),
          isFalse,
          reason:
              'A StateError thrown from inside the try lands in the '
              'generic failure handler and consumes one of the four '
              'kMaxSyncAttempts for an invariant break no retry could '
              'ever fix -- the hasKeylessPendingWaypoint guard above '
              'must be what prevents reaching this point instead.',
        );
      });

      // EFFECT assertion (36-20, CR-01/WR-01): the retirement return value
      // must be ASSIGNED (not discarded) and CONSUMED by the memo before the
      // detail/create screen's only other handle on the trail (the memo
      // itself) can exist.
      test(
        'assigns retireUploadedLocalTrail\'s return value -- it is never '
        'discarded (CR-01)',
        () {
          final body = drainOneBody();

          expect(
            body.contains('= retireUploadedLocalTrail('),
            isTrue,
            reason:
                'Calling retireUploadedLocalTrail( for its side effect '
                'alone and discarding the return value re-opens CR-01: the '
                'screen\'s trail.id snapshot stays the blank local-'
                'sentinel value forever (there are no ObjectBox '
                'Query.watch() streams anywhere in this app), and every '
                'post-upload save from that screen hits the dead POST.',
          );
        },
      );

      test(
        '_rememberRetiredServerId( runs after the retirement return value '
        'is captured, not before (CR-01)',
        () {
          final body = drainOneBody();

          final captureIdx = body.indexOf('= retireUploadedLocalTrail(');
          final rememberIdx = body.indexOf('_rememberRetiredServerId(');
          expect(captureIdx, isNot(-1));
          expect(
            rememberIdx,
            isNot(-1),
            reason:
                '_rememberRetiredServerId is gone from _drainOne. Without '
                'it, retireUploadedLocalTrail\'s return value is captured '
                'but never memoized, so serverIdForRetired can never '
                'resolve it for a still-mounted screen.',
          );
          expect(
            captureIdx < rememberIdx,
            isTrue,
            reason:
                '_rememberRetiredServerId must run AFTER the return value '
                'is captured into a local variable -- calling it before '
                'the capture line exists would memoize a stale or '
                'undefined value.',
          );
        },
      );

      test(
        'invalidates localTrailProvider(localId) AFTER retirement runs, so '
        'a mounted detail screen re-reads instead of rendering a dead row '
        '(WR-01)',
        () {
          final body = drainOneBody();

          final retireIdx = body.indexOf('retireUploadedLocalTrail(');
          final invalidateIdx = body.indexOf(
            'invalidate(localTrailProvider(localId))',
          );
          expect(retireIdx, isNot(-1));
          expect(
            invalidateIdx,
            isNot(-1),
            reason:
                'localTrailProvider(localId) is no longer invalidated in '
                '_drainOne. A hiker sitting on /trail/local/<localId> '
                'while the upload completes keeps rendering a row that no '
                'longer exists, with a live Edit button that walks '
                'straight into CR-01\'s dead end.',
          );
          expect(
            retireIdx < invalidateIdx,
            isTrue,
            reason:
                'The invalidation must run AFTER retireUploadedLocalTrail, '
                'or the still-mounted provider re-reads a row that has not '
                'been retired yet and observes no change.',
          );
        },
      );
    },
  );

  // Both an EFFECT assertion (the account comparison actually gates the
  // return, not merely appears somewhere in the body) and a threat-model
  // control (T-36-16-01/T-36-20-03): `trailSyncProvider` is deliberately
  // excluded from `accountScopedProviders`, so this memo survives an account
  // switch, and without the check account B could resolve account A's
  // retired server id and post an edit to A's trail.
  group('serverIdForRetired -- the memoized value never crosses an account '
      'boundary (T-36-16-01, T-36-20-03)', () {
    /// [TrailSync.serverIdForRetired]'s body, comment-stripped, isolated
    /// from the rest of the file. Built the same way as [drainOneBody]
    /// above.
    String serverIdForRetiredBody() {
      final source = File(
        'lib/provider/trail/trail_sync_provider.dart',
      ).readAsStringSync();
      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      final sigStart = codeOnly.indexOf(
        'String? serverIdForRetired(String localId) {',
      );
      expect(
        sigStart,
        isNot(-1),
        reason:
            'serverIdForRetired was renamed or its signature changed. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final bodyEnd = codeOnly.indexOf('\n  }', sigStart);
      expect(
        bodyEnd,
        isNot(-1),
        reason: 'Could not find the end of serverIdForRetired\'s body.',
      );

      return codeOnly.substring(sigStart, bodyEnd);
    }

    test('reads currentAccountId( fresh, before consulting the memo', () {
      final body = serverIdForRetiredBody();

      expect(
        body.contains('currentAccountId('),
        isTrue,
        reason:
            'serverIdForRetired must re-read currentAccountId fresh at the '
            'point of use (D-13), not from a cached field -- a stale '
            'cached id here is exactly the cross-account leak the account '
            'check exists to prevent.',
      );
    });

    test(
      'never returns the memoized server id except from inside an account '
      'comparison',
      () {
        final body = serverIdForRetiredBody();

        final entryIdx = body.indexOf('_retiredServerIds[localId]');
        expect(
          entryIdx,
          isNot(-1),
          reason:
              'Could not find the memo lookup. Re-point this gate rather '
              'than deleting it -- the invariant still matters.',
        );

        final returnIdx = body.lastIndexOf('return entry.serverId;');
        expect(
          returnIdx,
          isNot(-1),
          reason:
              'Could not find the final `return entry.serverId;`. '
              'Re-point this gate rather than deleting it -- the '
              'invariant still matters.',
        );
        expect(
          entryIdx < returnIdx,
          isTrue,
          reason:
              'The memo lookup must precede the value it returns.',
        );

        // Everything between the memo lookup and the eventual return is the
        // region that MUST contain the account comparison -- removing it
        // (T-36-20-03's falsifying rewrite: delete the
        // `if (entry.accountId != accountId) { ...; return null; }` block)
        // collapses this slice to nothing but the null-check, and the
        // assertions below fail.
        final guardSlice = body.substring(entryIdx, returnIdx);
        expect(
          guardSlice.contains('accountId'),
          isTrue,
          reason:
              'trailSyncProvider is deliberately excluded from '
              'accountScopedProviders (account_scope_invalidation.dart), '
              'so this memo survives an account switch. Without a '
              'per-entry account comparison between the memo lookup and '
              'the return, account B could resolve account A\'s retired '
              'server id through a stale localId and post an edit to '
              'A\'s trail.',
        );
        expect(
          guardSlice.contains('!='),
          isTrue,
          reason:
              'The account check must be a COMPARISON (entry.accountId != '
              'accountId), not merely a mention of the word "accountId" '
              'somewhere in the slice (e.g. in an unrelated debugPrint).',
        );
      },
    );
  });

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

    test(
      'deleteLocalTrailRow(\'s result gates the photo delete -- a no-op row '
      'delete must not be followed by an unscoped recursive photo delete '
      '(D-07, CR-01)',
      () {
        final body = deleteUnsyncedBody();

        final assignMatch = RegExp(
          r'(\w+)\s*=\s*deleteLocalTrailRow\(',
        ).firstMatch(body);
        expect(
          assignMatch,
          isNotNull,
          reason:
              'deleteLocalTrailRow now returns LocalRowDeleteOutcome (plan '
              '38.1-02, 38.1 CR-02); deleteUnsynced must capture that '
              'result rather than discarding it, or a no-op row delete '
              'cannot be distinguished from a real one.',
        );
        final resultVar = assignMatch!.group(1)!;

        final guardIdx = body.indexOf(
          'if ($resultVar == LocalRowDeleteOutcome.noMatch)',
        );
        expect(
          guardIdx,
          isNot(-1),
          reason:
              'The captured outcome must be guarded with '
              '`if ($resultVar == LocalRowDeleteOutcome.noMatch)` before '
              'anything destructive runs.',
        );

        final photoDeleteIdx = body.indexOf('_deletePhotoDirBestEffort(');
        expect(photoDeleteIdx, isNot(-1));

        expect(
          guardIdx < photoDeleteIdx,
          isTrue,
          reason:
              'CR-01: a no-op row delete (deleteLocalTrailRow matched no '
              'row owned by this account) followed by an unscoped '
              'recursive photo delete is exactly how account B destroyed '
              'account A\'s photos -- deleteUnsyncedPhotoDir had no '
              'account component and resolved the same directory for '
              'both accounts. The guard must sit before the photo '
              'delete, not after.',
        );
      },
    );

    test(
      'a no-op row delete returns UnsyncedDeleteResult.failed, never '
      '.deleted -- reporting success for a delete that never happened '
      'produces a success toast, a popped route and a map-provider '
      'announcement for nothing (D-07)',
      () {
        final body = deleteUnsyncedBody();

        final assignMatch = RegExp(
          r'(\w+)\s*=\s*deleteLocalTrailRow\(',
        ).firstMatch(body);
        expect(assignMatch, isNotNull);
        final resultVar = assignMatch!.group(1)!;

        final guardIdx = body.indexOf(
          'if ($resultVar == LocalRowDeleteOutcome.noMatch)',
        );
        expect(guardIdx, isNot(-1));

        final guardEnd = body.indexOf('}', guardIdx);
        expect(
          guardEnd,
          isNot(-1),
          reason: 'Could not find the end of the no-op-row-delete guard.',
        );
        final guardBlock = body.substring(guardIdx, guardEnd);

        expect(
          guardBlock.contains('UnsyncedDeleteResult.failed'),
          isTrue,
          reason:
              'Reporting UnsyncedDeleteResult.deleted for an operation '
              'that matched no row produces a success toast, a popped '
              'route and a map-provider announcement for a delete that '
              'never happened.',
        );

        // WR-08 (38.1): `deleted` is now reachable from this branch, but ONLY
        // when gated on `serverCopyDeleted`. An ungated `deleted` here is the
        // exact D-07 regression this gate exists to catch.
        if (guardBlock.contains('UnsyncedDeleteResult.deleted')) {
          expect(
            guardBlock.contains('serverCopyDeleted'),
            isTrue,
            reason:
                'The no-match branch may only report .deleted when '
                'serverCopyDeleted proves we owned the row and the server '
                'copy is gone. An unconditional .deleted here reports '
                'success for a delete that touched nothing (D-07).',
          );
        }
      },
    );

    test(
      'serverCopyDeleted can only be set inside the owner-scoped serverId '
      'branch -- otherwise WR-08 would hand the CR-01 overlap a success '
      'result for another account\'s row (D-07/CR-01)',
      () {
        final body = deleteUnsyncedBody();

        final serverIdGuardIdx = body.indexOf('if (serverId != null)');
        expect(
          serverIdGuardIdx,
          isNot(-1),
          reason: 'Could not find the serverId != null guard.',
        );

        // Every assignment that sets the flag true must appear after the
        // owner-scoped `readLocalTrailServerId` result has been proven
        // non-null. `readLocalTrailServerId` is owner-scoped, so in the CR-01
        // overlap it returns null, this block never runs, and the flag stays
        // false -- keeping the no-match branch on `failed`.
        final assignments = RegExp(
          r'serverCopyDeleted\s*=\s*true',
        ).allMatches(body).toList();
        expect(
          assignments,
          isNotEmpty,
          reason: 'serverCopyDeleted is never set -- WR-08 has regressed.',
        );
        for (final m in assignments) {
          expect(
            m.start,
            greaterThan(serverIdGuardIdx),
            reason:
                'serverCopyDeleted is set outside the `serverId != null` '
                'branch. That lets a non-owning account reach a .deleted '
                'result for a row it never owned (CR-01).',
          );
        }
      },
    );
  });

  group(
    'writeServerWaypointId -- localPhotos survive until the trail is '
    'retired (WR-09)',
    () {
      /// [writeServerWaypointId]'s body, comment-stripped, isolated from
      /// the rest of the file.
      String writeServerWaypointIdBody() {
        expect(
          libDir.existsSync(),
          isTrue,
          reason:
              'This test must be run with `flutter test`\'s working '
              'directory set to `app/` (e.g. "cd app && flutter test").',
        );

        final source = File(
          'lib/store/local_trail_store.dart',
        ).readAsStringSync();
        final codeOnly = source
            .split('\n')
            .where((line) => !RegExp(r'^\s*//').hasMatch(line))
            .join('\n');

        final sigStart = codeOnly.indexOf('void writeServerWaypointId(');
        expect(
          sigStart,
          isNot(-1),
          reason:
              'writeServerWaypointId was renamed or its signature '
              'changed. Re-point this gate rather than deleting it -- the '
              'invariant still matters.',
        );

        final bodyEnd = codeOnly.indexOf('\n}', sigStart);
        expect(
          bodyEnd,
          isNot(-1),
          reason: 'Could not find the end of writeServerWaypointId\'s body.',
        );

        return codeOnly.substring(sigStart, bodyEnd);
      }

      test('never clears localPhotos', () {
        final body = writeServerWaypointIdBody();

        expect(
          body.contains('localPhotos = []'),
          isFalse,
          reason:
              'Clearing localPhotos the instant one waypoint\'s create '
              'succeeds orphans its JPEGs on disk the moment a LATER '
              'waypoint in the same drain loop fails and the trail parks '
              'as `failed` -- the model can no longer reach files that '
              'still sit under unsynced/<localId>/waypoints/<key>/ until '
              'the trail is retired or deleted.',
        );
      });
    },
  );
}
