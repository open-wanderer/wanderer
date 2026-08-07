import 'dart:async';
import 'dart:io';

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/util/trail/form_data.dart';
import 'package:wanderer/util/object_diff.dart';

part 'trail_save_provider.g.dart';

class TrailSaveResult {
  final Trail trail;

  /// Whether `trail.expand.waypointsViaTrail` may be INCOMPLETE.
  ///
  /// Read by callers as "this waypoint set is not authoritative" -- it is
  /// wired straight to `applyServerTrailToLibraryRow`'s
  /// `waypointsAreAuthoritative:` (negated) in `trail_create_screen.dart`,
  /// where a false negative makes the store prune a still-live waypoint out
  /// of the offline copy.
  ///
  /// So EVERY path that omits a waypoint from the returned set must set
  /// this, not just the ones that caught an exception. The name
  /// says "failures" for history; the property callers depend on is
  /// completeness.
  final bool hadWaypointFailures;

  const TrailSaveResult({
    required this.trail,
    this.hadWaypointFailures = false,
  });
}

@Riverpod(keepAlive: true)
class TrailSave extends _$TrailSave {
  @override
  FutureOr<void> build() {}

  /// Resolves any client-created (unsaved, `id == null`) [tags] into real
  /// PocketBase records, returning the full tag list (existing + newly
  /// created) with real ids.
  ///
  /// Public so the deferred-upload drain (`trail_sync_provider.dart`) can
  /// reuse the exact same tag-reuse-vs-create rule it applies at
  /// interactive save time.
  Future<List<Tag>> resolveTags(List<Tag> tags) async {
    final resolved = <Tag>[];
    for (final tag in tags) {
      if (tag.id != null && tag.id!.isNotEmpty) {
        resolved.add(tag);
      } else {
        resolved.add(await ref.read(tagProvider.notifier).create(tag.name));
      }
    }
    return resolved;
  }

  Future<TrailSaveResult> createTrail(
    Trail trail, {
    required String authorId,
    required List<File> newPhotos,
  }) async {
    final resolvedTags = await resolveTags(trail.expand?.tags ?? const []);

    final trailToSend = trail.copyWith(
      author: authorId,
      tags: resolvedTags.map((t) => t.id!).toList(),
    );

    final api = ref.read(apiProvider);
    final formData = await trailToSend.toFormData(
      newPhotos: newPhotos,
      isCreate: true,
    );

    final response = await api.put(
      '/trail/form',
      data: formData,
      queryParameters: {'expand': 'category,tags,author'},
    );

    var model = Trail.fromJson(response.data);

    final waypointNotifier = ref.read(waypointSaveProvider.notifier);
    final createdWaypoints = <Waypoint>[];
    var hadFailures = false;

    for (final waypoint in trail.expand?.waypointsViaTrail ?? const []) {
      try {
        createdWaypoints.add(
          await waypointNotifier.create(
            waypoint,
            authorId: authorId,
            trailId: model.id,
          ),
        );
      } catch (e) {
        hadFailures = true;
      }
    }

    model = model.copyWith(
      expand: (model.expand ?? const TrailExpand()).copyWith(
        waypointsViaTrail: createdWaypoints,
        tags: resolvedTags,
        // Trail.fromJson never populates expand.gpx/gpxData (gpx is
        // includeFromJson: false; gpxData only comes from the server's file
        // endpoint). Carry the just-submitted track forward so TrailMap
        // (which gates the route line and markers on expand.gpx != null)
        // doesn't go blank the instant the caller adopts this result.
        gpx: trailToSend.expand?.gpx,
        gpxData: trailToSend.expand?.gpxData,
      ),
    );

    return TrailSaveResult(trail: model, hadWaypointFailures: hadFailures);
  }

  Future<TrailSaveResult> updateTrail(
    Trail oldTrail,
    Trail newTrail, {
    required String authorId,
    required List<File> newPhotos,
    required List<String> removedPhotoFilenames,
  }) async {
    final resolvedTags = await resolveTags(newTrail.expand?.tags ?? const []);

    final trailToSend = newTrail.copyWith(
      author: oldTrail.author,
      tags: resolvedTags.map((t) => t.id!).toList(),
    );

    final api = ref.read(apiProvider);
    final formData = await trailToSend.toFormData(
      newPhotos: newPhotos,
      removedPhotoFilenames: removedPhotoFilenames,
      isCreate: false,
    );

    final response = await api.post(
      '/trail/form/${newTrail.id}',
      data: formData,
      queryParameters: {'expand': 'category,tags,author'},
    );

    var model = Trail.fromJson(response.data);

    final diff = compareObjectArrays<Waypoint>(
      oldTrail.expand?.waypointsViaTrail ?? const [],
      newTrail.expand?.waypointsViaTrail ?? const [],
      idOf: (w) => w.listKey,
    );

    final waypointNotifier = ref.read(waypointSaveProvider.notifier);
    var hadFailures = false;

    final finalWaypoints = <Waypoint>[
      for (final wp in newTrail.expand?.waypointsViaTrail ?? const [])
        if (!diff.added.contains(wp) && !diff.updated.contains(wp)) wp,
    ];

    for (final wp in diff.added) {
      try {
        finalWaypoints.add(
          await waypointNotifier.create(
            wp,
            authorId: authorId,
            trailId: newTrail.id,
          ),
        );
      } catch (_) {
        hadFailures = true;
      }
    }

    for (final wp in diff.updated) {
      final old = (oldTrail.expand?.waypointsViaTrail ?? const [])
          .where((o) => o.id == wp.id)
          .firstOrNull;
      if (old == null) {
        // Dropping `wp` here without marking makes
        // `finalWaypoints` incomplete while `hadWaypointFailures` stays
        // false, and the caller passes `waypointsAreAuthoritative:
        // !hadWaypointFailures` -- so `applyServerTrailToLibraryRow` prunes
        // a still-live waypoint out of the offline copy. That is exactly
        // the failure `waypointsAreAuthoritative` was added to close.
        // Reachable because `diff` is keyed on `listKey` (`id` when
        // non-empty, else `localKey`) while this re-lookup is on `id`: an
        // old waypoint with id `X` and no localKey against a new one with
        // `localKey == 'X'` and an empty id matches on listKey and misses
        // here. `hadWaypointFailures` means "this waypoint set is not
        // authoritative", not "an exception was thrown".
        hadFailures = true;
        continue;
      }
      try {
        finalWaypoints.add(
          await waypointNotifier.updateWaypoint(old, wp, authorId: authorId),
        );
      } catch (_) {
        hadFailures = true;
      }
    }

    for (final wp in diff.deleted) {
      try {
        await waypointNotifier.delete(wp);
      } catch (_) {
        hadFailures = true;
      }
    }

    model = model.copyWith(
      expand: (model.expand ?? const TrailExpand()).copyWith(
        waypointsViaTrail: finalWaypoints,
        tags: resolvedTags,
        gpx: trailToSend.expand?.gpx,
        gpxData: trailToSend.expand?.gpxData,
      ),
    );

    return TrailSaveResult(trail: model, hadWaypointFailures: hadFailures);
  }

  Future<void> deleteTrail(Trail trail) async {
    // Trail.id arrives from the server (or is echoed back through a
    // model built from a server response) and is validated before it
    // reaches the Dio path, matching trail_sync_provider.dart's deleteUnsynced.
    await ref
        .read(apiProvider)
        .delete('/trail/${recordIdDirSegment(trail.id)}');
  }
}
