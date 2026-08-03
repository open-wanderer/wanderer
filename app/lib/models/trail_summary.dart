import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_sync_state.dart';

abstract class TrailSummary with RecordFunctions {
  String get name;

  String? get domain;
  String? get location;
  double get distance;
  double get duration;
  // D-10 display rule support: only `Trail` has a real moving-time value (a
  // separate, session-derived field). Every other TrailSummary implementer
  // (e.g. search-result summaries) has no such concept and must override
  // this with `null`, which `trailDisplayDuration` treats as "show
  // duration". `implements` does not inherit a default body, so every
  // implementer declares its own override explicitly.
  double? get movingDuration;
  double get elevationGain;
  double get elevationLoss;
  bool get public;

  DateTime? get summaryDate;
  String get summaryThumbnail;
  int get summaryDifficulty;
  String get summaryAuthorName;
  String get summaryAuthorAvatar;

  /// The author's actor record id, or null when this summary has no author
  /// resolved. `ActorAvatar` uses it to recognise the signed-in user and
  /// serve their avatar from the on-disk cache — the only branch that works
  /// offline, and therefore the one a just-recorded trail depends on.
  String? get summaryAuthorActorId;
  String? get categoryId;
  String? get subcategoryId;

  List<String>? get summaryShares;
  List<String>? get summaryTags;

  bool get isLocal;
  List<String> get localPhotos;

  // `implements` does not inherit a default body, so every implementer
  // declares its own override explicitly.
  TrailSyncState get syncState;
  String? get localId;
}
