import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:go_router/go_router.dart';
import 'package:textfield_tags/textfield_tags.dart';
import 'package:wanderer/components/base/trail_map.dart';
import 'package:wanderer/components/base/wanderer_autocomplete.dart';
import 'package:wanderer/components/base/wanderer_date_picker.dart';
import 'package:wanderer/components/base/wanderer_rich_text_editor.dart';
import 'package:wanderer/components/base/wanderer_select.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/components/trail/category_picker.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/trail/tag_provider.dart';

class TrailCreateScreen extends StatefulWidget {
  final Trail trail;
  const TrailCreateScreen({super.key, required this.trail});

  @override
  State<TrailCreateScreen> createState() => _TrailCreateScreenState();
}

class _TrailCreateScreenState extends State<TrailCreateScreen> {
  late final Trail trail = widget.trail;
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

  @override
  void initState() {
    super.initState();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
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
            icon: const FaIcon(FontAwesomeIcons.floppyDisk, size: 18),
            onPressed: () {},
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
  const TrailForm({
    super.key,
    required this.scrollController,
    required this.formKey,
    required this.trail,
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
          ],
        ),
      ),
    );
  }
}
