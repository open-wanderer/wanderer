import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/util/route_placeholder.dart';

class _Record with RecordFunctions {
  @override
  String get id => 'trail';
  @override
  String get collectionId => 'trails';
}

void main() {
  final record = _Record();
  const server = 'https://a.example/wanderer';

  test('resolves asset paths on the server', () {
    expect(
      record.getFileUrl(server, '/api/v1/files/assets/id/photo.jpg'),
      'https://a.example/api/v1/files/assets/id/photo.jpg',
    );
    expect(
      record.getFileUrl(server, 'api/v1/assets/id/file'),
      'https://a.example/wanderer/api/v1/assets/id/file',
    );
  });

  test(
    'resolved generated route previews retain SVG rendering with thumbnails',
    () {
      final url = record.getFileUrl(
        server,
        '/api/v1/files/assets/id/preview.svg?share=reader',
        thumb: '600x0',
      )!;
      expect(
        url,
        'https://a.example/api/v1/files/assets/id/preview.svg?share=reader&thumb=600x0',
      );
      expect(isSvgPhoto(url), isTrue);
    },
  );

  test('keeps absolute photo URLs', () {
    expect(
      record.getFileUrl(server, 'https://b.example/photo.jpg'),
      'https://b.example/photo.jpg',
    );
    expect(
      record.getFileUrl(server, '//b.example/photo.jpg'),
      'https://b.example/photo.jpg',
    );
  });

  test('keeps legacy filenames under the configured server prefix', () {
    expect(
      record.getFileUrl('$server/', 'photo.jpg'),
      '$server/api/v1/files/trails/trail/photo.jpg',
    );
  });

  test('adds a thumbnail while preserving existing access queries', () {
    for (final photo in [
      '/api/v1/files/assets/id/photo.jpg?share=reader&x=one&x=two',
      'https://b.example/photo.jpg?share=reader&x=one&x=two',
      'photo.jpg?share=reader&x=one&x=two',
    ]) {
      final url = Uri.parse(record.getFileUrl(server, photo, thumb: '600x0')!);
      expect(url.queryParametersAll, {
        'share': ['reader'],
        'x': ['one', 'two'],
        'thumb': ['600x0'],
      });
    }
  });

  test('replaces an existing thumbnail without changing the fragment', () {
    final url = Uri.parse(
      record.getFileUrl(
        server,
        '/photo.svg?thumb=100x0&share=reader#route',
        thumb: '600x0',
      )!,
    );
    expect(url.queryParameters['thumb'], '600x0');
    expect(url.queryParameters['share'], 'reader');
    expect(url.fragment, 'route');
  });

  test('does not add an empty thumbnail and ignores empty photos', () {
    expect(record.getFileUrl(server, null), isNull);
    expect(record.getFileUrl(server, ''), isNull);
    expect(
      record.getFileUrl(server, '/photo.svg?share=reader', thumb: ''),
      'https://a.example/photo.svg?share=reader',
    );
  });
}
