import 'package:wanderer/models/record.dart';

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
  String? get categoryId;
  String? get subcategoryId;

  List<String>? get summaryShares;
  List<String>? get summaryTags;

  bool get isOffline;
  List<String> get localPhotos;
}
