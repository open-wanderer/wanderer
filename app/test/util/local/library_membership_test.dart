import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/local/library_membership.dart';

// ---------------------------------------------------------------------------
// Tests for the offline library's per-account membership rules.
//
// These pin the reported regression: log in as A, download a trail, log out,
// log in as B (trail must be hidden), log back in as A -- the trail must
// STILL be there. It used to be deleted on logout, which destroyed it for A
// permanently.
//
// A trail is one row and one copy of its files shared by every account that
// downloaded it, so these two functions are the whole sharing contract. The
// Store-backed paths that use them (`trail_library_provider.dart`,
// `trail_download_service.dart`) cannot be exercised under `flutter test` --
// this repo has no way to construct a real ObjectBox Store in a unit test --
// so the rules are tested here and the wiring is covered by the on-device
// checks in the plan.
// ---------------------------------------------------------------------------

void main() {
  group('libraryMembersAfterSave', () {
    test('adds the downloading account to an empty set', () {
      expect(libraryMembersAfterSave(const [], 'a'), ['a']);
    });

    test('a second account downloading the same trail joins, never replaces', () {
      final members = libraryMembersAfterSave(const ['a'], 'b');
      expect(members, containsAll(<String>['a', 'b']));
      expect(members, hasLength(2));
    });

    test('re-downloading is idempotent — no duplicate id', () {
      // A duplicate would keep the delete-time reference count above zero
      // forever, so the row and its files could never be reclaimed.
      expect(libraryMembersAfterSave(const ['a'], 'a'), ['a']);
      expect(
        libraryMembersAfterSave(libraryMembersAfterSave(const ['a'], 'a'), 'a'),
        ['a'],
      );
    });

    test('a null account id grants nobody access', () {
      // Signed out: the trail must not land in every library.
      expect(libraryMembersAfterSave(const [], null), isEmpty);
      expect(libraryMembersAfterSave(const ['a'], null), ['a']);
    });
  });

  group('libraryMembersAfterDelete', () {
    test('removing the only holder empties the set', () {
      // Empty is the signal to delete the row and library/<id>/.
      expect(libraryMembersAfterDelete(const ['a'], 'a'), isEmpty);
    });

    test("one account's delete leaves the other account's copy intact", () {
      expect(libraryMembersAfterDelete(const ['a', 'b'], 'b'), ['a']);
    });

    test('deleting as an account that never held it changes nothing', () {
      expect(libraryMembersAfterDelete(const ['a'], 'b'), ['a']);
    });

    test('the full reported sequence: A downloads, B joins, both leave', () {
      var members = libraryMembersAfterSave(const [], 'a');
      expect(members, ['a']);

      // B signs in. A's trail is still recorded, just not B's to see.
      expect(members.contains('b'), isFalse);

      // B downloads the same trail.
      members = libraryMembersAfterSave(members, 'b');
      expect(members, hasLength(2));

      // B deletes it — A must keep a working offline copy, so the files stay.
      members = libraryMembersAfterDelete(members, 'b');
      expect(members, ['a']);
      expect(members, isNotEmpty, reason: 'files must NOT be deleted yet');

      // A deletes it too — now nothing holds it and the files can go.
      members = libraryMembersAfterDelete(members, 'a');
      expect(members, isEmpty);
    });
  });
}
