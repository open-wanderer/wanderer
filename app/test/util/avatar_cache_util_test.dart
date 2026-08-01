import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/avatar_cache_util.dart';

void main() {
  group('cachedAvatarFileName', () {
    test('embeds the user id and the basename for a plain remote filename', () {
      final name = cachedAvatarFileName('user123', 'avatar.png');
      expect(name, contains('user123'));
      expect(name, contains('avatar.png'));
    });

    test('strips a directory-traversal remote filename to its basename', () {
      final name = cachedAvatarFileName('user123', '../../evil.png');
      expect(name, isNot(contains('..')));
      expect(name, contains('evil.png'));
    });

    test('strips an absolute-path remote filename to its basename', () {
      final name = cachedAvatarFileName('user123', '/etc/passwd');
      expect(name, isNot(contains('/etc')));
      expect(name, contains('passwd'));
    });

    test('is stable across repeated calls with the same inputs', () {
      final a = cachedAvatarFileName('user123', 'avatar.png');
      final b = cachedAvatarFileName('user123', 'avatar.png');
      expect(a, b);
    });

    test('differs for the same filename under two different user ids', () {
      final a = cachedAvatarFileName('user123', 'avatar.png');
      final b = cachedAvatarFileName('user456', 'avatar.png');
      expect(a, isNot(b));
    });
  });

  group('cachedAvatarFile', () {
    test('reports no cached avatar for a null remote filename', () async {
      final file = await cachedAvatarFile('user123', null);
      expect(file, isNull);
    });

    test('reports no cached avatar for an empty remote filename', () async {
      final file = await cachedAvatarFile('user123', '');
      expect(file, isNull);
    });
  });
}
