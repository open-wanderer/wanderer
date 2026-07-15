import 'dart:async';
import 'dart:io';

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/util/form_data_util.dart';
import 'package:wanderer/util/object_diff_util.dart';

part 'trail_save_provider.g.dart';

class TrailSaveResult {
  final Trail trail;
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
  Future<List<Tag>> _resolveTags(List<Tag> tags) async {
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
    final resolvedTags = await _resolveTags(trail.expand?.tags ?? const []);

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
    final resolvedTags = await _resolveTags(newTrail.expand?.tags ?? const []);

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
      idOf: (w) => w.id,
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
      if (old == null) continue;
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
      ),
    );

    return TrailSaveResult(trail: model, hadWaypointFailures: hadFailures);
  }

  Future<void> deleteTrail(Trail trail) async {
    await ref.read(apiProvider).delete('/trail/${trail.id}');
  }
}
