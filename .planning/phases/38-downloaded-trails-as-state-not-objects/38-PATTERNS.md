# Phase 38: Downloaded Trails as State, Not Objects - Pattern Map

**Mapped:** 2026-08-04
**Files analyzed:** 12 (11 modified, 1 new l10n surface — no new .dart file)
**Analogs found:** 12 / 12 (this phase is almost entirely modification of existing files, so the "analog" is frequently the file's own neighbouring code)

> No RESEARCH.md exists for this phase by design — CONTEXT.md carries verified call chains with file:line.

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/components/trail/trail_dropdown.dart` | component (menu) | event-driven | itself (`_confirmDelete`, download item) + `app/lib/routes/settings_offline_regions_screen.dart:1020-1049` | exact |
| `app/lib/components/trail/trail_panel.dart` (~:214) | component (badge) | transform (model → pill) | `app/lib/routes/trail_detail_screen.dart:133-135` (`availableOffline` derivation) | exact |
| `app/lib/provider/trail/trail_provider.dart` | provider (async notifier) | request-response + file/DB write | itself (`_readCached`) + `app/lib/store/local_trail_store.dart:557-589` | exact |
| `app/lib/provider/trail/trail_provider.g.dart` | generated | n/a | `app/lib/provider/trail/local_trail_provider.g.dart` (single positional arg family) | exact |
| `app/lib/routes/trail_create_screen.dart` (~:768) | route/screen | request-response → DB write | itself (existing `reconcileLocalId` block) | exact |
| `app/lib/store/local_trail_store.dart` | store (ObjectBox) | CRUD (write txn) | `applyNetworkEditToLocalRow` itself + `trail_download_service.dart:187-218` | exact |
| `app/lib/util/trail/route_location.dart` | utility (pure) | transform | itself (pre-`forceOffline` shape, recoverable from git) | exact |
| `app/lib/provider/router_provider.dart` (:381-404) | config (routes) | request-response | the `/trail/local/:localId` GoRoute directly above (`:370-378`) — a builder with no query params | exact |
| `app/lib/routes/library_screen.dart` (:120-149) | route/screen | CRUD list | any other `trailDetailLocation(trail)` call site | exact |
| `app/lib/routes/trail_detail_screen.dart` | route/screen | request-response | itself | exact |
| `app/lib/components/trail/like_button.dart` (:12,:21,:39) | component | event-driven | itself | exact |
| `app/lib/i18n/app_en.arb` + 13 locales + `untranslated_messages.json` | config (l10n) | n/a | `regions_delete_confirm_*` block (`app_en.arb:447-456`) and `delete_needs_connection` | exact |

---

## Pattern Assignments

### `app/lib/components/trail/trail_dropdown.dart` — the un-download confirm dialog (D-04)

**Analog:** `app/lib/routes/settings_offline_regions_screen.dart:1020-1049`

This is the shape to mirror. Note it is the **awaited `showDialog<bool>` + `if (confirmed != true) return;`** form, which differs from `trail_dropdown.dart`'s own fire-and-forget `showDialog` in `_confirmDelete` (`:275-293`). Prefer the offline-regions form: it is newer, it is what D-04 names, and it keeps the action out of the dialog's `onPressed` closure.

```dart
  /// D-02: the Vector tile's delete action cascades to remove the DEM
  /// package too (one on-device region has one storage directory), so it
  /// requires a confirm dialog first — ...
  Future<void> _onDeleteRegion(RegionEntity region) async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.regions_delete_confirm_title(region.name)),
        content: Text(l10n.regions_delete_confirm_body),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(
              l10n.regions_delete_confirm_action,
              style: TextStyle(color: Colors.red.shade400),
            ),
          ),
        ],
      ),
    );

    if (confirmed != true) return;
    if (!mounted) return;

    await _save(
      () => ref.read(tileRepositoryStatusProvider.notifier).delete(region.path),
    );
  }
