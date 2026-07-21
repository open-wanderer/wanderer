import 'dart:io' show Platform;

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/components/route_planner/travel_profile_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/map_camera_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/route_travel_bucket.dart';
import 'package:wanderer/util/tracelet_position_source.dart';
import 'package:wanderer/util/trail_import_util.dart';

class TrailSourceSelectScreen extends ConsumerStatefulWidget {
  const TrailSourceSelectScreen({super.key});

  @override
  ConsumerState<TrailSourceSelectScreen> createState() =>
      _TrailSourceSelectScreenState();
}

class _TrailSourceSelectScreenState
    extends ConsumerState<TrailSourceSelectScreen> {
  bool _importLoading = false;
  bool _plannerLoading = false;
  bool _recorderLoading = false;

  /// Requests location permission (mirroring `navigation_launch_util.dart`'s
  /// `launchNavigation` gate — `NavigationScreen` does not self-request it),
  /// then waits for a real GPS fix before pushing the trail-less
  /// GPS-recording session — without this, `NavigationScreen` has no
  /// `response.shape` to derive a map center from and would briefly open at
  /// `Geographic(0, 0)`. `_recorderLoading` drives the card's spinner and
  /// disables all three source cards for the duration, matching
  /// `_openPlanner`/`_importGpx`'s existing loading-flag pattern.
  Future<void> _openRecorder(AppLocalizations l10n) async {
    if (_recorderLoading) return;

    void showError(String text) => ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.triangleExclamation,
            text: text,
          ),
        );

    if (!await Geolocator.isLocationServiceEnabled()) {
      showError(l10n.location_services_disabled);
      return;
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.deniedForever) {
      showError(l10n.location_permission_permanently_denied);
      return;
    }
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
      if (permission == LocationPermission.denied ||
          permission == LocationPermission.deniedForever) {
        showError(l10n.location_permission_denied);
        return;
      }
    }
    // iOS: re-requesting when WhenInUse triggers the "Change to Always
    // Allow?" prompt (requires NSLocationAlwaysAndWhenInUseUsageDescription).
    // No-op on Android. Recording proceeds either way if declined.
    if (permission == LocationPermission.whileInUse && Platform.isIOS) {
      permission = await Geolocator.requestPermission();
    }

    if (!mounted) return;
    setState(() => _recorderLoading = true);
    try {
      LocationMarkerPosition? pos;
      try {
        pos = await ref
            .read(foregroundPositionStreamProvider)
            .firstWhere((p) => p != null)
            .timeout(const Duration(seconds: 20));
      } catch (_) {
        pos = null;
      }
      if (!mounted) return;
      if (pos == null) {
        showError(l10n.location_unavailable);
        return;
      }
      context.push(
        '/record',
        extra: {
          'lat': pos.latitude,
          'lon': pos.longitude,
          'position': seedPositionFrom(pos),
        },
      );
    } finally {
      if (mounted) setState(() => _recorderLoading = false);
    }
  }

  Future<void> _openPlanner(AppLocalizations l10n) async {
    if (_plannerLoading) return;
    final bucket = await showTravelProfileSheet(context);
    if (!mounted || bucket == null) return;

    setState(() => _plannerLoading = true);
    try {
      final center = await _resolveInitialCenter();
      if (!mounted) return;
      context.push(
        '/route-planner',
        extra: {
          'travelProfile': bucket.costing,
          'costingOptions': bucket.costingOptions,
          'lat': center.lat,
          'lon': center.lon,
        },
      );
    } finally {
      if (mounted) setState(() => _plannerLoading = false);
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
    if (_importLoading) return;

    final result = await FilePicker.pickFiles(
      type: FileType.custom,
      allowedExtensions: trailImportExtensions,
    );
    final picked = result?.files.single;
    final path = picked?.path;
    if (picked == null || path == null) return;

    setState(() => _importLoading = true);
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
      if (mounted) setState(() => _importLoading = false);
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
            isLoading: _plannerLoading,
            onTap: (_importLoading || _plannerLoading || _recorderLoading)
                ? null
                : () => _openPlanner(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.solidCircleDot,
            title: l10n.trail_source_record,
            description:
                "Track your live coordinates and log your journey in real-time.",
            isLoading: _recorderLoading,
            onTap: (_importLoading || _plannerLoading || _recorderLoading)
                ? null
                : () => _openRecorder(l10n),
          ),
          const SizedBox(height: 12),
          _SourceActionCard(
            icon: FontAwesomeIcons.fileArrowUp,
            title: l10n.trail_source_import,
            description:
                "Upload external GPX files directly from your device storage.",
            isLoading: _importLoading,
            onTap: (_importLoading || _plannerLoading || _recorderLoading)
                ? null
                : () => _importGpx(l10n),
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
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: resolvedBgColor,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: FaIcon(icon, color: resolvedIconColor),
              ),
              const SizedBox(width: 16),

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
