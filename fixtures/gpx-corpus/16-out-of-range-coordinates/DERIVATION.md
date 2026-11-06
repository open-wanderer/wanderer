# Derivation: 16-out-of-range-coordinates

Lat 95.5 is measured as 90, lon 190 as -170. The hops are therefore 89/0 to 90/0, 90/0 to 10/179, and 10/179 to 10/-170 across the antimeridian.

Distance is the sum of the haversine hops (`R = 6371000`), evaluated in a scratch Python script; bounding box and centroid are plain arithmetic over the file's coordinates. No expected value came from executing the implementation under test. The values were then checked against the app's reader and metrics (`app/lib/vendor/gpx`, `app/lib/util/gpx/conversion.dart`).

- distance: `10211294.96071456`
- durationMs: `0`
