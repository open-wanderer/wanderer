import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_icon_picker.dart';
import 'package:wanderer/components/base/wanderer_photo_picker.dart';
import 'package:wanderer/components/base/wanderer_text_area.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/exif_util.dart';

class WaypointCreateScreen extends ConsumerStatefulWidget {
  final Waypoint waypoint;
  const WaypointCreateScreen({super.key, required this.waypoint});

  @override
  ConsumerState<WaypointCreateScreen> createState() =>
      _WaypointCreateScreenState();
}

class _WaypointCreateScreenState extends ConsumerState<WaypointCreateScreen> {
  late final Waypoint waypoint = widget.waypoint;
  final _formKey = GlobalKey<FormBuilderState>();

  late List<String> _localPhotos = List.of(waypoint.localPhotos);

  Future<void> _fillCoordinatesFromExif(String path) async {
    final coords = await readGpsFromImage(path);
    if (!mounted) return;

    final l10n = AppLocalizations.of(context)!;
    if (coords == null || coords.lat.isNaN || coords.lon.isNaN) {
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.no_gps_data_in_image,
            ),
          );
      return;
    }

    _formKey.currentState?.fields['lat']?.didChange(coords.lat.toString());
    _formKey.currentState?.fields['lon']?.didChange(coords.lon.toString());
  }

  void _onSave() {
    if (!(_formKey.currentState?.saveAndValidate() ?? false)) return;
    final v = _formKey.currentState!.value;

    final result = waypoint.copyWith(
      name: v['name'] as String? ?? '',
      description: v['description'] as String? ?? '',
      icon: v['icon'] as FaIconData,
      lat: double.parse(v['lat'] as String),
      lon: double.parse(v['lon'] as String),
      localPhotos: _localPhotos,
    );

    context.pop(result);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final isEdit = waypoint.name?.isNotEmpty == true;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
          onPressed: () => context.pop(),
        ),
        title: Text(isEdit ? l10n.edit_waypoint : l10n.add_waypoint),
        actions: [
          IconButton(
            icon: const FaIcon(FontAwesomeIcons.floppyDisk, size: 18),
            onPressed: _onSave,
          ),
        ],
      ),
      body: SafeArea(
        child: FormBuilder(
          key: _formKey,
          autovalidateMode: AutovalidateMode.onUnfocus,
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                WandererTextField(
                  name: 'name',
                  initialValue: waypoint.name,
                  label: l10n.name,
                ),
                const SizedBox(height: 12),
                WandererIconPicker(
                  name: 'icon',
                  initialValue: waypoint.icon,
                  label: l10n.icon,
                ),
                const SizedBox(height: 12),
                WandererTextArea(
                  name: 'description',
                  initialValue: waypoint.description,
                  label: l10n.description,
                ),
                const SizedBox(height: 12),
                WandererTextField(
                  name: 'lat',
                  initialValue: waypoint.lat.toString(),
                  label: l10n.latitude,
                  validator: FormBuilderValidators.compose([
                    FormBuilderValidators.required(),
                    FormBuilderValidators.numeric(),
                  ]),
                ),
                const SizedBox(height: 12),
                WandererTextField(
                  name: 'lon',
                  initialValue: waypoint.lon.toString(),
                  label: l10n.longitude,
                  validator: FormBuilderValidators.compose([
                    FormBuilderValidators.required(),
                    FormBuilderValidators.numeric(),
                  ]),
                ),
                const SizedBox(height: 16),
                Text(l10n.photos, style: theme.textTheme.titleMedium),
                const Divider(),
                const SizedBox(height: 8),
                WandererPhotoPicker(
                  initialLocalPhotos: waypoint.localPhotos,
                  onChanged: (photos) => _localPhotos = photos,
                  onExifRequested: _fillCoordinatesFromExif,
                ),
                const SizedBox(height: 24),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
