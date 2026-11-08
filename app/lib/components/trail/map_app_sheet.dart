import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/util/route/map_app.dart';

/// Presents the iOS map-app picker: one row per installed map application.
///
/// iOS has no system app chooser, so this stands in for the one Android gets
/// for free from a `geo:` intent. Returns the tapped [MapApp], or `null` if
/// the sheet is dismissed without a selection.
Future<MapApp?> showMapAppSheet(BuildContext context, List<MapApp> apps) {
  return showModalBottomSheet<MapApp>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) {
      final theme = Theme.of(context);
      final l18n = AppLocalizations.of(context)!;

      return SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
                child: Text(
                  l18n.directions,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              for (final app in apps)
                ListTile(
                  key: ValueKey(app.id),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                  leading: const FaIcon(
                    FontAwesomeIcons.diamondTurnRight,
                    size: 18,
                  ),
                  title: Text(app.name),
                  onTap: () => Navigator.pop(context, app),
                ),
            ],
          ),
        ),
      );
    },
  );
}
