# Derivation: 13-exponent-coordinates

Three points whose coordinates are written with exponents: 47/11, 47.1/11 and 47.1/11.1.

Distance is the sum of the haversine hops (`R = 6371000`), evaluated in a scratch Python script; bounding box and centroid are plain arithmetic over the file's coordinates. No expected value came from executing the implementation under test. The values were then checked against the app's reader and metrics (`app/lib/vendor/gpx`, `app/lib/util/gpx/conversion.dart`).

- distance: `18688.76285784502`
- durationMs: `0`
