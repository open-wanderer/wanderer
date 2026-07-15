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
        padding: EdgeInsets.symmetric(horizontal: 12),
        children: [
          _SourceActionCard(
            icon: FontAwesomeIcons.route,
            title: l10n.trail_source_planner,
            description:
                "Design your perfect route from scratch using our map tools.",
            onTap: () => _comingSoon(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.solidCircleDot,
            title: l10n.trail_source_record,
            description:
                "Track your live coordinates and log your journey in real-time.",
            onTap: () => _comingSoon(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.fileArrowUp,
            title: l10n.trail_source_import,
            description:
                "Upload external GPX files directly from your device storage.",
            isLoading: _importing,
            onTap: () => _importGpx(l10n),
          ),
        ],
      ),
    );
  }
}

class _SourceActionCard extends StatelessWidget {
  final FaIconData icon;
  final String title;
  final String description;
  final VoidCallback? onTap;
  final bool isLoading;

  const _SourceActionCard({
    super.key,
    required this.icon,
    required this.title,
    required this.description,
    this.onTap,
    this.isLoading = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final resolvedBgColor = theme.colorScheme.secondaryContainer.withValues(
      alpha: 0.4,
    );
    final resolvedIconColor = theme.colorScheme.primary;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outline),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: isLoading ? null : onTap,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Row(
            children: [
              // Icon Badge container
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: resolvedBgColor,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: FaIcon(icon, color: resolvedIconColor),
              ),
              const SizedBox(width: 16),

              // Text Content
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 16),

              // Trailing action (Spinner or Chevron)
              if (isLoading)
                const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              else
                const FaIcon(FontAwesomeIcons.chevronRight, size: 16),
            ],
          ),
        ),
      ),
    );
  }
}
