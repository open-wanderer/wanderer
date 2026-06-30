import 'package:wanderer/models/record.dart';

abstract class TrailSummary with RecordFunctions {
  String get name;

  String? get domain;
  String? get location;
  double get distance;
  double get duration;
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
