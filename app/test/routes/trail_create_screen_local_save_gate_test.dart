import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for three structural invariants introduced by
/// local-first recording (36-06, 36-10):
///
/// 1. Every not-yet-uploaded waypoint must carry an empty server id (plus a
///    local key) rather than a timestamp-derived synthetic id.
/// 2. `_onSave` must route on `resolveLocalSaveMode`/`LocalSaveMode`, and its
///    two local-first branches must never touch `trailSaveProvider`.
/// 3. Both save tails (`_finishLocalSave`, `_saveViaNetwork`) must call
///    `_invalidateOwnTrailsList()`, and that method must invalidate the same
///    pair `trail_sync_provider.dart` invalidates after a successful drain
///    (36-10, gap 3).
///
/// `trail_create_screen.dart` needs auth, a router, a live ObjectBox store,
/// an image picker and a map controller to drive behaviourally, so these
/// invariants are guarded at source level instead -- the same rationale and
/// form as `test/components/trail/trail_dropdown_delete_gate_test.dart`'s
/// delete-branch gate and the PORT-03 gate in
/// `test/util/trail_import_util_test.dart`.
///
/// Scope note on invariant 3 specifically: this gate asserts call-site
/// PLACEMENT only, not that the invalidation propagates to a rendered list --
/// `flutter test` cannot mount this screen (no ObjectBox store, no router, no
/// image picker, no map controller), so the propagation itself is verified on
/// a physical device (see 36-10-PLAN.md's `<verification>`). This differs
/// from the failure mode `trail_dropdown_delete_gate_test.dart` guards
/// against: that gate described the shape of UI nothing could reach, whereas
/// this one pins placement on a code path whose reachability is already
/// proven by UAT Test 1 and Test 2 both passing.
///
/// WR-05 (36-REVIEW.md, re-derived 2026-08-03): the three gates the prior
/// fix pass added asserted only token PRESENCE and ORDERING -- e.g. the
/// original CR-01 gate would still pass verbatim against
/// `if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }`, an empty guard
/// body that lets the blank-id POST straight through. Every gate rewritten
/// or added by 36-20 below asserts an EFFECT of the guard (a `return;`
/// actually present, a specific message actually chosen, a value actually
/// threaded into the call it protects) rather than only a substring's
/// position. Each such gate's `reason:` names the exact falsifying rewrite
/// it defends against, and 36-20-SUMMARY.md records the rewrite being
/// applied to a scratch copy and observed to fail before being reverted.
void main() {
  final libDir = Directory('lib');
  final source = File('lib/routes/trail_create_screen.dart').readAsStringSync();

  /// The index just past the closing brace of the method whose body starts
  /// at [start].
  ///
  /// Anchoring on the next MEMBER (e.g. `bool get _hasUnsavedChanges` after
  /// `_onSave`) was wrong: several other members can sit between a method and
  /// the next one, so the slice would silently include unrelated method
  /// bodies. The gate would then fail for a reference that was never in the
  /// method under test at all. A closing brace at method indentation is the
  /// actual end of the method.
  int methodEnd(int start) {
    final end = source.indexOf('\n  }\n', start);
    expect(end, isNot(-1), reason: 'Could not find the method\'s closing brace.');
    return end + '\n  }\n'.length;
  }

  test(
    'trail_create_screen.dart contains no microsecondsSinceEpoch outside comments',
    () {
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to "app/" (e.g. "cd app && flutter test").',
      );

      final codeOnly = source
          .split('\n')
          .where((line) => !RegExp(r'^\s*//').hasMatch(line))
          .join('\n');

      expect(
        codeOnly.contains('microsecondsSinceEpoch'),
        isFalse,
        reason:
            'A waypoint minted with a synthetic id looks already-uploaded '
            'to the drain\'s id.isEmpty resume check and silently never '
            'reaches the server (RESEARCH.md Pitfall 1).',
      );
    },
  );

  test('both Waypoint( constructions pass id: \'\'', () {
    // Word-boundary regex so this only matches the actual constructor call,
    // not the many method names in this file that end in "...Waypoint(" (
    // _onCreateWaypoint(, _onEditWaypoint(, _appendWaypoint(, etc).
    final matches = RegExp(r'\bWaypoint\(').allMatches(source).toList();

    expect(
      matches,
      hasLength(2),
      reason:
          'Expected exactly two Waypoint( constructor calls -- the manual '
          'stub in _onCreateWaypoint and the photo-EXIF construction in '
          '_onCreateWaypointsFromPhotos. If a new Waypoint( construction was '
          'added, extend this gate rather than deleting it; a waypoint '
          'minted with a synthetic id looks already-uploaded to the drain\'s '
          'id.isEmpty resume check and silently never reaches the server '
          '(RESEARCH.md Pitfall 1).',
    );

    for (final match in matches) {
      final tailEnd = (match.end + 300).clamp(0, source.length);
      final tail = source.substring(match.end, tailEnd);
      expect(
        tail.contains("id: ''"),
        isTrue,
        reason:
            'A waypoint constructed without id: \'\' mints a synthetic id '
            'that looks already-uploaded to the drain\'s id.isEmpty resume '
            'check and silently never reaches the server (RESEARCH.md '
            'Pitfall 1).',
      );
    }
  });

  test(
    '_onSave routes on resolveLocalSaveMode and references all three LocalSaveMode values',
    () {
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(
        saveStart,
        isNot(-1),
        reason:
            '_onSave was renamed or its signature changed. Re-point this '
            'gate rather than deleting it -- the invariant still matters.',
      );

      final body = source.substring(saveStart, methodEnd(saveStart));

      expect(
        body.contains('resolveLocalSaveMode'),
        isTrue,
        reason:
            'Routing on trail.id.isEmpty alone creates a second server '
            'trail when a hiker re-saves an unsynced draft (RESEARCH.md '
            'Pitfall 2).',
      );

      for (final mode in [
        'LocalSaveMode.networkUpdate',
        'LocalSaveMode.createLocal',
        'LocalSaveMode.updateLocal',
      ]) {
        expect(
          body.contains(mode),
          isTrue,
          reason:
              '_onSave no longer references $mode. Routing on '
              'trail.id.isEmpty alone creates a second server trail when a '
              'hiker re-saves an unsynced draft (RESEARCH.md Pitfall 2).',
        );
      }
    },
  );

  test(
    'the createLocal and updateLocal branches of _onSave never reference trailSaveProvider',
    () {
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(
        saveStart,
        isNot(-1),
        reason:
            '_onSave was renamed or its signature changed. Re-point this '
            'gate rather than deleting it -- the invariant still matters.',
      );

      final createStart = source.indexOf(
        'if (saveMode == LocalSaveMode.createLocal) {',
        saveStart,
      );
      expect(
        createStart,
        isNot(-1),
        reason:
            'The createLocal branch is gone from _onSave. Re-point this '
            'gate if the local-first branches moved elsewhere.',
      );

      // Everything from the createLocal branch to the end of _onSave covers
      // both local-first branches (createLocal, then updateLocal), and
      // deliberately excludes the earlier networkUpdate branch, which is
      // the one branch allowed to reach the network.
      final localBranches = source.substring(createStart, methodEnd(saveStart));

      expect(
        localBranches.contains('trailSaveProvider'),
        isFalse,
        reason:
            'A local save that touches the network reintroduces the '
            'offline save failure REC-01 removes.',
      );
    },
  );

  test(
    'the updateLocal branch reaches the network on alreadySynced, missing OR '
    'alreadyUploaded -- and via ONLY that three-outcome guard (WR-05)',
    () {
      // WR-05: this test previously asserted only that the literal
      // `LocalUpdateOutcome.alreadySynced` preceded the single
      // `_saveViaNetwork(` call, under the name "...ONLY on the
      // alreadySynced escape hatch". That was factually wrong the moment the
      // guard grew a second and third outcome (missing, alreadyUploaded) --
      // it kept passing for a reason unrelated to the property its name
      // claimed. This replacement names and asserts all three outcomes the
      // guard actually checks.
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      final createStart = source.indexOf(
        'if (saveMode == LocalSaveMode.createLocal) {',
        saveStart,
      );
      final localBranches = source.substring(createStart, methodEnd(saveStart));

      // Exactly one _saveViaNetwork call across BOTH local-first branches --
      // the createLocal branch has none, and the updateLocal branch's single
      // call is shared by all three outcomes below. This slice deliberately
      // excludes _onSave's earlier `networkUpdate` branch, which is the one
      // branch allowed to reach the network unconditionally (it IS the
      // network path) -- a separate gate below constrains what THAT branch
      // passes to _saveViaNetwork.
      final networkCalls = '_saveViaNetwork('.allMatches(localBranches).length;
      expect(
        networkCalls,
        1,
        reason:
            'Expected exactly one _saveViaNetwork call across the '
            'createLocal/updateLocal branches of _onSave. Any other network '
            'reach from a local save reintroduces REC-01. (This count '
            'deliberately excludes _onSave\'s networkUpdate branch, which '
            'sits before this slice starts.)',
      );

      final callIndex = localBranches.indexOf('_saveViaNetwork(');

      const outcomes = [
        'LocalUpdateOutcome.alreadySynced',
        'LocalUpdateOutcome.missing',
        'LocalUpdateOutcome.alreadyUploaded',
      ];
      for (final outcome in outcomes) {
        final idx = localBranches.indexOf(outcome);
        expect(
          idx,
          isNot(-1),
          reason:
              'The updateLocal branch\'s network-routing guard no longer '
              'names $outcome. A row in that state would silently fall '
              'through to _finishLocalSave -- a success toast over a write '
              'that never happened.',
        );
        expect(
          idx,
          lessThan(callIndex),
          reason:
              '$outcome must be checked BEFORE the _saveViaNetwork call it '
              'gates, or a row in that state falls through to '
              '_finishLocalSave first.',
        );
      }

      // Pin that the guard checks ONLY these three outcomes and nothing else
      // (in particular not `LocalUpdateOutcome.updated`, which is the
      // ordinary local-write path and must never reach the network). Slicing
      // from the nearest `if (` before the guard to the `_saveViaNetwork(`
      // call and counting `outcome ==` occurrences catches a guard that
      // silently grew a fourth condition.
      final outcomeAssignIdx = localBranches.indexOf(
        'final outcome = updateLocalTrail(',
      );
      expect(outcomeAssignIdx, isNot(-1));
      final guardIfIdx = localBranches.indexOf('if (', outcomeAssignIdx);
      expect(guardIfIdx, isNot(-1));
      final guardSlice = localBranches.substring(guardIfIdx, callIndex);
      final outcomeCheckCount = RegExp(
        r'outcome ==',
      ).allMatches(guardSlice).length;
      expect(
        outcomeCheckCount,
        3,
        reason:
            'The guard between `if (` and the _saveViaNetwork( call it gates '
            'should compare `outcome` against exactly three values '
            '(alreadySynced, missing, alreadyUploaded). A different count '
            'means either a new outcome slipped in unnoticed, or one of the '
            'three named checks above is matching something other than the '
            'live guard (e.g. a comment).',
      );
    },
  );

  test('_finishLocalSave calls _invalidateOwnTrailsList()', () {
    final start = source.indexOf('Future<void> _finishLocalSave(');
    expect(
      start,
      isNot(-1),
      reason:
          '_finishLocalSave was renamed or its signature changed. '
          'Re-point this gate rather than deleting it -- the invariant '
          'still matters.',
    );

    final body = source.substring(start, methodEnd(start));
    expect(
      body.contains('_invalidateOwnTrailsList()'),
      isTrue,
      reason:
          'Without this call an offline edit commits to ObjectBox but '
          'never notifies the own-trails list, which stays mounted '
          'beneath this pushed route and holds its pre-edit snapshot '
          'until a manual pull-to-refresh (UAT gap 3, 36-10).',
    );
  });

  test('_saveViaNetwork calls _invalidateOwnTrailsList()', () {
    final start = source.indexOf('Future<void> _saveViaNetwork(');
    expect(
      start,
      isNot(-1),
      reason:
          '_saveViaNetwork was renamed or its signature changed. '
          'Re-point this gate rather than deleting it -- the invariant '
          'still matters.',
    );

    final body = source.substring(start, methodEnd(start));
    expect(
      body.contains('_invalidateOwnTrailsList()'),
      isTrue,
      reason:
          'Without this call an already-uploaded trail\'s edit never '
          'notifies the own-trails list, which stays mounted beneath '
          'this pushed route and holds its pre-edit snapshot until a '
          'manual pull-to-refresh (UAT gap 3, 36-10).',
    );
  });

  test(
    '_invalidateOwnTrailsList invalidates the same pair the drain invalidates',
    () {
      final start = source.indexOf('void _invalidateOwnTrailsList() {');
      expect(
        start,
        isNot(-1),
        reason:
            '_invalidateOwnTrailsList was renamed or removed. Re-point '
            'this gate rather than deleting it -- the invariant still '
            'matters.',
      );

      final body = source.substring(start, methodEnd(start));

      expect(
        body.contains('invalidate(trailLibraryProvider)'),
        isTrue,
        reason:
            '_invalidateOwnTrailsList must invalidate trailLibraryProvider, '
            'matching trail_sync_provider.dart\'s post-drain invalidation '
            'verbatim, or a local save and an upload diverge.',
      );
      expect(
        body.contains('invalidate(') && body.contains('profileTrailsProvider('),
        isTrue,
        reason:
            '_invalidateOwnTrailsList must invalidate profileTrailsProvider, '
            'matching trail_sync_provider.dart\'s post-drain invalidation '
            'verbatim, or a local save and an upload diverge.',
      );
    },
  );

  test('_onSave routes on resolveLocalSaveModeForRow(', () {
    final saveStart = source.indexOf(
      'Future<void> _onSave(BuildContext context) async {',
    );
    expect(
      saveStart,
      isNot(-1),
      reason:
          '_onSave was renamed or its signature changed. Re-point this '
          'gate rather than deleting it -- the invariant still matters.',
    );

    final body = source.substring(saveStart, methodEnd(saveStart));

    // Plain `resolveLocalSaveMode` cannot tell "the row is gone because it
    // uploaded" from "this trail was never saved anywhere" -- it falls back
    // to a snapshot captured while `syncState` was still `pending`, and
    // therefore routes a post-upload edit into a local write against a row
    // that no longer exists -- a green success toast over a discarded edit.
    //
    // The older `body.contains('resolveLocalSaveMode')` assertion above
    // still passes on the substring and is deliberately left in place as
    // the weaker, more stable guard.
    expect(
      body.contains('resolveLocalSaveModeForRow('),
      isTrue,
      reason:
          '_onSave no longer routes through resolveLocalSaveModeForRow. '
          'Plain resolveLocalSaveMode cannot tell "the row is gone because '
          'it uploaded" from "this trail was never saved anywhere" -- it '
          'falls back to a stale pending snapshot and routes a post-upload '
          'edit into a local write against a row that no longer exists, a '
          'green success toast over a discarded edit.',
    );
  });

  test(
    'the updateLocal branch treats LocalUpdateOutcome.missing as a network case',
    () {
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(saveStart, isNot(-1));

      final outcomeIdx = source.indexOf(
        'final outcome = updateLocalTrail(',
        saveStart,
      );
      expect(
        outcomeIdx,
        isNot(-1),
        reason:
            'Could not find the updateLocalTrail call in _onSave. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final remainder = source.substring(outcomeIdx, methodEnd(saveStart));

      expect(
        remainder.contains('LocalUpdateOutcome.missing'),
        isTrue,
        reason:
            'A `missing` outcome falling through to _finishLocalSave shows '
            'trail_saved_successfully for a write ObjectBox declined to '
            'make, which is exactly the failure mode the CR-04 routing fix '
            'exists to prevent.',
      );
      expect(
        remainder.contains('_saveViaNetwork('),
        isTrue,
        reason:
            'The missing-outcome branch must still route to the network '
            'save path.',
      );

      final missingIdx = remainder.indexOf('LocalUpdateOutcome.missing');
      final finishIdx = remainder.indexOf('_finishLocalSave(');
      expect(
        missingIdx < finishIdx,
        isTrue,
        reason:
            'LocalUpdateOutcome.missing must be checked BEFORE '
            '_finishLocalSave is called, or a missing row still falls '
            'through to the success toast.',
      );
    },
  );

  test(
    'the createLocal branch publishes _localId only after saveNewLocalTrail '
    'succeeds (CR-02)',
    () {
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      final createStart = source.indexOf(
        'if (saveMode == LocalSaveMode.createLocal) {',
        saveStart,
      );
      expect(createStart, isNot(-1));

      final createBranch = source.substring(createStart, methodEnd(saveStart));

      final saveNewLocalTrailIdx = createBranch.indexOf(
        'saveNewLocalTrail(',
      );
      final publishIdx = createBranch.indexOf('_localId = localId;');
      expect(
        saveNewLocalTrailIdx,
        isNot(-1),
        reason:
            'Could not find the saveNewLocalTrail( call in the createLocal '
            'branch. Re-point this gate rather than deleting it -- the '
            'invariant still matters.',
      );
      expect(
        publishIdx,
        isNot(-1),
        reason:
            'Could not find `_localId = localId;` in the createLocal '
            'branch. Re-point this gate rather than deleting it -- the '
            'invariant still matters.',
      );
      expect(
        saveNewLocalTrailIdx < publishIdx,
        isTrue,
        reason:
            '_localId must be published AFTER saveNewLocalTrail succeeds. '
            '_copyPhotosForLocalSave and saveNewLocalTrail can both throw '
            '(filesystem, ObjectBox write), and publishing _localId before '
            'either succeeds leaves it pointing at an id with no row -- '
            'the next save then sees `persistedLocalId != null && '
            'persisted == null`, indistinguishable from a row a completed '
            'upload retired, and routes into CR-01\'s dead POST, bricking '
            'the screen for the rest of its life (CR-02).',
      );
    },
  );

  test(
    '_saveViaNetwork refuses to run with a blank trail id, and the refusal '
    'actually returns and names the recoverable state (CR-01, WR-05)',
    () {
      // WR-05's falsifying rewrite for the ORIGINAL version of this gate:
      // `if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }` -- an empty
      // guard body that falls through to the POST below it. The original
      // gate only checked that the substring `trailHasServerId(` appeared
      // before the substring `trailSaveProvider.notifier)`, which that
      // rewrite still satisfies verbatim. This version additionally requires
      // the guarded slice to actually `return;`, to actually surface the
      // named recoverable-state message, and to NOT surface the generic
      // `error_saving_trail` -- none of which the empty-body rewrite can
      // satisfy. Demonstrated against a scratch copy; see
      // 36-20-SUMMARY.md for the observed failure.
      final start = source.indexOf('Future<void> _saveViaNetwork(');
      expect(
        start,
        isNot(-1),
        reason:
            '_saveViaNetwork was renamed or its signature changed. '
            'Re-point this gate rather than deleting it -- the invariant '
            'still matters.',
      );

      final body = source.substring(start, methodEnd(start));

      expect(
        body.contains('trailHasServerId('),
        isTrue,
        reason:
            '_saveViaNetwork must refuse to run (rather than POST '
            '`/trail/form/` with an empty id) when its `updatedTrail` '
            'snapshot is stale -- D-06 blanks a local-sentinel id, and a '
            'screen that never re-read across a completed upload can '
            'still hold one. An empty-id POST can never be routed '
            '(SvelteKit normalizes the empty [id] segment away) and also '
            'reads photo files retirement already deleted from disk.',
      );

      final guardIdx = body.indexOf('if (!trailHasServerId(');
      final networkCallIdx = body.indexOf('trailSaveProvider.notifier)');
      expect(
        guardIdx != -1 && networkCallIdx != -1 && guardIdx < networkCallIdx,
        isTrue,
        reason:
            'The `if (!trailHasServerId(...))` guard must precede the '
            'network call it protects, or a blank-id POST can still slip '
            'through.',
      );

      // The slice from the guard's `if (` to the network call it gates is
      // the guard's ENTIRE body plus everything up to the protected call --
      // exactly the region a "TODO, fall through" rewrite would hollow out.
      final guardedSlice = body.substring(guardIdx, networkCallIdx);

      expect(
        guardedSlice.contains('return;'),
        isTrue,
        reason:
            'The blank-id guard must actually `return;` rather than fall '
            'through to the POST below it. A guard with an empty body (e.g. '
            '`if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }`) '
            'restores the un-routable blank-id POST while this substring '
            'check alone would previously still have passed (WR-05).',
      );
      expect(
        guardedSlice.contains('l10n.error_saving_trail'),
        isFalse,
        reason:
            'The refusal must not reuse the generic error_saving_trail '
            'string -- that is the exact regression CR-01 was raised '
            'against: a 100%-reproducible primary-flow failure showing the '
            'same message as a 500, a timeout or a malformed payload, with '
            'no indication the trail is actually fine and simply needs '
            're-opening.',
      );
      expect(
        guardedSlice.contains('trail_uploaded_reopen_to_edit'),
        isTrue,
        reason:
            'The refusal must name the actual recoverable state via the '
            'dedicated trail_uploaded_reopen_to_edit message, per the '
            'previous review\'s stated minimum ("refuse the save and tell '
            'the user to re-open the trail").',
      );
    },
  );

  test(
    'the updateLocal branch also treats LocalUpdateOutcome.alreadyUploaded '
    'as a network case, checked inside the guard condition itself (CR-03, '
    'WR-05)',
    () {
      // WR-05's falsifying rewrite for the ORIGINAL version of this gate:
      // moving `LocalUpdateOutcome.alreadyUploaded` out of the `if (...)`
      // condition and into an unrelated `debugPrint(...)` call placed
      // before `_finishLocalSave(`. The original gate only checked that the
      // substring appeared before `_finishLocalSave(`, which that rewrite
      // still satisfies verbatim. This version additionally requires the
      // token to sit strictly between an `if (` and its matching `) {`,
      // which a debugPrint placement cannot satisfy. Demonstrated against a
      // scratch copy; see 36-20-SUMMARY.md for the observed failure.
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(saveStart, isNot(-1));

      final outcomeIdx = source.indexOf(
        'final outcome = updateLocalTrail(',
        saveStart,
      );
      expect(outcomeIdx, isNot(-1));

      final remainder = source.substring(outcomeIdx, methodEnd(saveStart));

      expect(
        remainder.contains('LocalUpdateOutcome.alreadyUploaded'),
        isTrue,
        reason:
            'A row the drain\'s create step already stamped a real server '
            'id onto (`writeServerTrailId`, well before the row is '
            'retired) can sit as `pending`/`uploading`/`failed` for as '
            'long as a later step keeps failing. Without this outcome, an '
            'edit saved while the row is in that window is written '
            'locally and later destroyed by retirement with no trace '
            '(CR-03) -- `updateLocalTrail` returns `alreadyUploaded` for '
            'exactly this row shape, and this branch must route it to the '
            'network save alongside `missing` and `alreadySynced`.',
      );

      final missingIdx = remainder.indexOf('LocalUpdateOutcome.missing');
      final uploadedIdx = remainder.indexOf(
        'LocalUpdateOutcome.alreadyUploaded',
      );
      final finishIdx = remainder.indexOf('_finishLocalSave(');
      expect(
        missingIdx != -1 && uploadedIdx != -1 && finishIdx != -1,
        isTrue,
      );
      expect(
        uploadedIdx < finishIdx,
        isTrue,
        reason:
            'LocalUpdateOutcome.alreadyUploaded must be checked BEFORE '
            '_finishLocalSave is called, or the row still falls through '
            'to the success toast.',
      );

      // WR-05's specific strengthening: the token must sit INSIDE an `if (`
      // condition (i.e. between the nearest preceding `if (` and its
      // following `) {`), not merely somewhere before _finishLocalSave(.
      // Moving it into `debugPrint('...alreadyUploaded...')` placed earlier
      // in the method would satisfy every assertion above but fail this one.
      final ifIdx = remainder.lastIndexOf('if (', uploadedIdx);
      expect(
        ifIdx,
        isNot(-1),
        reason:
            'No `if (` precedes the LocalUpdateOutcome.alreadyUploaded '
            'reference -- it is no longer part of a conditional at all.',
      );
      final closeIdx = remainder.indexOf(') {', uploadedIdx);
      expect(
        closeIdx,
        isNot(-1),
        reason:
            'No `) {` follows the LocalUpdateOutcome.alreadyUploaded '
            'reference -- it is not inside a guard condition.',
      );
      expect(
        uploadedIdx > ifIdx && uploadedIdx < closeIdx,
        isTrue,
        reason:
            'LocalUpdateOutcome.alreadyUploaded must sit strictly between '
            'an `if (` and its matching `) {` -- i.e. inside the guard '
            'condition that actually gates _saveViaNetwork, not merely '
            'somewhere earlier in the method (e.g. a debugPrint mentioning '
            'the outcome name). A mention that is not part of the live '
            'condition does nothing to route the save.',
      );
    },
  );

  test(
    'the networkUpdate branch resolves a real server id before calling '
    '_saveViaNetwork, and never hands it the stale local-sentinel id '
    '(WR-05 -- what is passed to _saveViaNetwork)',
    () {
      // WR-05's central finding: none of the pre-36-20 gates in this file
      // constrained what is PASSED to _saveViaNetwork, which is where CR-01,
      // CR-03 and WR-13 all actually live. This gate targets the
      // networkUpdate branch specifically -- the one branch that resolves an
      // id BEFORE the network call, as opposed to the updateLocal branch's
      // already-uploaded escape hatch, which passes `updatedTrail` (whose id
      // is real by construction of that branch's own guard) unchanged.
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(saveStart, isNot(-1));

      final networkUpdateStart = source.indexOf(
        'if (saveMode == LocalSaveMode.networkUpdate) {',
        saveStart,
      );
      expect(
        networkUpdateStart,
        isNot(-1),
        reason:
            'The networkUpdate branch is gone from _onSave. Re-point this '
            'gate if the branch moved elsewhere.',
      );

      final createStart = source.indexOf(
        'if (saveMode == LocalSaveMode.createLocal) {',
        networkUpdateStart,
      );
      expect(createStart, isNot(-1));

      // The networkUpdate branch runs from its own `if (` to the start of
      // the next branch (createLocal), which is exactly its extent -- it is
      // the first branch checked and unconditionally `return`s.
      final networkUpdateSlice = source.substring(
        networkUpdateStart,
        createStart,
      );

      expect(
        networkUpdateSlice.contains('resolveNetworkSaveTarget('),
        isTrue,
        reason:
            'The networkUpdate branch must resolve its save target via '
            'resolveNetworkSaveTarget (screen id, then the retired-id '
            'fallback, then null meaning refuse) rather than trusting this '
            'screen\'s own possibly-stale `updatedTrail.id` directly.',
      );

      final callMatches = RegExp(
        r'_saveViaNetwork\(',
      ).allMatches(networkUpdateSlice).toList();
      expect(
        callMatches,
        hasLength(1),
        reason:
            'Expected exactly one _saveViaNetwork( call in the networkUpdate '
            'branch.',
      );

      final copyWithMatches = 'updatedTrail.copyWith(id: targetId'
          .allMatches(networkUpdateSlice)
          .toList();
      expect(
        copyWithMatches,
        hasLength(callMatches.length),
        reason:
            'Every _saveViaNetwork( call in the networkUpdate branch must '
            'be handed `updatedTrail.copyWith(id: targetId, ...)`, not the '
            'raw `updatedTrail` -- `updatedTrail.id` can still be the blank '
            'local-sentinel value here (D-06), and resolveNetworkSaveTarget '
            'resolved the REAL id into `targetId` specifically so it could '
            'be substituted in. Passing `updatedTrail` unchanged would hand '
            'a blank id straight to the network path (CR-01).',
      );

      final callIdx = networkUpdateSlice.indexOf('_saveViaNetwork(');
      final copyWithIdx = networkUpdateSlice.indexOf(
        'updatedTrail.copyWith(id: targetId',
      );
      expect(
        copyWithIdx > callIdx && copyWithIdx - callIdx < 350,
        isTrue,
        reason:
            'updatedTrail.copyWith(id: targetId, ...) must appear as an '
            'argument of THIS _saveViaNetwork( call (within a small window '
            'after it opens -- a doc comment between the call and its '
            'argument accounts for most of this window), not somewhere '
            'unrelated later in the branch.',
      );

      expect(
        networkUpdateSlice.contains('localPhotos: const []'),
        isTrue,
        reason:
            'The id substitution must also clear localPhotos -- retirement '
            'already deleted unsynced/<localId>/, so those local paths no '
            'longer resolve and a network save that still lists them would '
            'throw reading a file that no longer exists.',
      );
      expect(
        networkUpdateSlice.contains('existsSync()'),
        isTrue,
        reason:
            'The picked-photo file list passed to _saveViaNetwork must be '
            'filtered through existsSync() -- the same retirement-deleted-'
            'the-directory hazard applies to newPhotoFiles.',
      );
    },
  );

  test(
    '_saveViaNetwork reconciles the local row AFTER adopting the network '
    'result and BEFORE invalidating the own-trails list (CR-03)',
    () {
      final start = source.indexOf('Future<void> _saveViaNetwork(');
      expect(start, isNot(-1));

      final body = source.substring(start, methodEnd(start));

      expect(
        body.contains('reconcileLocalId'),
        isTrue,
        reason:
            '_saveViaNetwork must accept a reconcileLocalId parameter so '
            'callers with a persisted row can have it reconciled to the '
            'server-accepted edit -- without it, a save in the '
            'alreadyUploaded window reaches the server but leaves the local '
            'row (and therefore the own-trails list, which dedupes against '
            'that row\'s id) showing the pre-edit values indefinitely '
            '(CR-03).',
      );

      final resultAdoptIdx = body.indexOf('trail = result.trail');
      final reconcileCallIdx = body.indexOf('applyNetworkEditToLocalRow(');
      final invalidateIdx = body.indexOf('_invalidateOwnTrailsList();');
      expect(
        resultAdoptIdx != -1 && reconcileCallIdx != -1 && invalidateIdx != -1,
        isTrue,
        reason:
            'Could not find one of trail = result.trail / '
            'applyNetworkEditToLocalRow( / _invalidateOwnTrailsList(); in '
            '_saveViaNetwork. Re-point this gate rather than deleting it -- '
            'the invariant still matters.',
      );
      expect(
        resultAdoptIdx < reconcileCallIdx && reconcileCallIdx < invalidateIdx,
        isTrue,
        reason:
            'applyNetworkEditToLocalRow( must run strictly between adopting '
            'result.trail and invalidating the own-trails list. Reconciling '
            'before the network result is adopted would write stale data; '
            'invalidating before reconciling would let the own-trails list '
            're-read the row\'s pre-edit values one more time -- the exact '
            'CR-03 window under a green success toast.',
      );
    },
  );

  test(
    'photosNotYetOnServer filters the network photo payload only -- the '
    'createLocal branch never calls it (WR-13)',
    () {
      final saveStart = source.indexOf(
        'Future<void> _onSave(BuildContext context) async {',
      );
      expect(saveStart, isNot(-1));

      final body = source.substring(saveStart, methodEnd(saveStart));
      expect(
        body.contains('photosNotYetOnServer('),
        isTrue,
        reason:
            '_onSave must filter picked photo paths through '
            'photosNotYetOnServer before handing them to a network save -- '
            'without it, a save in the alreadyUploaded window re-sends '
            'every photo the drain already uploaded under the append-only '
            '`photos+` key, doubling the server-side set on every save '
            '(WR-13).',
      );

      final createStart = source.indexOf(
        'if (saveMode == LocalSaveMode.createLocal) {',
        saveStart,
      );
      expect(createStart, isNot(-1));
      final updateLocalCommentIdx = source.indexOf(
        '// LocalSaveMode.updateLocal',
        createStart,
      );
      expect(
        updateLocalCommentIdx,
        isNot(-1),
        reason:
            'Could not find the boundary comment marking the start of the '
            'updateLocal branch. Re-point this gate rather than deleting '
            'it -- the invariant still matters.',
      );

      final createLocalOnly = source.substring(
        createStart,
        updateLocalCommentIdx,
      );
      expect(
        createLocalOnly.contains('photosNotYetOnServer('),
        isFalse,
        reason:
            'photosNotYetOnServer belongs on the network photo payload '
            'only -- the createLocal branch writes every picked photo '
            'straight to the local row via _copyPhotosForLocalSave and '
            'must never filter against what the (nonexistent, for a brand '
            'new trail) server copy already has.',
      );
    },
  );
}
