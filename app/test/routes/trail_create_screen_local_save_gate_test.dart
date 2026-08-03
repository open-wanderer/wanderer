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

  test('the updateLocal branch reaches the network ONLY on the alreadySynced '
      'escape hatch (CR-04)', () {
    final saveStart = source.indexOf(
      'Future<void> _onSave(BuildContext context) async {',
    );
    final createStart = source.indexOf(
      'if (saveMode == LocalSaveMode.createLocal) {',
      saveStart,
    );
    final localBranches = source.substring(createStart, methodEnd(saveStart));

    // A trail the drain promoted to `synced` mid-save is no longer a
    // local-first save at all -- its row is on the server, and
    // `selectDrainCandidates` will never pick it up again. Writing the edit
    // locally would strand it on the device forever under a success toast,
    // so this one path delegates to _saveViaNetwork. It is the ONLY
    // sanctioned network reach out of these branches, and it must stay
    // guarded on that specific outcome.
    final networkCalls = '_saveViaNetwork('.allMatches(localBranches).length;
    expect(
      networkCalls,
      1,
      reason:
          'Expected exactly one _saveViaNetwork call in the local-first '
          'branches -- the alreadySynced escape hatch. Any other network '
          'reach from a local save reintroduces REC-01.',
    );

    final callIndex = localBranches.indexOf('_saveViaNetwork(');
    final guardIndex = localBranches.indexOf(
      'LocalUpdateOutcome.alreadySynced',
    );
    expect(
      guardIndex,
      isNot(-1),
      reason:
          'The _saveViaNetwork call lost its LocalUpdateOutcome.alreadySynced '
          'guard, so an ordinary unsynced re-save would now hit the network.',
    );
    expect(
      guardIndex,
      lessThan(callIndex),
      reason:
          'The alreadySynced guard must precede the _saveViaNetwork call it '
          'gates.',
    );
  });

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
}
