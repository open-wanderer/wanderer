import 'dart:io';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
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
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/util/category_preference_sort.dart';
import 'package:wanderer/util/exif_util.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailCreateScreen extends ConsumerStatefulWidget {
  final Trail trail;
  const TrailCreateScreen({super.key, required this.trail});

  @override
  ConsumerState<TrailCreateScreen> createState() => _TrailCreateScreenState();
}

class _TrailCreateScreenState extends ConsumerState<TrailCreateScreen> {
  late Trail trail = widget.trail;
  final DraggableScrollableController _sheetController =
      DraggableScrollableController();

  // Owned here (not by TrailForm) so a Save action can read the edited values
  // back out: `_formKey.currentState!.saveAndValidate()` then
  // `trail.copyWith(name: _formKey.currentState!.value['name'] as String)`.
  final _formKey = GlobalKey<FormBuilderState>();

  ml.MapController? _mapController;

  double sheetMinSize = 0.1;
  final sheetMediumSize = 0.4;
  final sheetMaxSize = 1.0;

  late final ValueNotifier<double> _sheetSize;

  bool _saving = false;
  List<String> _removedServerPhotos = [];

  @override
  void initState() {
    super.initState();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // A trail arriving here without a category (e.g. freshly imported from a
    // GPX file) should already show the user's top-preferred category rather
    // than an empty picker. Guarded by the empty check so it never overwrites
    // a category the user (or the source screen) already set.
    if (trail.category?.isEmpty ?? true) {
      final defaultCategory = _firstPreferredCategoryId(
        Localizations.localeOf(context),
      );
      if (defaultCategory != null) {
        trail = trail.copyWith(category: defaultCategory, subcategory: '');
      }
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
    // Defaults to the center of the currently visible map section (e.g. from
    // the "Create waypoint" button); a tap on the map passes the tapped point.
    final point = at ?? _mapController?.camera?.center;
    final stub = Waypoint(
      id: DateTime.now().microsecondsSinceEpoch.toString(),
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

  /// Picks multiple photos and creates one waypoint per photo that carries GPS
  /// EXIF data (coordinates pre-filled, the photo attached). Photos without GPS
  /// are skipped and reported.
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
            id: '${now.microsecondsSinceEpoch}-${created.length}',
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
            ..removeWhere((wp) => wp.id == waypoint.id),
        ),
      );
    });
  }

  /// Estimates [waypoint]'s `distanceFromStart` from the trail's GPX track so
  /// it shows up on the elevation profile immediately, ahead of the server
  /// recomputing the exact value on save.
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
              if (wp.id == withDistance.id) withDistance else wp,
          ],
        ),
      );
    });
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

    try {
      final result = trail.id.isEmpty
          ? await ref
                .read(trailSaveProvider.notifier)
                .createTrail(
                  updatedTrail,
                  authorId: authorId,
                  newPhotos: newPhotoFiles,
                )
          : await ref
                .read(trailSaveProvider.notifier)
                .updateTrail(
                  trail,
                  updatedTrail,
                  authorId: authorId,
                  newPhotos: newPhotoFiles,
                  removedPhotoFilenames: _removedServerPhotos,
                );

      if (!mounted) return;
      setState(() {
        trail = result.trail;
        _removedServerPhotos = [];
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

  /// Id of the first category the user would see in [CategoryPicker], i.e.
  /// the top of their preference ordering (priority first, then locale name)
  /// among categories they haven't hidden. Used to default a trail that has
  /// no category yet, rather than leaving it uncategorized.
  String? _firstPreferredCategoryId(Locale locale) {
    final categories = ref.read(categoryProvider).value ?? const [];
    final categoryPrefs = ref.read(categoryPreferenceProvider).value ?? const [];

    final ordered = sortedCategoriesByPreference(
      categories,
      categoryPrefs,
      locale,
    ).where((c) => categoryVisible(c.id, categoryPrefs));

    return ordered.firstOrNull?.id;
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
    return Scaffold(
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
          onPressed: () => context.pop(),
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.surface,
          ),
        ),
        actions: [
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
                showLocation: true,
                offline: trail.isOffline,
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
              ElevationProfile(trail: trail, gpx: trail.expand!.gpx!),
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
                      trail.getFileUrl(serverUrl ?? '', filename, thumb: '400x0') ??
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
