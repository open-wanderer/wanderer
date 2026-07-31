import 'package:gpx/gpx.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/category_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';

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
  // Moving time in seconds for trails recorded in the Wanderer app. No
  // default and stays nullable: absence is the meaningful "no moving time
  // known" state (D-10, phase 34) -- `duration` always means GPX-derived
  // elapsed time.
  double? movingDuration;
  double? lat;
  double? lon;
  double maxLat;
  double maxLon;
  double minLat;
  double minLon;
  String? gpxData;
  String? navCacheJson;
  String? description;
  @Property(type: PropertyType.dateUtc)
  DateTime created;
  @Property(type: PropertyType.dateUtc)
  DateTime updated;
  List<String> photos = [];

  /// Ids of the accounts that have this trail in their offline library.
  ///
  /// This is what makes the library per-account WITHOUT deleting anything on
  /// logout: one row and one copy of the files per trail, shared by however
  /// many accounts downloaded it, with every read path filtering on the
  /// signed-in account via `savedByUserIds.containsElement(userId)`.
  ///
  /// A per-(trail, account) row was not an option: [id] is
  /// `@Unique(onConflict: replace)`, so a second account downloading the same
  /// trail would silently replace the first account's row.
  ///
  /// Empty means no account holds it — `TrailLibraryNotifier.deleteTrail`
  /// treats that as the signal to finally remove the row and its
  /// `library/<id>/` directory.
  List<String> savedByUserIds = [];

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
    required this.created,
    required this.updated,
    this.location,
    this.date,
    this.public = false,
    this.distance = 0,
    this.elevationGain = 0,
    this.elevationLoss = 0,
    this.duration = 0,
    this.movingDuration,
    this.difficulty = TrailDifficulty.easy,
    this.lat,
    this.lon,
    this.maxLat = 0,
    this.maxLon = 0,
    this.minLat = 0,
    this.minLon = 0,
    this.gpxData,
    this.navCacheJson,
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
      movingDuration: trail.movingDuration,
      lat: trail.lat,
      lon: trail.lon,
      maxLat: trail.maxLat,
      maxLon: trail.maxLon,
      minLat: trail.minLat,
      minLon: trail.minLon,
      gpxData: trail.expand?.gpxData,
      description: trail.description,
      updated: trail.updated,
      created: trail.created,
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
      movingDuration: movingDuration,
      difficulty: difficulty,
      lat: lat,
      lon: lon,
      maxLat: maxLat,
      maxLon: maxLon,
      minLat: minLat,
      minLon: minLon,
      description: description ?? "",
      isOffline: true,
      localPhotos: photos,
      updated: updated,
      created: created,
      expand: TrailExpand(
        author: author.target?.toModel(),
        category: category.target?.toModel(),
        gpxData: gpxData,
        gpx: gpxData != null
            ? GpxReader().fromString(sanitizeGpxEmail(gpxData!))
            : null,
        waypointsViaTrail: waypoints.map((w) => w.toModel()).toList(),
      ),
    );
  }
}
