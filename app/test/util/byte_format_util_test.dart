import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/byte_format_util.dart';

// ---------------------------------------------------------------------------
// Tests for byte_format_util's pure formatBytes function. Convention per
// 24-UI-SPEC.md: one decimal place, unit steps at KB/MB/GB (e.g. "45 MB",
// "2.4 GB"); bare "N B" below 1 KB.
// ---------------------------------------------------------------------------

void main() {
  group('formatBytes', () {
    test('0 bytes formats as "0 B"', () {
      expect(formatBytes(0), '0 B');
    });

    test('512 bytes formats as "512 B"', () {
      expect(formatBytes(512), '512 B');
    });

    test('1024 bytes formats as "1.0 KB"', () {
      expect(formatBytes(1024), '1.0 KB');
    });

    test('1536 bytes formats as "1.5 KB"', () {
      expect(formatBytes(1536), '1.5 KB');
    });

    test('1 MB formats as "1.0 MB"', () {
      expect(formatBytes(1024 * 1024), '1.0 MB');
    });

    test('2.4 GB formats as "2.4 GB"', () {
      expect(formatBytes((2.4 * 1024 * 1024 * 1024).round()), '2.4 GB');
    });
  });
}
