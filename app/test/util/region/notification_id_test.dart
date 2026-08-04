import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/region/notification_id.dart';

void main() {
  group('regionNotificationId', () {
    test('is stable across calls for the same package', () {
      expect(
        regionNotificationId('Europe/Germany/Bavaria', dem: false),
        regionNotificationId('Europe/Germany/Bavaria', dem: false),
      );
    });

    test('separates the vector and DEM packages of one region', () {
      // Both downloads run concurrently and independently; sharing an id
      // would make each overwrite the other's progress notification.
      expect(
        regionNotificationId('Europe/Germany/Bavaria', dem: false),
        isNot(regionNotificationId('Europe/Germany/Bavaria', dem: true)),
      );
    });

    test('separates different regions', () {
      expect(
        regionNotificationId('Europe/Germany/Bavaria', dem: false),
        isNot(regionNotificationId('Europe/Germany/Saxony', dem: false)),
      );
    });

    test('never collides with the trail notification id 42', () {
      for (final path in [
        '',
        'a',
        'Europe',
        'Europe/Germany/Bavaria',
        'Africa/Tanzania/Kilimanjaro',
        'x' * 500,
      ]) {
        for (final dem in [true, false]) {
          expect(regionNotificationId(path, dem: dem), isNot(42));
        }
      }
    });

    test('stays inside the signed 32-bit range both platforms require', () {
      for (final path in ['', 'x' * 500, 'Europe/Germany/Bavaria', '🏔/ü/ß']) {
        for (final dem in [true, false]) {
          final id = regionNotificationId(path, dem: dem);
          expect(id, greaterThan(0));
          expect(id, lessThan(2147483647));
        }
      }
    });

    test('spreads a realistic catalog across distinct ids', () {
      final ids = <int>{};
      for (var i = 0; i < 2000; i++) {
        ids.add(regionNotificationId('Europe/Country$i/Region$i', dem: false));
        ids.add(regionNotificationId('Europe/Country$i/Region$i', dem: true));
      }
      expect(ids.length, 4000);
    });
  });
}