```

Load-bearing details to copy:
- Red is `Colors.red.shade400` here; `trail_dropdown.dart` uses bare `Colors.red` for its Delete. Either is defensible — pick one and be consistent within the file.
- `l10n` is resolved **before** the await; the `!mounted` guard sits after it. `trail_dropdown.dart` documents the same discipline explicitly at `:342-344` and `:436-439` (`context` is a *parameter* there, so a post-await `mounted` check does not license reading from it — resolve `AppLocalizations` and `GoRouter` up front).
- Title takes a placeholder (`region.name`). The trail equivalent may or may not want the trail name; if it does, copy the `@key` placeholder block below.

**Existing l10n keys (`app/lib/i18n/app_en.arb:447-456`):**

```json
  "regions_delete_confirm_title": "Delete {name}?",
  "@regions_delete_confirm_title": {
    "placeholders": {
      "name": {
        "type": "String"
      }
    }
  },
  "regions_delete_confirm_body": "This removes the downloaded map and elevation data for this region. You'll need to download it again to use it offline.",
  "regions_delete_confirm_action": "Delete",
```

**Non-English locale (`app/lib/i18n/app_de.arb:410-419`)** — proves these are fully translated, and shows the placeholder block is repeated per locale:

```json
  "regions_delete_confirm_title": "{name} löschen?",
  "@regions_delete_confirm_title": {
    "placeholders": { "name": { "type": "String" } }
  },
  "regions_delete_confirm_body": "Dies entfernt die heruntergeladene Karte und die Höhendaten für diese Region. Sie müssen sie erneut herunterladen, um sie offline zu nutzen.",
  "regions_delete_confirm_action": "Löschen",
