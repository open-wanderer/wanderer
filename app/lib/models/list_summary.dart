import 'package:wanderer/models/record.dart';

abstract class ListSummary with RecordFunctions {
  String get name;
  String? get avatar;
  int get trailCount;

  double? get elevationGain;
  double? get elevationLoss;
  double? get distance;
  double? get duration;
}
