import 'dart:async';
import 'dart:io';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:textfield_tags/textfield_tags.dart';
import 'package:wanderer/components/base/trail_map.dart';
import 'package:wanderer/components/base/wanderer_autocomplete.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_date_picker.dart';
import 'package:wanderer/components/base/wanderer_photo_picker.dart';
import 'package:wanderer/components/base/wanderer_rich_text_editor.dart';
import 'package:wanderer/components/base/wanderer_select.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/components/trail/category_picker.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/waypoint_card.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/util/category/preference_sort.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/exif.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/store/local_photo_store.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/geo/reverse_geocode.dart';
import 'package:wanderer/util/route/planner_handoff.dart';
import 'package:wanderer/util/route/valhalla.dart';

class TrailCreateScreen extends ConsumerStatefulWidget {
  final Trail trail;
  const TrailCreateScreen({super.key, required this.trail});

  @override
  ConsumerState<TrailCreateScreen> createState() => _TrailCreateScreenState();
}

class _TrailCreateScreenState extends ConsumerState<TrailCreateScreen> {
  late Trail trail = widget.trail;

  // Pristine snapshot kept separate from `trail`, which is mutated in place —
  // diffing against `trail` itself would never detect changed waypoints.
  late Trail _originalTrail = widget.trail;
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  // Owned here (not by TrailForm) so _onSave can read the edited values back out.
  final _formKey = GlobalKey<FormBuilderState>();

  ml.MapController? _mapController;
  ml.Geographic? _elevationMarkerPosition;

  double sheetMinSize = 0.1;
  final sheetMediumSize = 0.4;
  final sheetMaxSize = 1.0;

  late final ValueNotifier<double> _sheetSize;

  bool _saving = false;
  List<String> _removedServerPhotos = [];

  // Seeded from widget.trail.localId so re-opening an unsynced trail resumes
  // against the same local row instead of minting a new one (REC-05).
  String? _localId;

  // Ensures the default-category assignment below fires only once.
  bool _categoryDefaulted = false;

  // Ensures the default-privacy assignment below fires only once.
  bool _privacyDefaulted = false;

  @override
  void initState() {
    super.initState();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
    _localId = widget.trail.localId;
    _maybeResolveMissingLocation();
  }

  /// A draft trail handed in from the Route Planner or a recorded navigation
  /// track (see `buildDraftTrail`) carries `lat`/`lon` from its first track
  /// point, but no place-name `location` — unlike a plain GPX-file import,
  /// which gets `location` pre-computed server-side. Best-effort, fire-once:
  /// reverse-geocodes the first track point so the location field isn't
  /// blank in the create form.
  void _maybeResolveMissingLocation() {
    if (trail.location?.isNotEmpty ?? false) return;
    final firstPoint = trail.expand?.gpx?.allPoints.firstOrNull;
    if (firstPoint == null) return;
    unawaited(_resolveMissingLocation(firstPoint));
  }

  Future<void> _resolveMissingLocation(ml.Geographic firstPoint) async {
    try {
      final result = await searchLocationReverseStructured(
        ref.read(apiProvider),
        firstPoint.lat,
        firstPoint.lon,
        includeRoad: false,
      );
      if (!mounted || result == null) return;
      setState(() => trail = trail.copyWith(location: result.fullLabel));
      // WandererTextField's initialValue is only read on first build (see
      // flutter_form_builder's FormBuilderField.didUpdateWidget) — the
      // setState above alone would leave the visible field blank, so also
      // push the fetched value into the already-mounted form field directly.
      _formKey.currentState?.fields['location']?.didChange(result.fullLabel);
    } catch (_) {
      // Best-effort — leave location blank on failure, same silent-failure
      // precedent as _resolveAnchorLocation in route_anchor_provider.dart.
    }
  }

  @override
  void dispose() {
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    super.dispose();
  }

  void _onSheetSizeChanged() {
    _sheetSize.value = _sheetController.size;
  }

  void _onWaypointMoved(Waypoint waypoint, ml.Geographic point) {
    _replaceWaypoint(waypoint.copyWith(lat: point.lat, lon: point.lon));
  }

  Future<void> _onEditWaypoint(BuildContext context, Waypoint waypoint) async {
    final updated = await context.push<Waypoint>(
      '/waypoint/create',
      extra: waypoint,
    );
    if (updated == null) return;
    _replaceWaypoint(updated);
  }

  Future<void> _onCreateWaypoint(
    BuildContext context, {
    ml.Geographic? at,
  }) async {
    // Falls back to the map's current center when not called from a tap.
    final point = at ?? _mapController?.camera?.center;
    final stub = Waypoint(
      // Empty id + a minted local key, not a synthetic id: an id derived
      // from a timestamp would look already-uploaded to the drain's
      // `id.isEmpty` resume check and silently never reach the server
      // (RESEARCH.md Pitfall 1).
      id: '',
      localKey: mintLocalId(),
      lat: point?.lat ?? trail.lat ?? 0,
      lon: point?.lon ?? trail.lon ?? 0,
      created: DateTime.now(),
      updated: DateTime.now(),
    );

    final waypoint = await context.push<Waypoint>(
      '/waypoint/create',
      extra: stub,
    );
    if (waypoint == null) return;
    _appendWaypoint(waypoint);
  }

