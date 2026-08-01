import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/xyz_tile_bounds.dart';

void main() {
  group('tileToBounds', () {
    test('z0/x0/y0 returns Web-Mercator world bounds', () {
      final bounds = tileToBounds(0, 0, 0);

      expect(bounds.west, -180);
      expect(bounds.east, 180);
      expect(bounds.north, closeTo(85.051, 0.001));
      expect(bounds.south, closeTo(-85.051, 0.001));
    });

    test('increasing x increases west/east for fixed z', () {
      const z = 4;
      final tile0 = tileToBounds(z, 0, 0);
      final tile1 = tileToBounds(z, 1, 0);

      expect(tile1.west, greaterThan(tile0.west));
      expect(tile1.east, greaterThan(tile0.east));
    });

    test('increasing y decreases north/south for fixed z', () {
      const z = 4;
      final tile0 = tileToBounds(z, 0, 0);
      final tile1 = tileToBounds(z, 0, 1);

      expect(tile1.north, lessThan(tile0.north));
      expect(tile1.south, lessThan(tile0.south));
    });
  });
}
