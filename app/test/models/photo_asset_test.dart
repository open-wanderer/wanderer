import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/photo_asset.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/models/summit_log.dart';

void main() {
  test('trail summaries show the cover selected by the server', () {
    final trail = Trail.empty().copyWith(
      photos: ['preview.svg', 'photo.jpg'],
      thumbnail: 1,
    );
    expect(trail.summaryThumbnail, 'photo.jpg');
    expect(trail.copyWith(thumbnail: -1).summaryThumbnail, 'preview.svg');
    expect(trail.copyWith(thumbnail: 5).summaryThumbnail, 'preview.svg');
    expect(trail.copyWith(photos: []).summaryThumbnail, '');
  });

  const linked = {
    'asset': 'asset0000000001',
    'is_thumbnail': true,
    'expand': {
      'asset': {
        'id': 'asset0000000001',
        'collectionId': 'assets',
        'file': 'photo.svg',
        'metadata': {
          'generated': {'kind': 'route-preview'},
        },
      },
    },
  };

  test(
    'equal link responses do not make unchanged waypoint data look edited',
    () {
      final first = WaypointExpand.fromJson({
        'waypoint_assets_via_waypoint': [linked],
      });
      final second = WaypointExpand.fromJson({
        'waypoint_assets_via_waypoint': [linked],
      });
      expect(first, second);
      expect(first.hashCode, second.hashCode);
    },
  );

  test('retains linked photo markers in each response model', () {
    final trail = TrailExpand.fromJson({
      'trail_assets_via_trail': [linked],
    });
    final waypoint = WaypointExpand.fromJson({
      'waypoint_assets_via_waypoint': [linked],
    });
    final log = SummitLogExpand.fromJson({
      'summit_log_assets_via_summit_log': [linked],
    });
    for (final links in [
      trail.photoAssets!,
      waypoint.photoAssets!,
      log.photoAssets!,
    ]) {
      expect(links.single.asset, 'asset0000000001');
      expect(links.single.isThumbnail, isTrue);
      expect(links.single.photoAsset!.isGeneratedRoutePreview, isTrue);
      expect(
        links.single.photoAsset!.url,
        '/api/v1/files/assets/asset0000000001/photo.svg',
      );
    }
  });

  test(
    'recognizes only the reserved unowned preview names without a marker',
    () {
      expect(
        const PhotoAsset(
          id: 'id',
          file: 'wanderer_route_preview_abcdefghij.webp',
        ).isGeneratedRoutePreview,
        isTrue,
      );
      for (final asset in [
        const PhotoAsset(id: 'id', file: 'route_abcdef1234.png'),
        const PhotoAsset(id: 'id', file: 'user.svg'),
        const PhotoAsset(id: 'id', file: 'wanderer-route-preview-holiday.png'),
        const PhotoAsset(
          id: 'id',
          file: 'wanderer_route_preview_abcdefghij.webp',
          externalProvider: 'provider',
        ),
        const PhotoAsset(
          id: 'id',
          file: 'wanderer_route_preview_abcdefghij.webp',
          takenAt: '2026-01-01',
        ),
      ]) {
        expect(asset.isGeneratedRoutePreview, isFalse);
      }
    },
  );

  test('uses the asset media endpoint when no local filename exists', () {
    expect(
      const PhotoAsset(id: 'asset0000000001').url,
      '/api/v1/assets/asset0000000001/file',
    );
  });
}
