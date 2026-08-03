import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_summary.dart';

abstract class ListSummary with RecordFunctions {
  String get name;
  String? get avatar;
  int get trailCount;

  bool get public;

  String? get description;

  String get summaryAuthorName;
  String get summaryAuthorAvatar;

  /// The author's actor record id, or null when unresolved. See
  /// [TrailSummary.summaryAuthorActorId] — `ActorAvatar` uses it to serve the
  /// signed-in user's avatar from the on-disk cache.
  String? get summaryAuthorActorId;

  double? get elevationGain;
  double? get elevationLoss;
  double? get distance;
  double? get duration;
}
