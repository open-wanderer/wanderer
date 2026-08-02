import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/category_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';
import 'package:wanderer/util/local_id.dart';

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

  /// The `UserEntity.id` of the account that CAPTURED this trail on this
  /// device.
  ///
  /// Singular, not a list: authorship is 1:1, while downloading is 1:N
  /// (RESEARCH.md A3). This is NOT [savedByUserIds] and must never be
  /// conflated with it (D-10) — `savedByUserIds` tracks who has this trail
  /// in their offline library, `owner` tracks who recorded it on this
  /// device. A null owner means "not authored on this device" and must
  /// never match an owner filter.
  @Index()
  String? owner;

  /// Permanent local identity minted once at first local save.
  ///
  /// Never changed, including after the trail gains a server id — it
  /// outlives the promotion to synced because the drain still needs it to
  /// find the trail's photo directory and its in-flight-set key.
  String? localId;

  /// D-04's Trail half: app-owned copies of picked photos for a trail that
  /// has not uploaded yet. Declared as a plain field with an initializer,
  /// exactly like [photos] (not a constructor parameter).
  List<String> localPhotos = [];

  /// D-07 backoff bookkeeping: how many upload attempts have failed so far.
  /// Persisted so a parked failure survives an app restart.
  int syncAttempts = 0;

  /// D-07 backoff bookkeeping: the earliest time the drain should retry this
  /// trail's upload. Persisted so a parked failure survives an app restart.
  @Property(type: PropertyType.dateUtc)
  DateTime? syncNextAttemptAt;

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

  @Transient()
  TrailSyncState syncState = TrailSyncState.synced;

  int get dbSyncState {
    return syncState.index;
  }

  set dbSyncState(int value) {
    if (value >= 0 && value < TrailSyncState.values.length) {
      syncState = TrailSyncState.values[value];
    } else {
      syncState = TrailSyncState.synced;
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
    this.owner,
    this.localId,
    this.syncAttempts = 0,
    this.syncNextAttemptAt,
    this.syncState = TrailSyncState.synced,
  });

  // `fromModel` is the shared conversion used by the download path. It does
  // NOT set `owner` or `localPhotos` — every writer that owns those two
  // fields (a local capture, an upload retry) sets them explicitly inside
  // its own transaction. Do not "helpfully" add them here.
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
      localId: trail.localId,
      syncState: trail.syncState,
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
      // D-06: a local-sentinel id is blanked here so an empty id means
      // "not-yet-uploaded" at the model layer, even though ObjectBox needs
      // a unique non-empty value to store the row.
      id: isLocalId(id) ? '' : id,
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
      isLocal: true,
      // D-10: mutually exclusive by construction — a downloaded row carries
      // its local file copies in `photos`, an unsynced row carries them in
      // `localPhotos`. Falling back to `photos` keeps every existing
      // downloaded-trail reader working unchanged.
      localPhotos: localPhotos.isNotEmpty ? localPhotos : photos,
      localId: localId,
      syncState: syncState,
      updated: updated,
      created: created,
      expand: TrailExpand(
        author: author.target?.toModel(),
        category: category.target?.toModel(),
        gpxData: gpxData,
        // parseGpxSafely, not a bare GpxReader: this is third-party GPX read
        // back out of the offline cache and needs the full sanitize chain.
        // toModel() is called outside any try/catch, so a FormatException from
        // an unsanitized tag escaped the notifier entirely and made the trail
        // permanently un-openable offline once cached.
        gpx: gpxData != null ? parseGpxSafely(gpxData!) : null,
        waypointsViaTrail: waypoints.map((w) => w.toModel()).toList(),
      ),
    );
  }
}
