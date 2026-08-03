import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/theme/icons.dart';
import 'package:wanderer/util/local_id.dart';

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
  List<String> localPhotos;

  /// Carries list identity for a waypoint that has no server id yet
  /// (D-06 / RESEARCH.md Pitfall 1). Mirrors [id] once a not-yet-uploaded
  /// waypoint gains a real server id and its persisted `id` column changes.
  String? localKey;

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
    this.localPhotos = const [],
    this.localKey,
  });

  factory WaypointEntity.fromModel(Waypoint w) {
    // A waypoint with no server id needs a stable key to avoid colliding
    // under `@Unique(onConflict: replace)`: reuse an existing localKey if
    // one was already minted, otherwise mint a fresh one.
    final key = w.id.isNotEmpty ? w.id : (w.localKey ?? mintLocalId());

    return WaypointEntity(
      id: key,
      name: w.name,
      description: w.description,
      lat: w.lat,
      lon: w.lon,
      distanceFromStart: w.distanceFromStart,
      author: w.author,
      photos: w.photos,
      localPhotos: w.localPhotos,
      icon: fontAwesomeIconsMapReversed[w.icon],
      created: w.created,
      updated: w.updated,
      localKey: w.id.isNotEmpty ? w.localKey : key,
    );
  }
}

extension WaypointEntityMapping on WaypointEntity {
  Waypoint toModel() {
    return Waypoint(
      // D-06: a local-sentinel id is blanked here, same discipline as
      // TrailEntity.toModel — an empty id means not-yet-uploaded.
      id: isLocalId(id) ? '' : id,
      name: name,
      description: description,
      lat: lat,
      lon: lon,
      distanceFromStart: distanceFromStart,
      author: author,
      photos: photos,
      localPhotos: localPhotos,
      icon: fontAwesomeIconsMap[icon] ?? FontAwesomeIcons.circle,
      updated: updated,
      created: created,
      localKey: localKey,
    );
  }
}
