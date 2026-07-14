import 'package:dio/dio.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailSourceSelectScreen extends ConsumerStatefulWidget {
  const TrailSourceSelectScreen({super.key});

  @override
  ConsumerState<TrailSourceSelectScreen> createState() =>
      _TrailSourceSelectScreenState();
}

class _TrailSourceSelectScreenState
    extends ConsumerState<TrailSourceSelectScreen> {
  bool _importing = false;

  void _comingSoon(AppLocalizations l10n) {
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.info,
            icon: FontAwesomeIcons.circleInfo,
            text: l10n.coming_soon,
          ),
        );
  }

  static const _allowedExtensions = ['gpx', 'kml', 'kmz', 'tcx', 'fit'];

  Future<void> _importGpx(AppLocalizations l10n) async {
    if (_importing) return;

    final result = await FilePicker.pickFiles(
      type: FileType.custom,
      allowedExtensions: _allowedExtensions,
    );
    final picked = result?.files.single;
    final path = picked?.path;
    if (picked == null || path == null) return;

    // `allowedExtensions` is only a hint — several platforms ignore it and let
    // the user pick any file, so reject unsupported types before uploading.
    final ext = picked.extension?.toLowerCase();
    if (ext == null || !_allowedExtensions.contains(ext)) {
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.trail_source_import_error,
            ),
          );
      return;
    }

    setState(() => _importing = true);
    try {
      final formData = FormData.fromMap({
        'file': await MultipartFile.fromFile(path, filename: picked.name),
      });
      final res = await ref
          .read(apiProvider)
          .post('/trail/convert', data: formData);
      // The trail is not persisted yet, so the server omits id/created/updated.
      // Supply the same placeholders Trail.empty() uses; the spread lets any
      // server-provided value win. The record is timestamped on save.
      final data = res.data as Map<String, dynamic>;
      var trail = Trail.fromJson({
        'id': '',
        'created': DateTime.now().toIso8601String(),
        'updated': DateTime.now().toIso8601String(),
        ...data,
      });

      // The unsaved trail carries its raw GPX inline (no file to fetch yet).
      // Parse it into expand.gpx so the route can be drawn on the map. The
      // Gpx object isn't serializable, so this step stays client-side; the
      // bounding box already arrives on the record from gpx2trail.
      final gpxData = trail.expand?.gpxData;
      if (gpxData != null && gpxData.isNotEmpty) {
        final parsedGpx = GpxReader().fromString(sanitizeGpxEmail(gpxData));
        trail = trail.copyWith(
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            gpx: parsedGpx,
          ),
        );
      }

      if (!mounted) return;
      context.pushReplacement('/trail/create/edit', extra: trail);
    } catch (e) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.trail_source_import_error,
            ),
          );
    } finally {
      if (mounted) setState(() => _importing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.new_trail)),
      body: ListView(
        children: [
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.route),
            title: Text(l10n.trail_source_planner),
            trailing: const FaIcon(FontAwesomeIcons.chevronRight, size: 16),
            onTap: () => _comingSoon(l10n),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.solidCircleDot),
            title: Text(l10n.trail_source_record),
            trailing: const FaIcon(FontAwesomeIcons.chevronRight, size: 16),
            onTap: () => _comingSoon(l10n),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.fileArrowUp),
            title: Text(l10n.trail_source_import),
            trailing: _importing
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const FaIcon(FontAwesomeIcons.chevronRight, size: 16),
            onTap: _importing ? null : () => _importGpx(l10n),
          ),
        ],
      ),
    );
  }
}
