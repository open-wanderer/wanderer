# Derivation: 14-positionless-ends

The first and last track point have no position: they add no distance and no point, but the duration runs from the first time to the last, 31 s. The trail date needs both.

Distance is the sum of the haversine hops (`R = 6371000`), evaluated in a scratch Python script; bounding box and centroid are plain arithmetic over the file's coordinates. No expected value came from executing the implementation under test. The values were then checked against the app's reader and metrics (`app/lib/vendor/gpx`, `app/lib/util/gpx/conversion.dart`).

- distance: `134.5924014859008`
- durationMs: `31000`
