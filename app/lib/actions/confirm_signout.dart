/// Shared confirmation before signing out with pending trail or photo work.
/// Locally captured trails remain on the device; account-scoped photo jobs are
/// removed only after a confirmed sign-out; queued captures remain.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';

/// Whether a sign-out warning should be shown for [unsyncedCount] pending
/// trails.
///
/// A single named, testable decision rather than an inline `> 0` comparison
/// duplicated at every call site.
bool shouldWarnBeforeSignOut(int unsyncedCount) => unsyncedCount > 0;

/// Counts pending trails and journal jobs and asks before signing out.
/// Cancelling retains both the session and its pending changes.
Future<bool> confirmSignOutWithUnsyncedTrails(
  BuildContext context,
  WidgetRef ref,
) async {
  final store = ref.read(objectBoxProvider);
  final accountId = currentAccountId(store);
  if (accountId == null) return true;

  final user = store.box<UserEntity>().getAll().firstOrNull;
  final journalCount = user == null
      ? 0
      : await ref.read(photoMutationServiceProvider).pendingCount(user.actorId);
  final unsyncedCount = countUnsyncedTrails(store, accountId) + journalCount;
  if (!shouldWarnBeforeSignOut(unsyncedCount)) return true;

  if (!context.mounted) return true;

  final l10n = AppLocalizations.of(context)!;
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      content: Text(
        journalCount > 0
            ? l10n.signout_photo_jobs_warning(unsyncedCount)
            : l10n.signout_unsynced_warning(unsyncedCount),
      ),
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
