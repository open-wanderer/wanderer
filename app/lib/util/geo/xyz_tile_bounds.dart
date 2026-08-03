/// Pure Web-Mercator ("slippy map") tile z/x/y &lt;-&gt; lon/lat bbox math for the
/// local tile proxy (PROXY-01/02).
///
/// This is standard, widely-documented tile math (OSM wiki "Slippy map
/// tilenames") — no project-specific logic, no I/O, no dependency beyond
/// `dart:math`. Callers (the proxy request handler in `tile_proxy_server.dart`)
/// MUST bounds-check `z`/`x`/`y` themselves BEFORE calling [tileToBounds]: this
/// function performs no validation of its own, mirroring `util/region/file_path.dart`'s
/// "validate before use" convention even though the validation itself lives in
/// the caller here, not this pure-math module.
library;

import 'dart:math' show pi, atan, exp;

/// Returns the lon/lat bounding box for standard XYZ tile [z]/[x]/[y].
///
/// [z] is the zoom level; [x]/[y] are the tile column/row at that zoom.
/// Callers must ensure `0 <= x < 2^z` and `0 <= y < 2^z` before calling —
/// out-of-range input produces mathematically well-defined but meaningless
/// (out-of-world) coordinates rather than throwing.
({double west, double south, double east, double north}) tileToBounds(
  int z,
  int x,
  int y,
) {
  double lon(int tx) => tx / (1 << z) * 360.0 - 180.0;
  double lat(int ty) {
    final n = pi - 2.0 * pi * ty / (1 << z);
    return 180.0 / pi * atan(0.5 * (exp(n) - exp(-n)));
  }

  return (west: lon(x), east: lon(x + 1), north: lat(y), south: lat(y + 1));
}
