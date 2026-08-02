import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Source-level guard for two structural invariants introduced by
/// local-first recording (36-06):
///
/// 1. Every not-yet-uploaded waypoint must carry an empty server id (plus a
///    local key) rather than a timestamp-derived synthetic id.
/// 2. `_onSave` must route on `resolveLocalSaveMode`/`LocalSaveMode`, and its
///    two local-first branches must never touch `trailSaveProvider`.
///
/// `trail_create_screen.dart` needs auth, a router, a live ObjectBox store,
/// an image picker and a map controller to drive behaviourally, so these
/// invariants are guarded at source level instead -- the same rationale and
/// form as `test/components/trail/trail_dropdown_delete_gate_test.dart`'s
/// delete-branch gate and the PORT-03 gate in
/// `test/util/trail_import_util_test.dart`.
void main() {
  final libDir = Directory('lib');
  final source = File(
    'lib/routes/trail_create_screen.dart',
  ).readAsStringSync();

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

      final saveEnd = source.indexOf(
        '\n  bool get _hasUnsavedChanges',
        saveStart,
      );
      expect(
        saveEnd,
        isNot(-1),
        reason: 'Could not find the end of _onSave (the next member after it).',
      );

      final body = source.substring(saveStart, saveEnd);

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

      final saveEnd = source.indexOf(
        '\n  bool get _hasUnsavedChanges',
        saveStart,
      );
      expect(
        saveEnd,
        isNot(-1),
        reason: 'Could not find the end of _onSave (the next member after it).',
      );

      // Everything from the createLocal branch to the end of _onSave covers
      // both local-first branches (createLocal, then updateLocal), and
      // deliberately excludes the earlier networkUpdate branch, which is
      // the one branch allowed to reference trailSaveProvider.
      final localBranches = source.substring(createStart, saveEnd);

      expect(
        localBranches.contains('trailSaveProvider'),
        isFalse,
        reason:
            'A local save that touches the network reintroduces the '
            'offline save failure REC-01 removes.',
      );
    },
  );
}
