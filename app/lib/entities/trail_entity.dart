import 'dart:convert';

import 'package:flutter/foundation.dart' show debugPrint;
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/category_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/local/id.dart';

/// `Trail.author`'s `@Default` — a placeholder, not a real actor record id.
/// It must never be persisted as [TrailEntity.authorRecordId] or compared
/// against a signed-in user's `actorId`.
const _unknownAuthorId = '000000000000000';

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
  // known" state -- `duration` always means GPX-derived
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

  /// The `Trail.category` RECORD ID, persisted as a plain scalar.
  ///
  /// Named `categoryRecordId` and not `categoryId` because ObjectBox already
  /// claims the `categoryId` column for the [category] `ToOne` relation's
  /// foreign key -- a Dart field of that name would collide with the
  /// generated binding.
  ///
  /// The relation is NOT a substitute for this column: [TrailEntity.fromModel]
  /// can only populate it from `trail.expand?.category`, which is null on a
  /// locally-composed draft (the create form yields an id, never an expanded
  /// `Category`). Without this scalar the user's category selection was
  /// dropped by every local-first save and by the upload that followed it.
  String? categoryRecordId;

  /// The `Trail.subcategory` record id. Named for symmetry with
  /// [categoryRecordId]; there is no subcategory relation to collide with.
  String? subcategoryRecordId;

  /// The author's `ActorEntity.id` (the actor RECORD id), persisted as a
  /// plain scalar for the same reason as [categoryRecordId] — and for one
  /// more.
  ///
  /// [ActorEntity.id] is `@Unique(onConflict: replace)`, so putting a freshly
  /// constructed `ActorEntity` (`obxId == 0`) DELETES the old row and inserts
  /// a new one under a new ObjectBox id. The [author] ToOne stores its target
  /// by that ObjectBox id, so it silently resolved to null afterwards — the
  /// trail's author line degraded to "Unknown", the avatar to a bare grey
  /// circle, and `readOwnLocalTrails`' author clause stopped matching.
  ///
  /// Every writer now routes the actor through [actorEntityForUpsert], which
  /// reuses the existing row's ObjectBox id so a put UPDATES rather than
  /// re-mints, and the ToOne no longer goes stale. This scalar remains the
  /// durable record of authorship regardless: it is the only thing carrying
  /// the author id on a locally-composed draft, which has no expanded `Actor`
  /// at all. Rows written by the old path keep a dangling relation — nothing
  /// repairs them, so a device holding pre-fix data needs its app data
  /// cleared (or those trails re-downloaded) to pick the author back up.
  String? authorRecordId;

  /// The `Trail.completed` flag. A plain field with an initializer, like
  /// [photos] -- not a constructor parameter.
  bool completed = false;

  /// The trail's tags, serialized as a JSON array of `Tag.toJson()` maps.
  ///
  /// JSON rather than a `List<String>` of names so a tag that ALREADY has a
  /// server id round-trips with it: `TrailSave.resolveTags` reuses a tag whose
  /// id is non-empty and only `PUT /tag`s the ones without one. Persisting
  /// names alone would turn every already-known tag into a needless create on
  /// each upload. Parallel id/name columns were rejected -- they can drift out
  /// of step with each other; one JSON column cannot. Precedent: [navCacheJson]
  /// and `CategoryEntity.translationsJson`.
  String? tagsJson;

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
  /// This is NOT [savedByUserIds] and must never be
  /// conflated with it — `savedByUserIds` tracks who has this trail
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

  /// The Trail half: app-owned copies of picked photos for a trail that
  /// has not uploaded yet. Declared as a plain field with an initializer,
  /// exactly like [photos] (not a constructor parameter).
  List<String> localPhotos = [];

  /// Backoff bookkeeping: how many upload attempts have failed so far.
  /// Persisted so a parked failure survives an app restart.
  int syncAttempts = 0;

  /// Backoff bookkeeping: the earliest time the drain should retry this
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
  //
  /// [store] MUST be passed by every caller that intends to persist the
  /// result. It is what lets the [author] relation reuse the ObjectBox id of
  /// the `ActorEntity` row that already holds this actor (via
  /// [actorEntityForUpsert]) instead of minting a fresh `obxId == 0` entity.
  /// Because `ActorEntity.id` is `@Unique(onConflict: replace)`, putting a
  /// fresh one deletes and re-inserts the actor row — orphaning the `author`
  /// ToOne of every OTHER trail by that same author, which is how library
  /// cards ended up showing "Unknown"/"UN" for trails that had been stored
  /// correctly. It is optional only so pure model round-trip tests, which
  /// never persist, can call this without an open store.
  factory TrailEntity.fromModel(Trail trail, {Store? store}) {
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
    entity.categoryRecordId = trail.category;
    entity.subcategoryRecordId = trail.subcategory;
    entity.completed = trail.completed;
    entity.tagsJson = encodeTrailTags(trail.expand?.tags);

    if (trail.expand?.waypointsViaTrail != null) {
      final waypointEntities = trail.expand!.waypointsViaTrail!
          .map((w) => WaypointEntity.fromModel(w))
          .toList();

      entity.waypoints.addAll(waypointEntities);
    }

    // The scalar is written from `trail.author` (always present) rather than
    // from the expand, which is null on a locally-composed recording. The
    // sentinel `Trail.author` default is not a real actor id and is skipped
    // so it can never be mistaken for one.
    if (trail.author.isNotEmpty && trail.author != _unknownAuthorId) {
      entity.authorRecordId = trail.author;
    }

    if (trail.expand?.author != null) {
      final actor = trail.expand!.author!;
      entity.author.target = store != null
          ? actorEntityForUpsert(store, actor)
          : ActorEntity.fromModel(actor);
      entity.authorRecordId ??= actor.id;
    }

    if (trail.expand?.category != null) {
      final category = trail.expand!.category!;
      entity.category.target = store != null
          ? categoryEntityForUpsert(store, category)
          : CategoryEntity.fromModel(category);
    }

    return entity;
  }
}

