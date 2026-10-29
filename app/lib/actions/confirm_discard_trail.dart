/// Shared "discard this trail?" gate for screens that leave unsaved work
/// behind: the trail editor and the route planner.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Asks whether to discard the trail being edited and, if confirmed, leaves
/// the screen — popping when there is a route below, otherwise replacing it
/// with the map.
///
/// Callers wire this to both exits: `PopScope.onPopInvokedWithResult` for the
/// system back gesture, and their app-bar back button, which pops directly
/// and so never reaches the `PopScope`.
Future<void> confirmDiscardTrail(BuildContext context) async {
  final l10n = AppLocalizations.of(context)!;
  final discard = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      content: Text(l10n.discard_trail_confirm),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(false),
          child: Text(l10n.cancel),
        ),
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(true),
          child: Text(l10n.discard),
        ),
      ],
    ),
  );
  if (discard == true && context.mounted) {
    context.canPop() ? context.pop() : context.pushReplacement('/map');
  }
}
