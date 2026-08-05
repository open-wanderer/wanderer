/// Pure math shared by every screen using the map's draggable trail sheet
/// (`map_screen.dart`, `profile_trail_map_screen.dart`). Both take the
/// screen's own `sheetMinSize`/`sheetMediumsize`/`sheetMaxSize` as explicit
/// parameters rather than closing over instance fields, since the profile
/// screen's `sheetMinSize` differs (no bottom nav to account for).
library;

/// Fades the sheet's drag handle and trail-count header out as the sheet is
/// dragged open past [sheetMediumsize], fully gone by [sheetMaxSize].
double sheetHeaderOpacity(
  double currentSize, {
  required double sheetMediumsize,
}) {
  if (currentSize <= sheetMediumsize) return 1.0;

  final opacity =
      1.0 - ((currentSize - sheetMediumsize) / (1 - sheetMediumsize));
  return opacity.clamp(0.0, 1.0);
}

/// Interpolates the sheet list's bottom padding between [sheetMinSize] (max
/// bottom padding) through [sheetMediumsize] (no padding) up to
/// [sheetMaxSize] (max top padding) — a single continuous ramp used to keep
/// content clear of the drag handle/header at every sheet height.
double dynamicSheetPadding(
  double currentSize, {
  required double sheetMinSize,
  required double sheetMediumsize,
  required double sheetMaxSize,
}) {
  const double minPadding = 0.0;
  const double maxTopPadding = 156.0;
  const double maxBottomPadding = 64.0;

  final double startThreshold = sheetMediumsize;
  final double endTopThreshold = sheetMaxSize;
  final double endBottomThreshold = sheetMinSize;

  if (currentSize >= endTopThreshold) return maxTopPadding;
  if (currentSize <= endBottomThreshold) return maxBottomPadding;

  if (currentSize <= startThreshold) {
    final double percentage =
        (startThreshold - currentSize) / (startThreshold - endBottomThreshold);
    return minPadding + (percentage * (maxBottomPadding - minPadding));
  } else {
    final double percentage =
        (currentSize - startThreshold) / (endTopThreshold - startThreshold);
    return minPadding + (percentage * (maxTopPadding - minPadding));
  }
}
