import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_provider.dart';

enum TrailAction { edit, share, download, delete }

class TrailDropdown extends ConsumerWidget {
  final Trail trail;
  const TrailDropdown({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l18n = AppLocalizations.of(context)!;
    return PopupMenuButton<TrailAction>(
      offset: const Offset(0, 60),
      child: FloatingActionButton(
        onPressed: null,
        elevation: 0,
        child: FaIcon(
          FontAwesomeIcons.ellipsisVertical,
          color: Theme.of(context).colorScheme.onPrimaryContainer,
        ),
      ),
      onSelected: (TrailAction action) {
        switch (action) {
          case TrailAction.edit:
            break;
          case TrailAction.share:
            break;
          case TrailAction.download:
            break;
          case TrailAction.delete:
            break;
        }
      },
      itemBuilder: (BuildContext context) => <PopupMenuEntry<TrailAction>>[
        PopupMenuItem<TrailAction>(
          value: TrailAction.edit,
          child: ListTile(
            leading: FaIcon(FontAwesomeIcons.pen, size: 18),
            title: Text(l18n.edit),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        PopupMenuItem<TrailAction>(
          value: TrailAction.share,
          child: ListTile(
            leading: FaIcon(FontAwesomeIcons.share, size: 18),
            title: Text(l18n.share),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        PopupMenuItem<TrailAction>(
          value: TrailAction.download,
          onTap: () => _downloadTrail(context, ref, trail),
          child: ListTile(
            leading: FaIcon(FontAwesomeIcons.download, size: 18),
            title: Text(l18n.download),
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
    final user = ref.watch(authProvider).value;
    return user != null && user.actorId == trail.author;
  }

  void _downloadTrail(BuildContext context, WidgetRef ref, Trail trail) async {
    final trailDownloadService = ref.read(trailDownloadServiceProvider);

    try {
      await trailDownloadService.downloadTrail(trail);
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
