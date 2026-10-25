import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/server_compatibility.dart';

Map<String, dynamic> nodeInfo({
  Object? version = '0.21.0',
  String name = 'wanderer',
}) => {
  'version': '2.1',
  'software': {'name': name, 'version': version},
};

void main() {
  group('isCompatibleNodeInfo', () {
    test('accepts the minimum version', () {
      expect(
        isCompatibleNodeInfo(
          200,
          nodeInfo(version: kMinimumServerVersion.join('.')),
        ),
        isTrue,
      );
    });

    test('accepts newer versions', () {
      expect(isCompatibleNodeInfo(200, nodeInfo(version: '0.22.0')), isTrue);
      expect(isCompatibleNodeInfo(200, nodeInfo(version: '1.0.0')), isTrue);
      expect(isCompatibleNodeInfo(200, nodeInfo(version: 'v0.21.3')), isTrue);
    });

    test('ignores a pre-release suffix', () {
      expect(
        isCompatibleNodeInfo(200, nodeInfo(version: '0.22.0-rc.1')),
        isTrue,
      );
    });

    test('rejects an older version', () {
      expect(isCompatibleNodeInfo(200, nodeInfo(version: '0.20.9')), isFalse);
    });

    test('accepts an unparseable version, e.g. a development build', () {
      expect(isCompatibleNodeInfo(200, nodeInfo(version: 'dev')), isTrue);
      expect(isCompatibleNodeInfo(200, nodeInfo(version: null)), isTrue);
    });

    test('rejects a server without NodeInfo', () {
      expect(isCompatibleNodeInfo(404, '<!doctype html>'), isFalse);
    });

    test('rejects other software', () {
      expect(isCompatibleNodeInfo(200, nodeInfo(name: 'mastodon')), isFalse);
    });

    test('rejects a non-JSON success, e.g. a login redirect target', () {
      expect(isCompatibleNodeInfo(200, '<!doctype html>'), isFalse);
    });

    test('is undecided on a server error', () {
      expect(isCompatibleNodeInfo(500, {'message': 'boom'}), isNull);
      expect(isCompatibleNodeInfo(503, null), isNull);
    });
  });
}
