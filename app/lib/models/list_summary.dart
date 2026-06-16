import 'package:wanderer/models/record.dart';

abstract class ListSummary with RecordFunctions {
  String get name;
  String? get avatar;
  int get trailCount;

  bool get public;

  String? get description;

  String get summaryAuthorName;
  String get summaryAuthorAvatar;

  double? get elevationGain;
  double? get elevationLoss;
  double? get distance;
  double? get duration;
}
