# Derivation: 15-route-anchor

Legs: 10:00:00 to 10:01:40 (100 s) and, from the anchor, 10:05:00 to 10:06:40 (100 s), 200 s in total. Last minus first would be 400 s. Distance runs through the anchor.

Distance is the sum of the haversine hops (`R = 6371000`), evaluated in a scratch Python script; bounding box and centroid are plain arithmetic over the file's coordinates. No expected value came from executing the implementation under test. The values were then checked against the app's reader and metrics (`app/lib/vendor/gpx`, `app/lib/util/gpx/conversion.dart`).

- distance: `333.584779933514`
- durationMs: `200000`
