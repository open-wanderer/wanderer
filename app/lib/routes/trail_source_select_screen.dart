import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/components/route_planner/travel_profile_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/map_camera_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/trail_import_util.dart';

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

  Future<void> _openPlanner(AppLocalizations l10n) async {
    final profile = await showTravelProfileSheet(context);
    if (!mounted || profile == null) return;

    setState(() => _importing = true);
    try {
      final center = await _resolveInitialCenter();
      if (!mounted) return;
      context.push(
        '/route-planner',
        extra: {'travelProfile': profile, 'lat': center.lat, 'lon': center.lon},
      );
    } finally {
      if (mounted) setState(() => _importing = false);
    }
  }

  Future<Geographic> _resolveInitialCenter() async {
    final settings = ref.read(settingsProvider);
    final fallback = _fallbackCenter(settings);

    if (settings?.behavior?.allowAutoGeolocate != true) return fallback;

    try {
      final pos = await ref
          .read(foregroundPositionStreamProvider)
          .firstWhere((p) => p != null)
          .timeout(const Duration(seconds: 4));
      if (pos == null) return fallback;
      return Geographic(lat: pos.latitude, lon: pos.longitude);
    } catch (_) {
      return fallback;
    }
  }

  Geographic _fallbackCenter(Settings? settings) {
    final cam = ref.read(mapCameraProvider);
    if (cam != null) return cam.center;

    final loc = settings?.location;
    if (loc != null) return Geographic(lat: loc.lat, lon: loc.lon);

    return const Geographic(lat: 0, lon: 0);
  }

  Future<void> _importGpx(AppLocalizations l10n) async {
    if (_importing) return;

    final result = await FilePicker.pickFiles(
      type: FileType.custom,
      allowedExtensions: trailImportExtensions,
    );
    final picked = result?.files.single;
    final path = picked?.path;
    if (picked == null || path == null) return;

    setState(() => _importing = true);
    try {
      if (!mounted) return;
      await importTrailFile(
        ref: ref,
        path: path,
        name: picked.name,
        navContext: context,
        l10n: l10n,
      );
    } finally {
      if (mounted) setState(() => _importing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.new_trail),
        automaticallyImplyLeading: false,
      ),
      body: ListView(
        padding: EdgeInsets.symmetric(horizontal: 12),
        children: [
          _SourceActionCard(
            icon: FontAwesomeIcons.route,
            title: l10n.trail_source_planner,
            description:
                "Design your perfect route from scratch using our map tools.",
            isLoading: _importing,
            onTap: _importing ? null : () => _openPlanner(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.solidCircleDot,
            title: l10n.trail_source_record,
            description:
                "Track your live coordinates and log your journey in real-time.",
            onTap: _importing ? null : () => _comingSoon(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.fileArrowUp,
            title: l10n.trail_source_import,
            description:
                "Upload external GPX files directly from your device storage.",
            isLoading: _importing,
            onTap: _importing ? null : () => _importGpx(l10n),
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
    required this.icon,
    required this.title,
    required this.description,
    this.onTap,
    this.isLoading = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final disabled = onTap == null && !isLoading;

    final resolvedBgColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.05)
        : theme.colorScheme.secondaryContainer.withValues(alpha: 0.4);
    final resolvedIconColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.38)
        : theme.colorScheme.primary;
    final resolvedTitleColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.38)
        : null;
    final resolvedDescriptionColor = disabled
        ? theme.colorScheme.onSurface.withValues(alpha: 0.38)
        : theme.colorScheme.onSurfaceVariant;
    final resolvedBorderColor = disabled
        ? theme.colorScheme.outline.withValues(alpha: 0.3)
        : theme.colorScheme.outline;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: resolvedBorderColor),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
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
                        color: resolvedTitleColor,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: resolvedDescriptionColor,
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
                FaIcon(
                  FontAwesomeIcons.chevronRight,
                  size: 16,
                  color: disabled
                      ? theme.colorScheme.onSurface.withValues(alpha: 0.38)
                      : null,
                ),
            ],
          ),
        ),
      ),
    );
  }
}