/// Serializes [tags] for [TrailEntity.tagsJson]. Null for an absent or empty
/// tag list so an untagged trail stores nothing rather than the string `[]`.
String? encodeTrailTags(List<Tag>? tags) {
  if (tags == null || tags.isEmpty) return null;
  return jsonEncode(tags.map((t) => t.toJson()).toList());
}

/// Reads [TrailEntity.tagsJson] back into models.
///
/// Degrades to an empty list on malformed JSON rather than throwing, unlike
/// the `gpxData` parse right below it in [TrailEntityMapping.toModel]. The two
/// differ on purpose: an unparseable GPX means the trail's defining payload is
/// gone and there is nothing useful left to show, whereas unparseable tags are
/// a cosmetic subset of a trail that is otherwise entirely intact -- throwing
/// there would hide the whole row behind a `toModel()` failure.
List<Tag> decodeTrailTags(String? tagsJson) {
  if (tagsJson == null || tagsJson.isEmpty) return const [];
  try {
    final decoded = jsonDecode(tagsJson) as List<dynamic>;
    return decoded.map((e) => Tag.fromJson(e as Map<String, dynamic>)).toList();
  } catch (e, st) {
    debugPrint('trail_entity: could not decode tagsJson "$tagsJson": $e\n$st');
    return const [];
  }
}

extension TrailEntityMapping on TrailEntity {
  /// [includeGpx] gates the expensive part: parsing `gpxData` into a full
  /// `Gpx` object graph (and carrying the raw XML string alongside it).
  /// List surfaces (library, own-trails) render only the scalar columns
  /// stored on this row, yet used to pay a synchronous UI-thread XML parse
  /// per trail — and then hold every parsed trackpoint plus the raw XML in
  /// memory for as long as the list was alive. Pass `false` for list rows;
  /// detail/save/upload paths keep the default and read a fresh full model.
  Trail toModel({bool includeGpx = true}) {
    final tags = decodeTrailTags(tagsJson);

    return Trail(
      // A local-sentinel id is blanked here so an empty id means
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
      // Falls back to the relation's id so a downloaded row written before
      // `categoryRecordId` existed still reports its category.
      category: categoryRecordId ?? category.target?.id,
      subcategory: subcategoryRecordId,
      completed: completed,
      // Ids only, for `toFormData`'s `tags` field. Tags still awaiting their
      // first `PUT /tag` have no id yet and are carried by `expand.tags`
      // below, which is what `resolveTags` reads.
      tags: tags
          .map((t) => t.id)
          .whereType<String>()
          .where((id) => id.isNotEmpty)
          .toList(),
      // Falls back to the relation's id, same shape as `category` above, so a
      // row written before `authorRecordId` existed still reports its author.
      // Left at the `Trail.author` default when neither is available.
      author: authorRecordId ?? author.target?.id ?? _unknownAuthorId,
      isLocal: true,
      // A row's own `localPhotos` win when present -- an overlap row's
      // capture copies are the ones the drain still needs. A plain
      // downloaded row has an empty `localPhotos`, so the fallback to
      // `photos` keeps every downloaded-trail reader working. This
      // expression's correctness never depended on unsynced-ness and
      // download membership excluding one another.
      localPhotos: localPhotos.isNotEmpty ? localPhotos : photos,
      localId: localId,
      syncState: syncState,
      updated: updated,
      created: created,
      expand: TrailExpand(
        tags: tags,
        author: author.target?.toModel(),
        category: category.target?.toModel(),
        gpxData: includeGpx ? gpxData : null,
        // parseGpxSafely, not a bare GpxReader: this is third-party GPX read
        // back out of the offline cache and needs the full sanitize chain.
        // toModel() is called outside any try/catch, so a FormatException from
        // an unsanitized tag escaped the notifier entirely and made the trail
        // permanently un-openable offline once cached.
        gpx: includeGpx && gpxData != null ? parseGpxSafely(gpxData!) : null,
        waypointsViaTrail: waypoints.map((w) => w.toModel()).toList(),
      ),
    );
  }
}
