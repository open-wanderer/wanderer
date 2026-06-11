import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

enum TrailAction { open, directions, download, delete }

class TrailDropdown extends ConsumerWidget {
  final Trail trail;
  const TrailDropdown({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l18n = AppLocalizations.of(context)!;
    final isOffline = ref
        .watch(trailLibraryProvider)
        .any((t) => t.id == trail.id);
    return PopupMenuButton<TrailAction>(
      offset: const Offset(0, 60),
      child: FloatingActionButton(
        onPressed: null,
        elevation: 0,
        mini: true,
        child: FaIcon(
          FontAwesomeIcons.ellipsisVertical,
          color: Theme.of(context).colorScheme.onPrimaryContainer,
          size: 18,
        ),
      ),
      itemBuilder: (BuildContext context) => <PopupMenuEntry<TrailAction>>[
        PopupMenuItem<TrailAction>(
          value: TrailAction.open,
          onTap: () => context.push('/trail/${trail.id}/map'),
          child: ListTile(
            leading: FaIcon(FontAwesomeIcons.map, size: 18),
            title: Text(l18n.show_on_map),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        PopupMenuItem<TrailAction>(
          value: TrailAction.directions,
          onTap: trail.lat != null && trail.lon != null
              ? () => _openDirections(trail.lat!, trail.lon!)
              : null,
          child: ListTile(
            leading: FaIcon(
              FontAwesomeIcons.car,
              size: 18,
              color: trail.lat == null ? Colors.grey : null,
            ),
            title: Text(
              l18n.directions,
              style: trail.lat == null
                  ? const TextStyle(color: Colors.grey)
                  : null,
            ),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        const PopupMenuDivider(),
        PopupMenuItem<TrailAction>(
          value: TrailAction.download,
          onTap: isOffline ? null : () => _downloadTrail(context, ref, trail),
          enabled: !isOffline,
          child: ListTile(
            leading: FaIcon(
              isOffline
                  ? FontAwesomeIcons.circleCheck
                  : FontAwesomeIcons.download,
              size: 18,
              color: isOffline ? Colors.green : null,
            ),
            title: Text(
              isOffline ? 'Available offline' : l18n.download,
              style: isOffline ? const TextStyle(color: Colors.green) : null,
            ),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        if (_allowDelete(ref)) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.delete,
            child: ListTile(
              leading: FaIcon(
                FontAwesomeIcons.trash,
                color: Colors.red,
                size: 18,
              ),
              title: Text(l18n.delete, style: TextStyle(color: Colors.red)),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
      ],
    );
  }

  bool _allowDelete(WidgetRef ref) {
    return false;
    final user = ref.watch(authProvider).value;
    return user != null && user.actorId == trail.author;
  }

  Future<void> _openDirections(double lat, double lon) async {
    final nativeUrl = Uri.parse(
      Platform.isIOS ? 'maps://?daddr=$lat,$lon' : 'geo:$lat,$lon?q=$lat,$lon',
    );
    final webUrl = Uri.parse(
      'https://www.openstreetmap.org/directions?to=$lat,$lon',
    );
    if (await canLaunchUrl(nativeUrl)) {
      await launchUrl(nativeUrl);
    } else {
      await launchUrl(webUrl, mode: LaunchMode.externalApplication);
    }
  }

  void _downloadTrail(BuildContext context, WidgetRef ref, Trail trail) async {
    final trailDownloadService = ref.read(trailDownloadServiceProvider);
    final notificationService = ref.read(downloadNotificationServiceProvider);

    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.info,
            icon: FontAwesomeIcons.download,
            text: 'Downloading ${trail.name}...',
          ),
        );

    try {
      await trailDownloadService.downloadTrail(
        trail,
        onProgress: (done, total) =>
            notificationService.showProgress(trail.name, done, total),
      );
      await notificationService.showSuccess(trail.name);
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.success,
              icon: FontAwesomeIcons.circleCheck,
              text: 'Trail saved for offline use',
            ),
          );
    } catch (e) {
      await notificationService.showError(trail.name);
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.xmark,
              text: 'Error saving trail',
            ),
          );
    }
  }
}
