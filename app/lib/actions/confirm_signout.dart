/// Shared count-and-confirm gate in front of every sign-out call site.
///
/// A locally-captured trail (`TrailEntity` with a non-null `owner`) is never
/// purged on logout (`account_data_purge.dart`) -- nothing is ever lost
/// by signing out. But the hiker has no way to know that, and signing into a
/// different account while trails are still waiting to upload is the single
/// most common "my tour is missing" report the design record cites (D-12).
///
/// This gate never blocks the sign-out itself. It exists only to tell the
/// hiker, before they act, how many trails are still pending -- confirming
/// proceeds exactly as if the dialog never appeared, and cancelling simply
/// leaves them signed in.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';

/// Whether a sign-out warning should be shown for [unsyncedCount] pending
/// trails.
///
/// A single named, testable decision rather than an inline `> 0` comparison
/// duplicated at every call site.
bool shouldWarnBeforeSignOut(int unsyncedCount) => unsyncedCount > 0;

/// Resolves the current account's unsynced trail count and, if
/// [shouldWarnBeforeSignOut] is true for it, shows a warning dialog naming
/// that count before the caller proceeds with sign-out.
///
/// Returns `true` when the sign-out should proceed: either there is no
/// signed-in account, there is nothing pending, or the hiker confirmed the
/// warning. Returns `false` only when the hiker cancelled or dismissed the
/// dialog.
///
/// The sign-out itself is never blocked by this function -- `TrailEntity`
/// rows deliberately survive logout (`account_data_purge.dart`), so
/// this dialog exists only to tell the hiker that, not to stop them.
///
/// A typical call site: `if (!await confirmSignOutWithUnsyncedTrails(context,
/// ref)) return;` before calling the real sign-out.
Future<bool> confirmSignOutWithUnsyncedTrails(
  BuildContext context,
  WidgetRef ref,
) async {
  final store = ref.read(objectBoxProvider);
  final accountId = currentAccountId(store);
  if (accountId == null) return true;

  final unsyncedCount = countUnsyncedTrails(store, accountId);
  if (!shouldWarnBeforeSignOut(unsyncedCount)) return true;

  if (!context.mounted) return true;

  final l10n = AppLocalizations.of(context)!;
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      content: Text(l10n.signout_unsynced_warning(unsyncedCount)),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(false),
          child: Text(l10n.cancel),
        ),
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(true),
          child: Text(l10n.logout),
        ),
      ],
    ),
  );

  return confirmed ?? false;
}
