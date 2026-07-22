/// Human-readable byte formatting for the region tile repository's disk
/// usage displays (SETUI-05).
///
/// Convention per `24-UI-SPEC.md`: one decimal place, unit steps at
/// KB/MB/GB (e.g. "45 MB", "2.4 GB"); a bare `'$bytes B'` below 1 KB (no
/// decimal -- a byte count is already an exact integer).
library;

const int _kb = 1024;
const int _mb = _kb * 1024;
const int _gb = _mb * 1024;

/// Formats [bytes] as a human-readable string using 1024-based unit steps.
String formatBytes(int bytes) {
  if (bytes >= _gb) return '${(bytes / _gb).toStringAsFixed(1)} GB';
  if (bytes >= _mb) return '${(bytes / _mb).toStringAsFixed(1)} MB';
  if (bytes >= _kb) return '${(bytes / _kb).toStringAsFixed(1)} KB';
  return '$bytes B';
}