```

**Keys already translated in all 14 locales — reuse, do not re-mint (D-06):**
- `remove` — `app_en.arb:361` `"Remove"` / `app_de.arb:324` `"Entfernen"`
- `available_offline` — `app_en.arb:371` `"Available offline"` / `app_de.arb:334` `"Offline verfügbar"`
- Both are absent from `app/lib/i18n/untranslated_messages.json`.

**English-only precedent for the two NEW strings (confirm body, D-04; edit refusal, D-17):**
`delete_needs_connection` (`app_en.arb:71`, `"This trail is already on the server. Connect to the internet to delete it."`) appears in the `untranslated_messages.json` array for all 13 non-English locales. Any new key must be added to each of those 13 arrays **and** to `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md`.

**l10n regeneration:** `app/l10n.yaml` sets `untranslated-messages-file: lib/i18n/untranslated_messages.json` and its own comment states the workflow: *"Regenerate with `flutter gen-l10n` after touching any .arb file."* The file is committed on purpose so growth shows in a diff.

---

### `app/lib/components/trail/trail_dropdown.dart` — menu item shape (D-08)

**Analog:** the file's own download item, `trail_dropdown.dart:169-202`. This is the canonical `PopupMenuItem` + `ListTile` + `enabled`/`onTap`-null idiom **and** the in-progress spinner treatment *Update* must reuse:

```dart
        if (!isUnsynced) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.download,
            onTap: downloadEnabled
                ? () => ref
                      .read(downloadingTrailIdsProvider.notifier)
                      .download(trail)
                : null,
            enabled: downloadEnabled,
            child: ListTile(
              leading: isDownloading
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : FaIcon(
                      widget.availableOffline
                          ? FontAwesomeIcons.circleCheck
                          : FontAwesomeIcons.download,
                      size: 18,
                      color: widget.availableOffline ? Colors.green : null,
                    ),
              title: Text(
                widget.availableOffline ? 'Available offline' : l18n.download,
                //                         ^^^^^^^^^^^^^^^^^ D-07: hardcoded
                //                         English literal to replace with
                //                         l18n.available_offline
                style: widget.availableOffline
                    ? const TextStyle(color: Colors.green)
                    : null,
              ),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
```

Idioms to preserve verbatim in the new *Update* / *Remove download* items:
- `onTap: <cond> ? () => ... : null` **paired with** `enabled: <cond>` — both, always. `enabled` alone still fires `onTap`.
- `contentPadding: EdgeInsets.zero` on every `ListTile`.
- `FaIcon(..., size: 18)`.
- Disabled colouring is `Colors.grey` on both `leading` and `title` (`:96-110` for Show on map, `:209-220` for the drain-blocked Delete).
- Conditional groups are `if (cond) ...[ const PopupMenuDivider(), PopupMenuItem(...) ]` — the divider lives inside the spread. D-08's "one divider or two" discretion is a choice about how many of these spreads to use.
- `TrailAction` enum at `:28` — `enum TrailAction { open, directions, download, edit, delete }`. New items need new values (e.g. `update`, `removeDownload`).

**Toast-based refusal (D-15/D-17)** — `trail_dropdown.dart:381-391` is the closest precedent (a `needsConnection` refusal), and `trail_create_screen.dart:736-744` is the `trail_uploaded_reopen_to_edit` one:

```dart
          ref
              .read(toastProvider.notifier)
              .add(
                ToastMessage(
                  type: ToastType.warning,
                  icon: FontAwesomeIcons.circleExclamation,
                  text: l18n.delete_needs_connection,
                ),
              );
```

`ToastType.warning` + `circleExclamation` for a refusal; `ToastType.error` + `FontAwesomeIcons.xmark` for a genuine failure (`:396-401`). D-15's Edit-fetch-failed toast is arguably a failure, not a refusal — but D-17 models the *string* on `delete_needs_connection`, whose call site uses `warning`.

**Authorship gate (D-01/D-02)** — reuse `_canEditTrail` / `_allowDelete` at `:227-249`. `_allowDelete`'s `if (widget.trail.isLocal) return true;` at `:242-243` is exactly the `isLocal` branch D-01 forbids and must go; the ownership check below it (`trail.author == user.actorId`) is what stays.

---

### `app/lib/components/trail/trail_panel.dart:214` — the Offline pill re-gate (D-10)

**Current (defective) gate:**
```dart
                      if (trail.isLocal && !isUnsyncedState(trail.syncState)) ...[
```

**Analog for the replacement signal:** `app/lib/routes/trail_detail_screen.dart:133-135` — library membership, read from the provider, not from the model:

```dart
    final availableOffline = ref
        .watch(trailLibraryProvider)
        .any((t) => t.id.isNotEmpty && t.id == trail.id);
```

Note the `t.id.isNotEmpty` guard: D-06 blanks a local-sentinel id, so without it every unsynced trail would match every other unsynced trail. Whichever route CONTEXT's discretion bullet takes (forward `availableOffline` into `TrailPanel` like `TrailDropdown` already gets it at `trail_detail_screen.dart:169-173`, versus re-deriving inside the panel), keep that guard.

`TrailPanel` is not a `ConsumerWidget`-free widget — check its class declaration before assuming `ref` is in scope; it already takes `forceOffline` as a constructor field (`trail_panel.dart:34,:42`) that this phase deletes, so the constructor is being edited anyway.

---

### `app/lib/provider/trail/trail_provider.dart` — retire `forceOffline` (D-20) + opportunistic refresh (D-14)

**Current signature (`:26-31`)** — everything in this block goes, along with the 10-line doc comment at `:16-25`:

```dart
  @override
  FutureOr<Trail> build(String id, {bool forceOffline = false}) async {
    if (forceOffline) {
      final cached = _readCached(id);
      if (cached != null) return cached;
    }
```
becomes `FutureOr<Trail> build(String id) async {`. The `catch (_)` fallback at `:76-80` stays untouched — that is D-22's "disk is the offline fallback".

**Where the D-14 refresh goes:** immediately before `return trail;` at `:75`, after the GPX has been folded into `trail.expand` at `:67-72`. Both the record and `gpxData` are in hand there at zero extra cost (D-23).

**Analog for the write itself:** `app/lib/store/local_trail_store.dart:557-589`, `applyNetworkEditToLocalRow`. Same discipline — query, mutate the **found** entity in place, put; never `TrailEntity.fromModel`:

```dart
void applyNetworkEditToLocalRow(
  Store store, {
  required String localId,
  required String accountId,
  required Trail trail,
}) {
  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox
        .query(
          TrailEntity_.localId.equals(localId) &
              TrailEntity_.owner.equals(accountId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.name = trail.name;
    entity.location = trail.location;
    entity.date = trail.date;
    entity.public = trail.public;
    entity.completed = trail.completed;
    entity.difficulty = trail.difficulty;
    entity.description = trail.description;
    entity.categoryRecordId = trail.category;
    entity.subcategoryRecordId = trail.subcategory;
    entity.tagsJson = encodeTrailTags(trail.expand?.tags);
    entity.updated = trail.updated;

    trailBox.put(entity);
  });
}
```

Its doc comment at `:543-556` enumerates exactly which columns it writes and — more usefully for this phase — which it **deliberately never touches**: `id`, `owner`, `localId`, `syncState`, `syncAttempts`, `syncNextAttemptAt`, `savedByUserIds`, `photos`, `localPhotos`, `gpxData`, `waypoints`. D-14 adds `gpxData` (and waypoints, if the planner takes them) to the written set for the *downloaded-row* variant — which is a **different function with a different key**: the D-14 refresh must query on `TrailEntity_.id.equals(id) & TrailEntity_.savedByUserIds.containsElement(userId)` (library membership), not `localId & owner` (local capture). `TrailNotifier._readCached` at `:89-101` already builds exactly that query:

```dart
    final store = ref.read(objectBoxProvider);
    final userId = currentAccountId(store);
    if (userId == null) return null;

    final box = store.box<TrailEntity>();
    final query = box
        .query(
          TrailEntity_.id.equals(id) &
              TrailEntity_.savedByUserIds.containsElement(userId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
```

**Carry-forward discipline (why blind-put is banned)** — `app/lib/services/trail_download_service.dart:187-218`:

```dart
    _store.runInTransaction(TxMode.write, () {
      // Carry the existing library membership across a re-download. `id` is
      // `@Unique(onConflict: replace)` and `entity` is a FRESH row built by
      // `fromModel`, so putting it blind would wipe `savedByUserIds` -- and
      // with it every other account's claim on this trail. ...
      final query = box.query(TrailEntity_.id.equals(trailId)).build();
      final existing = query.findFirst();
      query.close();

      entity.savedByUserIds = libraryMembersAfterSave(
        existing?.savedByUserIds ?? const [],
        savedByUserId,
      );
      entity.owner = existing?.owner;
      entity.localId = existing?.localId;
      entity.syncState = existing?.syncState ?? entity.syncState;
      entity.syncAttempts = existing?.syncAttempts ?? 0;
      entity.syncNextAttemptAt = existing?.syncNextAttemptAt;
      entity.localPhotos = existing?.localPhotos ?? const [];

      box.put(entity);
    });
```

Six fields carried forward + `savedByUserIds`. This is also D-12's evidence that *Update* needs no new download path.

**Failure handling for the D-14 write:** it must never turn a successful fetch into an error. Copy `trail_create_screen.dart:768-792`'s best-effort-and-log posture (below), not a rethrow. `_readCached`'s own `catch (e, st) { debugPrint(...); return null; }` at `:111-118` is the in-file precedent for the same posture.

---

### `app/lib/provider/trail/trail_provider.g.dart` — regeneration target

**Command:** `dart run build_runner build --delete-conflicting-outputs` from `app/`.

**Current generated family (`trail_provider.g.dart:73-77, 83-105`)** — the record-typed argument `(String, {bool forceOffline})` is what CONTEXT's D-21 means by "part of the family key":

```dart
  TrailNotifierProvider call(String id, {bool forceOffline = false}) =>
      TrailNotifierProvider._(
        argument: (id, forceOffline: forceOffline),
        from: this,
      );

abstract class _$TrailNotifier extends $AsyncNotifier<Trail> {
  late final _$args = ref.$arg as (String, {bool forceOffline});
  String get id => _$args.$1;
  bool get forceOffline => _$args.forceOffline;

  FutureOr<Trail> build(String id, {bool forceOffline = false});
```

**Target shape after regeneration** — a single positional arg collapses the record to a bare `String`. `app/lib/provider/trail/local_trail_provider.g.dart` shows exactly that (it is a functional provider, not a class notifier, but the argument handling is the point):

```dart
final class LocalTrailFamily extends $Family
    with $FunctionalFamilyOverride<Trail?, String> {
  ...
  LocalTrailProvider call(String localId) =>
      LocalTrailProvider._(argument: localId, from: this);
}
```

Do not hand-edit the `.g.dart`; the `_$trailNotifierHash()` constant (`:53`) changes with the source and only build_runner computes it correctly.

**Every call site that must lose the argument** (verified by grep, 2026-08-04):

| File:line | What |
|---|---|
| `lib/provider/trail/trail_provider.dart:16-31` | doc comment, signature, early-return block |
| `lib/components/trail/like_button.dart:12,:21,:39` | field, ctor param, `trailProvider(trail.id, forceOffline: forceOffline)` |
| `lib/components/trail/trail_dropdown.dart:34-44,:76-79,:150-156` | doc, field, ctor, `trailMapLocation(...)`, `ref.invalidate(trailProvider(...))` |
| `lib/components/trail/trail_panel.dart:34,:42,:70` | ctor param, field, forwarding |
| `lib/routes/trail_detail_screen.dart:30,:36,:115,:163,:166-173` | field, ctor, `ref.watch`, LikeButton, TrailDropdown |
| `lib/routes/library_screen.dart:120-128,:148-149` | two `trailDetailLocation(trail, forceOffline: true)` + their comments |
| `lib/provider/router_provider.dart:385-392,:399-402` | `?offline=1` parsing, two builders |
| `lib/util/trail/route_location.dart:24-26,:27,:34,:39,:41-48` | doc, both signatures, both bodies |
| `test/util/trail/route_location_test.dart:38-73` | two tests to delete |

Also grep for `TrailDetailMapScreen`'s own `forceOffline` field — `router_provider.dart:399-402` passes it, so the screen declares it.

---

### `app/lib/provider/router_provider.dart:381-404` — remove `?offline=1`

**Analog:** the `/trail/local/:localId` route immediately above (`:370-378`) — a builder that reads only path parameters:

```dart
              builder: (context, state) {
                final localId = state.pathParameters['localId']!;
                return TrailDetailMapScreen(id: '', localId: localId);
              },
```

Target: `return TrailDetailScreen(id: trailId);` and `return TrailDetailMapScreen(id: trailId);`, with the `:385-388` comment about surviving deep links deleted along with the parameter.

---

### `app/lib/util/trail/route_location.dart` — remove the options (D-20)

Both functions revert to unnamed-parameter form. Delete the `[forceOffline]` paragraph from `trailDetailLocation`'s doc (`:24-26`) and the "Built from the UN-flagged base" comment (`:41-43`) — the latter only exists to explain query-string placement. What must survive untouched: the file-header contract (`:1-10`), the unsynced `/trail/local/<localId>` branch, and the null-means-not-addressable rule.

Target bodies:
```dart
  if (trail.id.isEmpty) return null;
  return '/trail/${trail.id}';
```
```dart
  final base = trailDetailLocation(trail);
  if (base == null) return null;
  return '$base/map';
```

---

### `app/lib/routes/trail_create_screen.dart:768-792` — widen the reconciliation gate (D-13)

**Analog is the block itself.** The gate `if (reconcileLocalId != null)` is the too-narrow part; the try/catch/debugPrint posture inside is exactly right and must be preserved for the widened case:

```dart
      if (reconcileLocalId != null) {
        // The server write already succeeded, so a failure reconciling the
        // LOCAL row must never be surfaced as a failed save -- that would
        // misreport an edit the server already accepted. Best-effort,
        // logged: ...
        try {
          final reconcileStore = ref.read(objectBoxProvider);
          final accountId = currentAccountId(reconcileStore);
          if (accountId != null) {
            applyNetworkEditToLocalRow(
              reconcileStore,
              localId: reconcileLocalId,
              accountId: accountId,
              trail: result.trail,
            );
          }
        } catch (e) {
          debugPrint(
            'trail_create_screen: applyNetworkEditToLocalRow failed, '
            'reconcileLocalId: "$reconcileLocalId": $e',
          );
        }
      }

      // Must run AFTER the reconciliation above, never before -- otherwise
      // the own-trails list re-reads the local row before the edit lands on
      // it, reproducing CR-03 under a green success toast.
      _invalidateOwnTrailsList();
```

Three constraints on any edit here:
1. **Ordering is asserted by a test.** `test/routes/trail_create_screen_local_save_gate_test.dart:849-864` greps this file's source text for the index of `applyNetworkEditToLocalRow(` relative to `_invalidateOwnTrailsList();`. Moving the block past the invalidate breaks that test by design.
2. `currentAccountId(reconcileStore)` is read **fresh**, never cached — `local_trail_store.dart:536-541` (T-36-17-01/D-13) states account B's edit could otherwise be written onto account A's row.
3. The widened branch is a *different query* (`id` + `savedByUserIds`), so it is a second call to a second function, not a loosened `reconcileLocalId != null`. `applyNetworkEditToLocalRow`'s own key is `localId & owner` and a downloaded row has `localId == null`.

---

## Shared Patterns

### Riverpod codegen
**Source:** every `lib/provider/**/*.dart` with `part '*.g.dart';`
**Apply to:** `trail_provider.dart`
```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';
part 'trail_provider.g.dart';

@riverpod
class TrailNotifier extends _$TrailNotifier {
  @override
  FutureOr<Trail> build(String id) async { ... }
}
```
Regenerate with `dart run build_runner build --delete-conflicting-outputs`. Both `.dart` and `.g.dart` are committed.

### ObjectBox write transaction
**Source:** `lib/store/local_trail_store.dart:563-588`, `lib/services/trail_download_service.dart:187-218`
**Apply to:** the D-14 refresh
```
store.runInTransaction(TxMode.write, () {
  final box = store.box<TrailEntity>();
  final query = box.query(<account-scoped predicate>).build();
  final entity = query.findFirst();
  query.close();               // ALWAYS, immediately after the read
  if (entity == null) return;  // silent no-op, never an error
  ... mutate the FOUND entity in place ...
  box.put(entity);
});
```
Never `TrailEntity.fromModel` into an existing row; never `put` a fresh entity without carrying `savedByUserIds`/`owner`/`localId`/`syncState`/`syncAttempts`/`syncNextAttemptAt`/`localPhotos` forward.

### Account scoping
**Source:** `lib/store/current_account.dart` → `currentAccountId(store)`
**Apply to:** every ObjectBox read/write in this phase
Read fresh at the call site. Library reads filter `TrailEntity_.savedByUserIds.containsElement(userId)`; local-capture reads filter `TrailEntity_.owner.equals(accountId)`. Memory rule "scope, don't delete user data" applies — *Remove download* must drop this account's membership, not purge the row for everyone. See `libraryMembersAfterSave` and `TrailLibraryNotifier.deleteTrail`.

### Toast
**Source:** `lib/provider/toast_provider.dart`; call sites `trail_dropdown.dart:373-379, 382-390, 393-401`
**Apply to:** D-15/D-17 edit refusal
`ref.read(toastProvider.notifier).add(ToastMessage(type:, icon:, text:))`. `warning` + `circleExclamation` for refusals, `error` + `xmark` for failures.

### Async + BuildContext
**Source:** `trail_dropdown.dart:342-345`, `:436-440`; `settings_offline_regions_screen.dart:1021,:1044`
**Apply to:** the confirm dialog and the Edit-fetches-server-copy flow
Resolve `AppLocalizations.of(context)!` and `GoRouter.of(context)` **before** the first await. Use `mounted` for `State.context`; use `context.mounted` when `context` arrived as a parameter.

### Destructive-action l10n
**Source:** `app/l10n.yaml`, `lib/i18n/untranslated_messages.json`, `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md`
**Apply to:** the two new strings
Add to `app_en.arb`, run `flutter gen-l10n`, commit the resulting `untranslated_messages.json` growth, and append to the pending todo's backlog. Machine translation is explicitly rejected for irreversible actions.

---

## Testing Patterns

**Hard constraint:** the host `flutter test` environment cannot open a real ObjectBox `Store` — `libobjectbox.dylib` is absent, which is why no existing unit test constructs one. Anything asserting on library membership or stored-row reconciliation must go through a seam that avoids a live `Store`, or be verified on device.

### Widget test with provider overrides (no Store)
**Source:** `app/test/components/trail/trail_dropdown_menu_test.dart`
**Apply to:** all D-04…D-09, D-15…D-17 menu assertions

This is the seam. It mounts the real `TrailDropdown` in an `AppBar`, overrides every provider it touches with a stub notifier, and asserts on rendered `PopupMenuItem`s — no `Store`, no network, no source-text grepping.

```dart
Widget _harness(Trail trail, {Set<String> inFlight = const {}, Set<String> downloading = const {}}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith(_StubAuth.new),
      trailSyncProvider.overrideWith(() => _StubTrailSync(inFlight)),
      downloadingTrailIdsProvider.overrideWith(() => _StubDownloadingTrailIds(downloading)),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('en'),
      home: Scaffold(
        appBar: AppBar(actions: [TrailDropdown(trail: trail, availableOffline: false)]),
        body: Consumer(builder: (context, ref, _) { ref.watch(authProvider); return const SizedBox.shrink(); }),
      ),
    ),
  );
}

Future<void> _openMenu(WidgetTester tester) async {
  await tester.pump();  // let the async _StubAuth.build() resolve
  await tester.tap(find.byType(PopupMenuButton<TrailAction>));
  await tester.pumpAndSettle();
}
```

Three non-obvious details this harness encodes, all of which the new tests inherit:
- The inert `Consumer` in `body` exists **only** to prime `authProvider`. `@riverpod` providers build lazily, so without it `_canEditTrail`/`_allowDelete` read `AsyncLoading` on the first menu open and Edit/Delete render as absent. Documented at `:103-115`.
- `await tester.pump()` before the tap, for the same reason.
- Assertions use `find.text('Download')` and `tester.widget<PopupMenuItem<TrailAction>>(find.widgetWithText(PopupMenuItem<TrailAction>, 'Delete')).enabled` — literal English strings, because `locale: const Locale('en')` is pinned.

**Direct implication for this phase:** `availableOffline` reaches `TrailDropdown` as a plain constructor bool, so *both* menu branches (downloaded / not) are testable through this harness with **zero** Store involvement. If the planner instead makes the dropdown read `trailLibraryProvider` itself, add a stub `TrailLibrary` notifier override alongside the three existing stubs — same shape as `_StubTrailSync`. Do not let the membership signal become a direct `objectBoxProvider` read inside the widget; that would push it out of reach of every existing test.

**Existing tests this phase will break and must update:**
- `:171-184` and `:179` assert `find.text('Available offline')` — D-07/D-08 replace that item with Update + Remove download.
- `:198-241`, `:270-296` assert delete confirm copy — unaffected by D-04 (which adds a *separate* dialog) but re-read them before touching `_confirmDelete`.

### Pure-function test
**Source:** `app/test/util/trail/route_location_test.dart` — "No ProviderScope, no Store -- plain `Trail.empty().copyWith(...)` fixtures only."
**Apply to:** `route_location.dart`
D-20 deletes exactly two tests: `'forceOffline appends ?offline=1 to the FULL path, once'` (`:38-54`) and `'forceOffline is a no-op for an unsynced trail'` (`:56-73`). The remaining six — including `'no returned location ever contains a double slash'` (`:120-164`) — stay and must still pass.

### Source-text assertion (use sparingly)
**Source:** `app/test/routes/trail_create_screen_local_save_gate_test.dart:849-864`
Greps `trail_create_screen.dart`'s source for `applyNetworkEditToLocalRow(` vs `_invalidateOwnTrailsList();` ordering. The header of `trail_dropdown_menu_test.dart` explains why this style is a last resort: `trail_dropdown_delete_gate_test.dart` stayed green for a whole phase over a menu that could not be opened in the running app. Use it only for ordering invariants no behavioural test can reach — and D-13's widening touches exactly that block, so expect to update it.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| — | — | — | Every file this phase touches already exists; no new file needs an analog. |

The nearest thing to a gap is **D-14's opportunistic stored-row refresh inside a provider's `build()`**: no existing provider writes to ObjectBox from its build path (`_readCached` only reads). The write *mechanics* are fully covered by `applyNetworkEditToLocalRow` and `TrailDownloadService`; what is new is the placement, and the constraint is that it must be non-fatal (best-effort + `debugPrint`, per `trail_create_screen.dart:786-791` and `trail_provider.dart:111-118`).

## Metadata

**Analog search scope:** `app/lib/components/trail/`, `app/lib/provider/trail/`, `app/lib/routes/`, `app/lib/store/`, `app/lib/services/`, `app/lib/util/trail/`, `app/lib/i18n/`, `app/test/`
**Files read:** 16
**Pattern extraction date:** 2026-08-04
