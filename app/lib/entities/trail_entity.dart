import 'package:gpx/gpx.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/category_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';

@Entity()
class TrailEntity {
  @Id()
  int obxId = 0;

  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;

  String name;
  String? location;
  @Property(type: PropertyType.dateUtc)
  DateTime? date;
  bool public;
  double? distance;
  double? elevationGain;
  double? elevationLoss;
  double? duration;
  double? lat;
  double? lon;
  String? gpxData;
  String? description;
  List<String> photos = [];
  List<String> pmTiles = [];

  @Transient()
  TrailDifficulty difficulty = TrailDifficulty.easy;

  int get dbDifficulty {
    return difficulty.index;
  }

  set dbDifficulty(int value) {
    if (value >= 0 && value < TrailDifficulty.values.length) {
      difficulty = TrailDifficulty.values[value];
    } else {
      difficulty = TrailDifficulty.easy;
    }
  }

  @Backlink('trail')
  final waypoints = ToMany<WaypointEntity>();

  final author = ToOne<ActorEntity>();
  final category = ToOne<CategoryEntity>();

  TrailEntity({
    required this.id,
    required this.name,
    this.location,
    this.date,
    this.public = false,
    this.distance = 0,
    this.elevationGain = 0,
    this.elevationLoss = 0,
    this.duration = 0,
    this.difficulty = TrailDifficulty.easy,
    this.lat,
    this.lon,
    this.gpxData,
    this.description = "",
  });

  factory TrailEntity.fromModel(Trail trail) {
    final entity = TrailEntity(
      id: trail.id,
      name: trail.name,
      location: trail.location,
      date: trail.date,
      public: trail.public,
      distance: trail.distance,
      elevationGain: trail.elevationGain,
      elevationLoss: trail.elevationLoss,
      duration: trail.duration,
      lat: trail.lat,
      lon: trail.lon,
      gpxData: trail.expand?.gpxData,
      description: trail.description,
    );

    entity.dbDifficulty = trail.difficulty.index;

    if (trail.expand?.waypointsViaTrail != null) {
      final waypointEntities = trail.expand!.waypointsViaTrail!
          .map((w) => WaypointEntity.fromModel(w))
          .toList();

      entity.waypoints.addAll(waypointEntities);
    }

    if (trail.expand?.author != null) {
      entity.author.target = ActorEntity.fromModel(trail.expand!.author!);
    }

    if (trail.expand?.category != null) {
      entity.category.target = CategoryEntity.fromModel(
        trail.expand!.category!,
      );
    }

    return entity;
  }
}

extension TrailEntityMapping on TrailEntity {
  Trail toModel() {
    return Trail(
      id: id,
      name: name,
      location: location,
      date: date,
      public: public,
      distance: distance ?? 0,
      elevationGain: elevationGain ?? 0,
      elevationLoss: elevationLoss ?? 0,
      duration: duration ?? 0,
      difficulty: difficulty,
      lat: lat,
      lon: lon,
      description: description ?? "",
      isOffline: true,
      localPhotos: photos,
      pmTiles: pmTiles,
      expand: TrailExpand(
        author: author.target?.toModel(),
        category: category.target?.toModel(),
        gpxData: gpxData,
        gpx: gpxData != null ? GpxReader().fromString(gpxData!) : null,
        waypointsViaTrail: waypoints.map((w) => w.toModel()).toList(),
      ),
    );
  }
}
