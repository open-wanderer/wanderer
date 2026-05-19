import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/icon_util.dart';

@Entity()
class WaypointEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;
  String? name;
  String? description;
  double lat;
  double lon;
  double? distanceFromStart;
  String author;
  String? icon;

  List<String> photos;

  @Property(type: PropertyType.dateUtc)
  DateTime created;
  @Property(type: PropertyType.dateUtc)
  DateTime updated;

  final trail = ToOne<TrailEntity>();

  WaypointEntity({
    required this.id,
    required this.created,
    required this.updated,
    this.name,
    this.description,
    required this.lat,
    required this.lon,
    this.distanceFromStart,
    required this.author,
    this.icon,
    this.photos = const [],
  });

  factory WaypointEntity.fromModel(Waypoint w) {
    return WaypointEntity(
      id: w.id,
      name: w.name,
      description: w.description,
      lat: w.lat,
      lon: w.lon,
      distanceFromStart: w.distanceFromStart,
      author: w.author,
      photos: w.photos,
      icon: fontAwesomeIconsMapReversed[w.icon],
      created: w.created,
      updated: w.updated,
    );
  }
}

extension WaypointEntityMapping on WaypointEntity {
  Waypoint toModel() {
    return Waypoint(
      id: id,
      name: name,
      description: description,
      lat: lat,
      lon: lon,
      distanceFromStart: distanceFromStart,
      author: author,
      photos: photos,
      icon: fontAwesomeIconsMap[icon] ?? FontAwesomeIcons.circle,
      updated: updated,
      created: created,
    );
  }
}