  /// Creates one waypoint per picked photo that has GPS EXIF data; photos
  /// without GPS are skipped and reported.
  Future<void> _onCreateWaypointsFromPhotos() async {
    final picked = await ImagePicker().pickMultiImage();
    if (picked.isEmpty || !mounted) return;

    final now = DateTime.now();
    final created = <Waypoint>[];
    var skipped = 0;

    for (final photo in picked) {
      final coords = await readGpsFromImage(photo.path);
      if (coords == null || coords.lat.isNaN || coords.lon.isNaN) {
        skipped++;
        continue;
      }
      created.add(
        _withDistanceFromStart(
          Waypoint(
            // Empty id + a minted local key -- see the identical comment in
            // `_onCreateWaypoint` (RESEARCH.md Pitfall 1).
            id: '',
            localKey: mintLocalId(),
            lat: coords.lat,
            lon: coords.lon,
            created: now,
            updated: now,
            localPhotos: [photo.path],
          ),
        ),
      );
    }

    if (!mounted) return;

    if (created.isNotEmpty) {
      setState(() {
        trail = trail.copyWith(
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            waypointsViaTrail: [
              ...?trail.expand?.waypointsViaTrail,
              ...created,
            ],
          ),
        );
      });
    }

    if (skipped > 0) {
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: created.isEmpty ? ToastType.error : ToastType.warning,
              icon: FontAwesomeIcons.circleExclamation,
              text: AppLocalizations.of(
                context,
              )!.photos_skipped_no_gps(skipped),
            ),
          );
    }
  }

  void _onDeleteWaypoint(Waypoint waypoint) {
    setState(() {
      trail = trail.copyWith(
        expand: (trail.expand ?? const TrailExpand()).copyWith(
          waypointsViaTrail: [...?trail.expand?.waypointsViaTrail]
            ..removeWhere((wp) => wp.listKey == waypoint.listKey),
        ),
      );
    });
  }

  /// Estimates [waypoint]'s `distanceFromStart` from the GPX track so it
  /// shows on the elevation profile immediately, ahead of the server value.
  Waypoint _withDistanceFromStart(Waypoint waypoint) {
    final gpx = trail.expand?.gpx;
    if (gpx == null) return waypoint;
    final distance = gpx.distanceFromStartTo(
      ml.Geographic(lat: waypoint.lat, lon: waypoint.lon),
    );
    if (distance == null) return waypoint;
    return waypoint.copyWith(distanceFromStart: distance);
  }

  void _appendWaypoint(Waypoint waypoint) {
    setState(() {
      trail = trail.copyWith(
        expand: (trail.expand ?? const TrailExpand()).copyWith(
          waypointsViaTrail: [
            ...?trail.expand?.waypointsViaTrail,
            _withDistanceFromStart(waypoint),
          ],
        ),
      );
    });
  }

  void _replaceWaypoint(Waypoint updated) {
    final withDistance = _withDistanceFromStart(updated);
    setState(() {
      trail = trail.copyWith(
        expand: (trail.expand ?? const TrailExpand()).copyWith(
          waypointsViaTrail: [
            for (final wp in trail.expand?.waypointsViaTrail ?? const [])
              if (wp.listKey == withDistance.listKey) withDistance else wp,
          ],
        ),
      );
    });
  }

  /// Opens the Route Planner seeded from this trail's existing track and
  /// awaits the edited route back. `null` means the user backed out without
  /// saving. A non-null result is merged onto `trail` via
  /// [mergeRouteIntoTrail], preserving every other field.
  ///
  /// `travelProfile` is resolved from the LIVE form category/subcategory
  /// selection (not `trail.expand?.category`, a relation that is never
  /// populated on an unsaved draft) so re-opening the planner after manually
  /// fixing the category uses the corrected profile rather than reverting to
  /// pedestrian.
  ///
  /// The resolved costing passes through verbatim — an `auto`-mapped ("Car")
  /// trail correctly seeds the planner with `auto` rather than being clamped
  /// to pedestrian/bicycle.
  Future<void> _onEditRoute(
    BuildContext context,
    List<ml.Geographic> points,
  ) async {
    // Read the live field value directly — deliberately no saveAndValidate(),
    // which would surface unrelated validation errors mid-fill.
    final categoryValue =
        _formKey.currentState?.fields['category']?.value as String?;
    final subcategories = ref.read(subcategoryProvider);
    final selection = CategoryPicker.resolve(categoryValue, subcategories);
    // Fall back to the trail's own saved values only when the form field is
    // empty/unset (e.g. before the picker mounts a value).
    final categoryId = selection?.category ?? trail.category;
    final subcategoryId = selection?.subcategory ?? trail.subcategory;
    final category = (ref.read(categoryProvider).value ?? const [])
        .firstWhereOrNull((c) => c.id == categoryId);

    final profile = resolveValhallaProfile(
      category: category,
      subcategoryId: subcategoryId,
      subcategories: subcategories,
    );

    final newGpx = await context.push<Gpx>(
      '/route-planner',
      extra: {
        'mode': 'edit',
        'seedAnchors': points,
        'seedSegmentPolylines': segmentPolylinesFromTrack(
          trail.expand!.gpx!,
          points,
        ),
        'travelProfile': profile?.costing ?? 'pedestrian',
        // Carry the resolved bike variant so an MTB/Gravel/Road trail
        // re-opens on its own bicycle_type instead of Valhalla's Hybrid
        // default.
        if (profile?.bicycleType != null)
          'costingOptions': <String, dynamic>{
            'bicycle_type': profile!.bicycleType,
          },
        'lat': trail.lat,
        'lon': trail.lon,
      },
    );
    if (newGpx == null || !mounted) return;
    // routeAnchorsProvider is keepAlive, so the just-finished planner
    // session's state is still readable here before it gets overwritten.
    final estimatedDurationSeconds = ref
        .read(routeAnchorsProvider)
        .estimatedDurationSeconds;
    setState(
      () => trail = mergeRouteIntoTrail(
        trail,
        newGpx,
        estimatedDurationSeconds: estimatedDurationSeconds,
      ),
    );
  }

  void _onServerPhotosChanged(List<String> remainingFilenames) {
    _removedServerPhotos = trail.photos
        .where((p) => !remainingFilenames.contains(p))
        .toList();
  }

  Future<void> _onSave(BuildContext context) async {
    if (_saving) return;

    final formState = _formKey.currentState;
    if (formState == null || !formState.saveAndValidate()) return;

    final authorId = ref.read(authProvider).value?.actorId;
    if (authorId == null) return;

    setState(() => _saving = true);

    final l10n = AppLocalizations.of(context)!;
    final values = formState.value;
    final subcategories = ref.read(subcategoryProvider);
    final categorySelection = CategoryPicker.resolve(
      values['category'] as String?,
      subcategories,
    );

    final localPhotoPaths = (values['photos'] as List<String>?) ?? const [];
    final newPhotoFiles = localPhotoPaths.map((p) => File(p)).toList();

    final updatedTrail = trail.copyWith(
      name: values['name'] as String,
      location: values['location'] as String?,
      date: values['date'] as DateTime?,
      description: values['description'] as String? ?? '',
      public: values['public'] as bool? ?? false,
      completed: values['completed'] as bool? ?? false,
      difficulty: values['difficulty'] as TrailDifficulty,
      category: categorySelection?.category ?? trail.category,
      subcategory: categorySelection?.subcategory ?? trail.subcategory,
      expand: (trail.expand ?? const TrailExpand()).copyWith(
        tags: values['tags'] as List<Tag>? ?? const [],
      ),
    );

    // Routes on the LOCAL identity of the trail rather than `trail.id.isEmpty`
    // alone: after this phase an empty server id means both "never saved
    // anywhere" AND "saved locally, still unsynced" (RESEARCH.md Pitfall 2).
    //
    // The routing input is the PERSISTED row, not this screen's `trail`
    // snapshot. `trail` is captured once at the end of `_finishLocalSave`,
    // while `syncState` is still `pending`, and the screen does not watch the
    // drain -- so moments later, once the upload finishes, the snapshot is
    // stale. Routing on it sent a post-upload edit down `updateLocal`, which
    // carried the row's `synced` state forward, and `selectDrainCandidates`
    // skips synced rows: the edit stayed on the device forever, under a green
    // "trail saved successfully" toast.
    final store = ref.read(objectBoxProvider);
    final persistedLocalId = _localId;
    final persisted = persistedLocalId == null
        ? null
        : readLocalTrail(store, persistedLocalId);
    // `persisted == null` while `persistedLocalId != null` is now a
    // MEANINGFUL state, not an impossible one: a successful upload
    // retires the row (`retireUploadedLocalTrail`), and nothing else
    // removes a row out from under this screen. Falling back to the
    // snapshot there routes a post-upload edit into `updateLocal`
    // against a row that no longer exists, which returns `missing` and
    // -- before the branch below -- reported success for an edit that
    // was written nowhere.
    final saveMode = resolveLocalSaveModeForRow(
      screenTrail: updatedTrail,
      persistedLocalId: persistedLocalId,
      persisted: persisted,
    );

    if (saveMode == LocalSaveMode.networkUpdate) {
      await _saveViaNetwork(
        l10n,
        updatedTrail,
        authorId: authorId,
        newPhotoFiles: newPhotoFiles,
      );
      return;
    }

    if (saveMode == LocalSaveMode.createLocal) {
      // First save of a captured trail (recording or offline GPX import).
      // Fully local-first: never touches the network, online or offline.
      try {
        // D-13: read fresh here, never a cached value -- a mid-session
        // account switch must not mis-attribute this capture.
        final accountId = currentAccountId(store);
        if (accountId == null) {
          throw StateError(
            'trail_create_screen: no signed-in account for a local save',
          );
        }

        final localId = mintLocalId();

        final photoCopy = await _copyPhotosForLocalSave(
          localId,
          updatedTrail,
          localPhotoPaths,
        );

        saveNewLocalTrail(
          store,
          trail: updatedTrail,
          ownerAccountId: accountId,
          authorActorId: authorId,
          localId: localId,
          trailLocalPhotos: photoCopy.trailPhotos,
          waypointLocalPhotosByKey: photoCopy.waypointPhotosByKey,
        );
        // Only now is the row real. `_copyPhotosForLocalSave` and
        // `saveNewLocalTrail` above can both throw (a filesystem error, an
        // ObjectBox write failure -- the `catch` below exists precisely
        // because they can), and publishing `_localId` before either
        // succeeded left it pointing at an id with no row. The next save
        // then saw `persistedLocalId != null && persisted == null` --
        // indistinguishable from a row a completed upload had retired -- and
        // routed into CR-01's dead POST, bricking the screen for the rest of
        // its life (CR-02).
        _localId = localId;

        await _finishLocalSave(
          store,
          l10n,
          photoCopy.failedCount,
          localId: localId,
        );
      } catch (e) {
        // A connectivity failure can no longer reach this branch on a local
        // save -- only an ObjectBox write or filesystem error lands here.
        if (!mounted) return;
        ref
            .read(toastProvider.notifier)
            .add(
              ToastMessage(
                type: ToastType.error,
                icon: FontAwesomeIcons.circleExclamation,
                text: l10n.error_saving_trail,
              ),
            );
      } finally {
        if (mounted) setState(() => _saving = false);
      }
      return;
    }

    // LocalSaveMode.updateLocal -- a re-save of a still-unsynced trail
    // (REC-05). Fully local-first: never touches the network, online or
    // offline.
    try {
      // Prefer the id the router actually decided on. `_localId` is kept in
      // sync only by `initState` and the `createLocal` branch above, so a
      // future entry point handing in a trail that already carries a
      // `localId` would otherwise die on a bare null-check and surface as the
      // undiagnosable generic `error_saving_trail` toast.
      final localId = persisted?.localId ?? updatedTrail.localId ?? _localId!;

      final photoCopy = await _copyPhotosForLocalSave(
        localId,
        updatedTrail,
        localPhotoPaths,
      );

      final outcome = updateLocalTrail(
        store,
        trail: updatedTrail,
        localId: localId,
        trailLocalPhotos: photoCopy.trailPhotos,
        waypointLocalPhotosByKey: photoCopy.waypointPhotosByKey,
      );

      if (outcome == LocalUpdateOutcome.alreadySynced ||
          outcome == LocalUpdateOutcome.missing ||
          outcome == LocalUpdateOutcome.alreadyUploaded) {
        // The drain finished uploading between the `readLocalTrail` above
        // and this write -- a narrow window, but the one that produces the
        // exact silent-loss the routing change was made to prevent. Since a
        // successful upload now RETIRES the row rather than marking it
        // synced, that discovery arrives as `missing` far more often than
        // as `alreadySynced`; `alreadySynced` survives only for a row this
        // device did not capture. `alreadyUploaded` is the widest window of
        // the three: the drain's create step stamps a real server id well
        // before the row is retired, so a row parked `pending`/`uploading`/
        // `failed` can carry one for as long as later waypoint uploads keep
        // failing (CR-03) -- and `updateLocalTrail` has no update path for
        // it, only `missing`-vs-write. Either way the trail is on the server
        // now, so the network `PUT` is the only write target that can carry
        // this edit anywhere. A `missing` row that was never real routes
        // here too and fails loudly, which is the correct outcome for a
        // bug -- the alternative is a success toast over nothing.
        await _saveViaNetwork(
          l10n,
          updatedTrail,
          authorId: authorId,
          newPhotoFiles: newPhotoFiles,
        );
        return;
      }

      await _finishLocalSave(
        store,
        l10n,
        photoCopy.failedCount,
        localId: localId,
      );
    } catch (e) {
      // A connectivity failure can no longer reach this branch on a local
      // save -- only an ObjectBox write or filesystem error lands here.
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_trail,
            ),
          );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  /// The network `PUT`/`POST` save path for a trail that already lives on the
  /// server, extracted so BOTH routes into it share one implementation: the
  /// ordinary [LocalSaveMode.networkUpdate] decision, and the late discovery
  /// that the drain promoted this trail to `synced` while the save was already
  /// under way ([LocalUpdateOutcome.alreadySynced]).
  ///
  /// Editing a synced server trail offline remains out of scope for this phase
  /// (D-16): with no connection this fails and reports `error_saving_trail`.
  /// That is the point -- the user is told the edit did not land, instead of
  /// being shown a success toast over an edit that no longer has anywhere to
  /// go.
  Future<void> _saveViaNetwork(
    AppLocalizations l10n,
    Trail updatedTrail, {
    required String authorId,
    required List<File> newPhotoFiles,
  }) async {
    if (!trailHasServerId(updatedTrail.id)) {
      // Every route into this method is supposed to guarantee a real server
      // id -- but this screen's own `trail` snapshot can still be the stale
      // one captured right after a local-first save, before the drain ever
      // ran (D-06 blanks a local-sentinel id, so that snapshot's `id` reads
      // `''`). Posting `/trail/form/` with a blank id can never be routed
      // (SvelteKit's `[id]` normalizes an empty segment away, so the create
      // screen would 404/405 on every retry) and would also read photo
      // files a completed retirement already deleted from disk. Refuse
      // rather than issue a request with nowhere to go: the hiker must back
      // out and re-open the trail from the server list to keep editing it
      // (CR-01). The missing-server-id defect this guards against is
      // tracked separately; this only stops it from masquerading as success.
      if (mounted) {
        ref
            .read(toastProvider.notifier)
            .add(
              ToastMessage(
                type: ToastType.error,
                icon: FontAwesomeIcons.circleExclamation,
                text: l10n.error_saving_trail,
              ),
            );
        setState(() => _saving = false);
      }
      return;
    }

    try {
      final result = await ref
          .read(trailSaveProvider.notifier)
          .updateTrail(
            _originalTrail,
            updatedTrail,
            authorId: authorId,
            newPhotos: newPhotoFiles,
            removedPhotoFilenames: _removedServerPhotos,
          );

      if (!mounted) return;
      setState(() {
        trail = result.trail;
        _removedServerPhotos = [];
        _originalTrail = trail;
      });
      _invalidateOwnTrailsList();

      // Every field latches `_dirty` on its first edit and nothing but
      // `reset()` clears it, so without this a saved trail still reports
      // `isDirty` and leaving asks the user to discard changes they already
      // saved. `reset()` is safe here precisely because of the setState above:
      // each field re-seeds from its widget's `initialValue`, which now reads
      // off the just-saved `trail`, so fields land on the saved values rather
      // than the pre-edit ones. That ordering is the whole reason this runs
      // post-frame — called inline it would still see the old widgets and
      // would revert the user's input.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) _formKey.currentState?.reset();
      });

      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: result.hadWaypointFailures
                  ? ToastType.warning
                  : ToastType.success,
              icon: result.hadWaypointFailures
                  ? FontAwesomeIcons.circleExclamation
                  : FontAwesomeIcons.circleCheck,
              text: result.hadWaypointFailures
                  ? l10n.some_waypoints_failed_to_save
                  : l10n.trail_saved_successfully,
            ),
          );
    } catch (e) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_trail,
            ),
          );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  /// Copies picked photos for a local-first save into app-owned storage
  /// under [localId] -- the trail's own photos plus each waypoint that
  /// carries `localPhotos`, keyed on its `localKey` -- and returns the kept
  /// paths plus a total failure count (D-03).
  Future<
    ({
      List<String> trailPhotos,
      Map<String, List<String>> waypointPhotosByKey,
      int failedCount,
    })
  >
  _copyPhotosForLocalSave(
    String localId,
    Trail forTrail,
    List<String> trailPhotoPaths,
  ) async {
    final appDir = await getApplicationDocumentsDirectory();
    var failedCount = 0;

    final trailResult = await reconcileLocalPhotos(
      dir: unsyncedTrailPhotoDir(appDir.path, localId),
      desiredPaths: trailPhotoPaths,
    );
    failedCount += trailResult.failedCount;

    final waypointPhotosByKey = <String, List<String>>{};
    // Typed empty fallback, not a bare `const []`: an untyped one makes `wp`
    // dynamic, which silently turns `failedCount += wp.localPhotos.length`
    // into a `num` and loses static checking over the whole loop body.
    for (final wp in forTrail.expand?.waypointsViaTrail ?? const <Waypoint>[]) {
      final key = wp.localKey;
      if (wp.localPhotos.isEmpty) continue;

      // A keyless waypoint has nowhere to copy its photos TO, so its picked
      // files stay at OS-purgeable image_picker cache paths -- precisely the
      // D-01 failure local_photo_store_util.dart exists to prevent. Skipping
      // silently reported success over photos that can disappear at any
      // moment; counting them makes the user see photo_copy_failed_toast
      // instead.
      if (key == null) {
        failedCount += wp.localPhotos.length;
        continue;
      }

      final waypointResult = await reconcileLocalPhotos(
        dir: unsyncedWaypointPhotoDir(appDir.path, localId, key),
        desiredPaths: wp.localPhotos,
      );
      failedCount += waypointResult.failedCount;
      waypointPhotosByKey[key] = waypointResult.paths;
    }

    return (
      trailPhotos: trailResult.paths,
      waypointPhotosByKey: waypointPhotosByKey,
      failedCount: failedCount,
    );
  }

  /// Notifies every other consumer of the row this screen just wrote.
  ///
  /// The local-first save path commits to ObjectBox, and this app has
  /// no ObjectBox `Query.watch()` streams anywhere, so an explicit
  /// invalidate is the ONLY propagation mechanism that exists. Without
  /// it the own-trails list -- which stays mounted beneath this pushed
  /// edit route (`maintainState: true`), keeping its auto-dispose
  /// provider alive -- holds its pre-edit snapshot until the hiker
  /// pull-to-refreshes, which reads as the edit having been lost.
  ///
  /// This is the same PAIR `trail_sync_provider` invalidates after a
  /// successful drain upload (`:303`, `:377`), spelled identically on
  /// purpose: a local save and an upload must produce the same
  /// refresh, and the family key must match the one
  /// `profile_trail_screen` watches or the invalidation is a silent
  /// no-op. The username is read FRESH here, never from a cached
  /// field (D-13).
  ///
  /// `asReload` defaults to false, i.e. a seamless refresh --
  /// `AsyncValue.when`'s `skipLoadingOnRefresh` defaults to true, so
  /// the list updates without flashing a spinner (36-09).
  void _invalidateOwnTrailsList() {
    ref.invalidate(trailLibraryProvider);
    ref.invalidate(
      profileTrailsProvider(
        '@${ref.read(authProvider).value?.preferredUsername}',
      ),
    );
  }

  /// Shared tail of both local-first [LocalSaveMode] save branches (create
  /// and update): kicks the upload drain, reports any photo-copy failures
  /// (D-03), and mirrors the network path's post-save bookkeeping (re-read,
  /// form reset, success toast).
  Future<void> _finishLocalSave(
    Store store,
    AppLocalizations l10n,
    int failedPhotoCount, {
    required String localId,
  }) async {
    if (mounted) _invalidateOwnTrailsList();

    // Fire-and-forget: online, the trail uploads within a moment of the
    // toast below; offline, this returns immediately after its own
    // connectivity refresh.
    unawaited(ref.read(trailSyncProvider.notifier).drainIfOnline());

    if (failedPhotoCount > 0) {
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.triangleExclamation,
              text: l10n.photo_copy_failed_toast(failedPhotoCount),
            ),
          );
    }

    if (!mounted) return;
    final savedTrail = readLocalTrail(store, localId);
    if (savedTrail != null) {
      setState(() {
        trail = savedTrail;
        _removedServerPhotos = [];
        _originalTrail = trail;
      });
    }

    // See the identical comment on the network-update branch above for why
    // this must run post-frame.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _formKey.currentState?.reset();
    });

    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.success,
            icon: FontAwesomeIcons.circleCheck,
            text: l10n.trail_saved_successfully,
          ),
        );
  }

  bool get _hasUnsavedChanges =>
      trail != _originalTrail || _formKey.currentState?.isDirty == true;

  Future<void> _confirmDiscard(BuildContext context) async {
    final l10n = AppLocalizations.of(context)!;
    final discard = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(l10n.discard_trail_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.discard),
          ),
        ],
      ),
    );
    if (discard == true && context.mounted) {
      context.canPop() ? context.pop() : context.pushReplacement('/map');
    }
  }

  double _getDynamicPadding(double currentSize) {
    const double minPadding = 0.0;
    const double maxPadding = 96.0;

    double startThreshold = sheetMediumSize;
    double endThreshold = sheetMaxSize;

    if (currentSize <= startThreshold) return minPadding;

    if (currentSize >= endThreshold) return maxPadding;

    double percentage =
        (currentSize - startThreshold) / (endThreshold - startThreshold);

    return minPadding + (percentage * (maxPadding - minPadding));
  }

  @override
  Widget build(BuildContext context) {
    // Default a categoryless trail (e.g. freshly imported from GPX) to the
    // user's top-preferred category, once the providers resolve.
    final categoriesAsync = ref.watch(categoryProvider);
    final categoryPrefsAsync = ref.watch(categoryPreferenceProvider);
    if (!_categoryDefaulted &&
        (trail.category?.isEmpty ?? true) &&
        categoriesAsync.hasValue &&
        categoryPrefsAsync.hasValue) {
      _categoryDefaulted = true;
      final defaultCategory = visibleSortedCategories(
        categoriesAsync.value!,
        categoryPrefsAsync.value!,
        Localizations.localeOf(context),
      ).firstOrNull?.id;
      if (defaultCategory != null) {
        trail = trail.copyWith(category: defaultCategory, subcategory: '');
      }
    }

    // Default a brand-new trail's visibility to the user's privacy setting;
    // never overrides an existing trail's saved visibility.
    final settings = ref.watch(settingsProvider);
    if (!_privacyDefaulted && trail.id.isEmpty && settings != null) {
      _privacyDefaulted = true;
      trail = trail.copyWith(public: settings.privacy?.trails == 'public');
    }

    final isOffline = !ref.watch(onlineStatusProvider);

    return PopScope(
      canPop: !_hasUnsavedChanges,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _confirmDiscard(context);
      },
      child: Scaffold(
        extendBodyBehindAppBar: true,
        appBar: AppBar(
          leading: IconButton(
            icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
            onPressed: () => _hasUnsavedChanges
                ? _confirmDiscard(context)
                : (context.canPop()
                      ? context.pop()
                      : context.pushReplacement('/map')),
            style: IconButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.surface,
            ),
          ),
          actions: [
            if (!isOffline)
              Builder(
                builder: (context) {
                  // Only enabled when the track yields >=2 seedable anchors, so a
                  // trk with no coordinates doesn't silently open "new route" mode.
                  final seedAnchors = trail.expand?.gpx != null
                      ? anchorsFromTrack(trail.expand!.gpx!)
                      : const <ml.Geographic>[];
                  return IconButton(
                    icon: Stack(
                      clipBehavior: Clip.none,
                      children: [
                        const FaIcon(FontAwesomeIcons.route, size: 18),
                        Positioned(
                          top: -2,
                          child: const FaIcon(FontAwesomeIcons.pen, size: 9),
                        ),
                      ],
                    ),
                    tooltip: AppLocalizations.of(context)!.edit_route,
                    onPressed: seedAnchors.length >= 2
                        ? () => _onEditRoute(context, seedAnchors)
                        : null,
                    style: IconButton.styleFrom(
                      backgroundColor: Theme.of(context).colorScheme.surface,
                      disabledBackgroundColor: Theme.of(
                        context,
                      ).colorScheme.surface,
                    ),
                  );
                },
              ),
            IconButton(
              icon: _saving
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const FaIcon(FontAwesomeIcons.floppyDisk, size: 18),
              onPressed: _saving ? null : () => _onSave(context),
              style: IconButton.styleFrom(
                backgroundColor: Theme.of(context).colorScheme.surface,
                disabledBackgroundColor: Theme.of(context).colorScheme.surface,
              ),
            ),
          ],
          backgroundColor: Colors.transparent,

          shadowColor: Colors.transparent,
          elevation: 0,
          scrolledUnderElevation: 0,
        ),

        body: SafeArea(
          top: false,
          child: Stack(
            children: [
              Positioned.fill(
                child: TrailMap(
                  trail: trail,
                  elevationMarkerPosition: _elevationMarkerPosition,
                  showLocation: true,
                  offline: isOffline,
                  initialCameraFitPadding: EdgeInsets.only(
                    bottom: MediaQuery.of(context).size.height * 0.4 + 40,
                    left: 40,
                    right: 40,
                    top: 40,
                  ),
                  controls: [
                    const ml.MapCompass(
                      hideIfRotatedNorth: true,

                      padding: EdgeInsets.only(top: 112, right: 4),
                    ),
                  ],
                  onMapCreated: (controller) => _mapController = controller,
                  onTap: (point) => _onCreateWaypoint(context, at: point),
                  onWaypointTap: (wp) => _onEditWaypoint(context, wp),
                  onWaypointDragEnd: _onWaypointMoved,
                ),
              ),
              DraggableScrollableSheet(
                minChildSize: sheetMinSize,
                maxChildSize: sheetMaxSize,
                initialChildSize: sheetMediumSize,
                snap: true,
                snapSizes: [sheetMinSize, sheetMediumSize, sheetMaxSize],
                shouldCloseOnMinExtent: false,
                controller: _sheetController,

                builder: (context, scrollController) {
                  return Container(
                    padding: EdgeInsets.symmetric(horizontal: 16),
                    decoration: BoxDecoration(
                      color: Theme.of(context).canvasColor,
                      borderRadius: const BorderRadius.vertical(
                        top: Radius.circular(16),
                      ),
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withValues(alpha: 0.15),
                          blurRadius: 10,
                          offset: const Offset(0, -2),
                        ),
                      ],
                    ),
                    child: ValueListenableBuilder<double>(
                      valueListenable: _sheetSize,

                      builder: (context, size, child) {
                        return Padding(
                          padding: EdgeInsets.only(
                            top: size >= sheetMediumSize
                                ? _getDynamicPadding(size)
                                : 8.0,
                          ),
                          child: TrailForm(
                            scrollController: scrollController,
                            formKey: _formKey,
                            trail: trail,
                            onCreateWaypoint: (context) =>
                                _onCreateWaypoint(context),
                            onCreateWaypointsFromPhotos:
                                _onCreateWaypointsFromPhotos,
                            onEditWaypoint: _onEditWaypoint,
                            onDeleteWaypoint: _onDeleteWaypoint,
                            onPhotosChanged: _onServerPhotosChanged,
                            onElevationLineTouch: (p) => setState(
                              () => _elevationMarkerPosition = p?.lonlat,
                            ),
                          ),
                        );
                      },
                    ),
                  );
                },
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class TrailForm extends ConsumerWidget {
  final ScrollController scrollController;
  final GlobalKey<FormBuilderState> formKey;
  final Trail trail;
  final Future<void> Function(BuildContext context) onCreateWaypoint;
  final Future<void> Function() onCreateWaypointsFromPhotos;
  final Future<void> Function(BuildContext context, Waypoint waypoint)
  onEditWaypoint;
  final void Function(Waypoint waypoint) onDeleteWaypoint;
  final void Function(List<String> remainingFilenames) onPhotosChanged;
  final void Function(TrackPoint? point) onElevationLineTouch;
  const TrailForm({
    super.key,
    required this.scrollController,
    required this.formKey,
    required this.trail,
    required this.onCreateWaypoint,
    required this.onCreateWaypointsFromPhotos,
    required this.onEditWaypoint,
    required this.onDeleteWaypoint,
    required this.onPhotosChanged,
    required this.onElevationLineTouch,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return FormBuilder(
      key: formKey,
      autovalidateMode: AutovalidateMode.onUnfocus,
      child: SingleChildScrollView(
        controller: scrollController,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.3),
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            Text(l10n.basic_info, style: theme.textTheme.titleMedium),
            Divider(),
            SizedBox(height: 12),
            if (trail.expand?.gpx != null)
              ElevationProfile(
                trail: trail,
                gpx: trail.expand!.gpx!,
                onLineTouch: onElevationLineTouch,
              ),
            SizedBox(height: 12),
            WandererTextField(
              name: 'name',
              initialValue: trail.name,
              label: l10n.name,
              validator: FormBuilderValidators.required(),
            ),
            SizedBox(height: 12),
            WandererTextField(
              name: 'location',
              initialValue: trail.location,
              label: l10n.location,
            ),
            SizedBox(height: 12),
            WandererDatePicker(
              name: 'date',
              initialValue: trail.date,
              label: l10n.date,
            ),
            SizedBox(height: 12),
            FormBuilderField<String>(
              name: 'description',
              initialValue: trail.description,
              builder: (field) => WandererRichTextEditor(
                label: l10n.description,
                initialValue: trail.description,
                onChanged: field.didChange,
              ),
            ),

            SizedBox(height: 12),
            FormBuilderField<List<Tag>>(
              name: 'tags',
              initialValue: trail.expand?.tags ?? const [],
              builder: (field) {
                return WandererAutocomplete<Tag>(
                  label: l10n.tags,
                  initialTags: trail.expand?.tags
                      ?.map((t) => DynamicTagData(t.name, t))
                      .toList(),
                  optionsBuilder: (textEditingValue) async {
                    final tags = await ref
                        .read(tagProvider.notifier)
                        .searchByName(textEditingValue.text);
                    final tagData = tags
                        .map((t) => DynamicTagData(t.name, t))
                        .toList();

                    return tagData;
                  },
                  onSubmitted: (value) {
                    final newTag = Tag(name: value);
                    field.didChange([...?field.value, newTag]);
                    return DynamicTagData(value, newTag);
                  },
                  onSelected: (value) =>
                      field.didChange([...?field.value, value.data]),
                  onDeleted: (value) => field.didChange(
                    [...?field.value]
                      ..removeWhere((t) => t.name == value.data.name),
                  ),
                );
              },
            ),
            SizedBox(height: 12),
            CategoryPicker(
              name: 'category',
              label: l10n.category,
              category: trail.category,
              subcategory: trail.subcategory,
            ),
            SizedBox(height: 12),
            WandererSelect<TrailDifficulty>(
              name: 'difficulty',
              label: l10n.difficulty,
              initialValue: trail.difficulty,
              items: [
                SelectItem(value: TrailDifficulty.easy, label: l10n.easy),
                SelectItem(
                  value: TrailDifficulty.moderate,
                  label: l10n.moderate,
                ),
                SelectItem(
                  value: TrailDifficulty.difficult,
                  label: l10n.difficult,
                ),
              ],
            ),
            SizedBox(height: 12),

            FormBuilderField<bool>(
              name: 'public',
              initialValue: trail.public,
              builder: (field) {
                final isPublic = field.value ?? false;
                return Material(
                  type: MaterialType.transparency,
                  child: SwitchListTile(
                    secondary: FaIcon(
                      isPublic ? FontAwesomeIcons.globe : FontAwesomeIcons.lock,
                      size: 16,
                    ),
                    title: Text(isPublic ? l10n.public : l10n.private),
                    value: isPublic,
                    activeThumbColor: theme.brightness == Brightness.dark
                        ? theme.colorScheme.onSurface
                        : theme.colorScheme.primary,
                    onChanged: (value) => field.didChange(value),
                  ),
                );
              },
            ),
            SizedBox(height: 12),
            FormBuilderField<bool>(
              name: 'completed',
              initialValue: trail.completed,
              builder: (field) {
                final isCompleted = field.value ?? false;
                return Material(
                  type: MaterialType.transparency,
                  child: SwitchListTile(
                    secondary: FaIcon(
                      isCompleted
                          ? FontAwesomeIcons.flagCheckered
                          : FontAwesomeIcons.compassDrafting,
                      size: 16,
                    ),
                    title: Text(
                      isCompleted ? l10n.completed : l10n.not_completed,
                    ),
                    value: isCompleted,
                    activeThumbColor: theme.brightness == Brightness.dark
                        ? theme.colorScheme.onSurface
                        : theme.colorScheme.primary,
                    onChanged: (value) => field.didChange(value),
                  ),
                );
              },
            ),
            SizedBox(height: 16),
            Text(l10n.photos, style: theme.textTheme.titleMedium),
            Divider(),
            SizedBox(height: 12),
            FormBuilderField<List<String>>(
              name: 'photos',
              initialValue: trail.localPhotos,
              builder: (field) {
                final serverUrl = ref.watch(authProvider).value?.serverUrl;
                return WandererPhotoPicker(
                  initialLocalPhotos: trail.localPhotos,
                  initialWebPhotos: trail.photos,
                  resolveWebPhotoUrl: (filename) =>
                      trail.getFileUrl(
                        serverUrl ?? '',
                        filename,
                        thumb: '400x0',
                      ) ??
                      '',
                  onChanged: (photos) => field.didChange(photos),
                  onWebPhotosChanged: onPhotosChanged,
                );
              },
            ),
            SizedBox(height: 16),
            Text(l10n.waypoints(2), style: theme.textTheme.titleMedium),
            Divider(),
            SizedBox(height: 12),
            for (final wp in trail.expand?.waypointsViaTrail ?? const [])
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: WaypointCard(
                  waypoint: wp,
                  onEdit: () => onEditWaypoint(context, wp),
                  onDelete: () => onDeleteWaypoint(wp),
                ),
              ),
            WandererButton(
              secondary: true,
              onPressed: () => onCreateWaypoint(context),
              child: Row(
                children: [
                  FaIcon(FontAwesomeIcons.plus, size: 16),
                  const SizedBox(width: 6),
                  Text(l10n.create_waypoint),
                ],
              ),
            ),
            SizedBox(height: 16),
            WandererButton(
              secondary: true,
              onPressed: onCreateWaypointsFromPhotos,
              child: Row(
                children: [
                  FaIcon(FontAwesomeIcons.image, size: 16),
                  const SizedBox(width: 6),
                  Text(l10n.from_photos),
                ],
              ),
            ),
            SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}
