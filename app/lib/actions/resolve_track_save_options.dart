import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/navigation/track_save_options_sheet.dart';
import 'package:wanderer/provider/online_status_provider.dart';

// Callers identify themselves with this enum, so it travels with the gate.
export 'package:wanderer/components/navigation/track_save_options_sheet.dart'
    show TrackSaveOptionsSource;

/// The single post-capture gate shared by all three capture sources —
/// recording, route planner and file import (D-15).
///
/// Reads the app's connectivity status (see `online_status_provider.dart`).
/// When offline, returns `(false, false)`
/// immediately WITHOUT touching [context] or showing anything — both
/// toggles off, which is behaviourally identical to the user declining both
/// (D-15's "one code path, not two": offline and "declined both" are the
/// same outcome and callers must not distinguish them). When online, shows
/// the bottom sheet defined in `track_save_options_sheet.dart` and returns
/// its result verbatim.
///
/// A `null` return means the user cancelled/dismissed the sheet — the caller
/// must abort the save entirely with no change to the session (matches that
/// sheet's own contract). A `(false, false)` return can mean either
/// "offline" or "the user declined both toggles online" — callers must
/// treat the two identically.
///
/// Both post-capture transforms this gate covers — `recalcHeights` (a
/// `/valhalla/height` call) and `followRoads` (a `/valhalla/trace-route`
/// call) — are network calls, which is why a single connectivity gate is
/// sufficient to cover both rather than needing per-toggle gating.
///
/// [source] identifies the calling flow; the sheet itself owns every
/// presentation difference between the sources (see
/// [TrackSaveOptionsSource]).
Future<(bool recalcHeights, bool followRoads)?> resolveTrackSaveOptions(
  WidgetRef ref,
  BuildContext context,
  TrackSaveOptionsSource source,
) async {
  if (!ref.read(onlineStatusProvider)) {
    return (false, false);
  }
  if (!context.mounted) return null;
  return showTrackSaveOptionsSheet(context, source: source);
}
