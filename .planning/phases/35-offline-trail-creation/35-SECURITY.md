---
phase: 35-offline-trail-creation
audited: 2026-08-02T00:00:00Z
auditor: Claude (gsd-secure-phase)
scope: "derived from diff — no <threat_model> block exists for this phase (executed without plans, see 35-SUMMARY.md)"
commit_audited: 21c4b1ee
items_reviewed: 6
threats_open: 0
regressions_found: 0
pre_existing_findings: 2
na_rows: 0
---

# Phase 35: Offline Trail Creation — Security Audit

**Scope note.** Phase 35 shipped without `/gsd-plan-phase`, so there is no `<threat_model>` block
to verify against — the usual "declared mitigation present in code" check doesn't apply here.
Instead this audit derives the threat surface directly from `git show 21c4b1ee --stat` and its
hunks, and checks each surface the audit brief named plus one general sweep of the remainder of
the diff. Phase 34's surface (convert endpoint, vendored GPX reader, KMZ handling, Valhalla
proxies) is **not** re-audited — see `../34-dart-conversion-port/34-SECURITY.md`
(`threats_open: 0`, 39/39 declared threats closed).

**Result: 0 regressions introduced by Phase 35.** Every change in this commit that touches
security-relevant behavior (extension gating, error-handling split, route topology, the
provenance-flag rename) preserves or strengthens the prior posture. Two **pre-existing** findings
were surfaced while inspecting the exact lines the audit brief pointed at — neither was introduced
or widened by this phase, both are reported below rather than silently dropped, per the audit
brief's explicit instruction.

## Items reviewed

| # | Surface | File | Verdict |
|---|---------|------|---------|
| 1 | Client-side extension guard before transcode upload | `trail_import_util.dart:74-83` | **VERIFIED** — UX-only, does not weaken server validation |
| 2 | Tag search filter string interpolation | `tag_provider.dart:55` | **PRE-EXISTING FINDING** (not a Phase 35 regression) — see below |
| 3 | Shared profile URL (`@` prefix, QR/share sheet) | `profile_share_screen.dart:24-27` | **VERIFIED**, low residual risk, unchanged posture |
| 4 | `/profile/share` moved out of `ShellRoute` | `router_provider.dart:186-190` | **VERIFIED** — auth guard is global, not shell-scoped |
| 5 | `Trail.isOffline` → `Trail.isLocal` rename, all 9 consumers | `trail.dart`, `trail_dropdown.dart`, `trail_card.dart`, `trail_list_item.dart`, `trail_panel.dart`, `trail_entity.dart`, `global_search_models.dart`, `trail_summary.dart`, `trail_detail_map_screen.dart` | **VERIFIED** rename itself is exact, no inversion — **but surfaced a PRE-EXISTING data-safety bug** in the branch it touches, see below |
| 6 | Remainder of diff (library empty states, i18n extraction, form-dirty reset, `trail_create_screen.dart` map-offline wiring) | `library_screen.dart`, `list_screen.dart`, `trail_source_select_screen.dart`, `trail_create_screen.dart` | **VERIFIED** no security-relevant surface — cosmetic/UX only |

---

### 1. Extension guard (`trail_import_util.dart:74-83`)

```dart
final ext = p.extension(name).replaceFirst('.', '').toLowerCase();
if (ext.isEmpty || !trailImportExtensions.contains(ext)) {
  showError(l10n.trail_source_import_error);
  return;
}
if (ext != 'gpx' && isOffline) {
  showError(l10n.trail_source_offline_import_error);
  return;
}
```

Confirmed **UX control, not a security control**, and it does not weaken `assertParsableGpx`
(Phase 34, `web/src/routes/api/v1/trail/convert/+server.ts:24-39`, CR-04 — `34-SECURITY.md`
verified this at `threats_open: 0`):

- The `.gpx` branch never reaches the server at all — parsed on-device via `parseGpxSafely`
  (docstring at `trail_import_util.dart:36-39`, confirmed unchanged).
- Every non-`.gpx` branch still goes through `transcodeToGpx` → `/trail/convert`, which still runs
  `assertParsableGpx` server-side regardless of what this client-side guard decided. A malicious
  client bypassing this guard entirely (calling `transcodeToGpx` directly with any path/name) gains
  nothing — the server validates by content, not by extension or client cooperation.
- `p.extension()` edge cases tested live (`dart run`, `package:path` 1:1 with the app's pinned
  version):

  | input | `ext` result | outcome |
  |---|---|---|
  | `evil.gpx.kml` | `kml` | correctly classified as kml (last extension wins — matches what the server actually receives as the filename) |
  | `evil.GPX` | `gpx` | case-insensitive, correct |
  | `..%2f` | `%2f` | not in allowlist → rejected (fail-closed) |
  | `..`, `` (empty), `noext` | `` (empty) | rejected via the `ext.isEmpty` branch |
  | `.gpx` (dotfile) | `` (empty) | rejected — not misclassified as gpx |

  No path-traversal risk: `name` is never used for filesystem access — the actual file read uses
  the separate `path` parameter (`File(path).readAsString()` at line 93; `MultipartFile.fromFile(path, filename: name)` at line 225), and `name`'s only two uses are (a) this extension
  parse and (b) an opaque multipart form field / fallback trail name.
- Both entry points exercise the same guard: the picker (`trail_source_select_screen.dart:195`)
  and the OS share-intent handler (`main.dart:185`), confirmed by `grep -rn "importTrailFile("`.

### 3. Shared profile URL (`profile_share_screen.dart:24-27`)

```dart
final serverUrl = userEntity.serverUrl.endsWith('/')
    ? userEntity.serverUrl.substring(0, userEntity.serverUrl.length - 1)
    : userEntity.serverUrl;
final profileUrl = '$serverUrl/profile/@${userEntity.preferredUsername}';
```

Neither `serverUrl` nor `preferredUsername` is percent-encoded before interpolation. Confirmed as
low residual risk, unchanged by this phase (only the literal `@` and a background redesign were
added — the interpolation itself is untouched):

- `serverUrl` is always normalized to an explicit `http://`/`https://` scheme at entry
  (`server_selection_screen.dart:33-34`, `if (!url.startsWith('http://') && !url.startsWith('https://')) url = 'https://$url';`), so no `javascript:`/`intent:` scheme injection via this
  field.
- Both values are always the **current, already-authenticated user's own account data** — this
  screen has no path that renders another user's `serverUrl`/`preferredUsername`. There is no
  cross-user or unauthenticated-attacker angle: a user can only mangle their own share output.
- The interpolation is never re-parsed as a URI by the app itself; it is only handed to
  `QrImageView` (rendered as an image) and `SharePlus.instance.share(ShareParams(text: profileUrl))`
  (opaque text to the OS share sheet). No in-app navigation or code-execution path consumes this
  string.
- Not a Phase 35 regression: the pre-existing line was `'$serverUrl/profile/${username}'` with
  identical lack of encoding; this phase only changed the literal to `.../profile/@${username}`.

### 4. `/profile/share` moved to a top-level route (`router_provider.dart:186-190`)

Confirmed the auth guard is **not** shell-scoped. `GoRouter`'s `redirect` callback
(`router_provider.dart:88-120`) runs for every navigation regardless of nesting, and gates purely
on `state.matchedLocation` against a hardcoded allowlist:

```dart
final authRoutes = ['/login', '/register', '/welcome', '/select-server'];
...
if (!loggedIn) {
  if (isAtSplash || !isAtAuthRoute) return '/welcome';
  return null;
}
```

`/profile/share` is not in `authRoutes`, so an unauthenticated request for it redirects to
`/welcome` exactly as it did when nested inside the `ShellRoute` — `ShellRoute` membership plays no
part in this router's auth logic. This is also not a novel pattern: `/settings` and its subtree
were already top-level (outside the `ShellRoute`) before this phase, using the identical
protection.

### 5. `isOffline` → `isLocal` rename — 9 consumers, all checked

`git show 21c4b1ee` diff confirms every consumer is a mechanical 1:1 token rename with **zero**
surrounding logic change:

- `trail_card.dart:46`, `trail_list_item.dart:36`: `if (trail.isLocal && localPath != null)` — unchanged condition, just renamed.
- `trail_panel.dart:81,117,239`: three occurrences, all straight renames (badge display, TabBar
  gating).
- `trail_entity.dart:177`: constructor field rename (`isOffline: true` → `isLocal: true`), same
  call site (`TrailEntity.toModel()`), same "always true" semantics.
- `global_search_models.dart:80`, `trail_summary.dart:32`: interface/override rename, same `false`
  literal.
- `trail_dropdown.dart:170,204`: both occurrences renamed identically — see finding below.

**The rename question the audit brief asked is answered cleanly: no inversion, no drop, exact
preservation.**

However, reading `_deleteTrail` in full (not just the two renamed lines) to answer that question
surfaced a genuine, load-bearing, **pre-existing** bug — reported below rather than left invisible,
per the audit brief's explicit instruction not to let "pre-existing, not our problem" hide a real
issue.

---

## Pre-existing findings surfaced by this audit

These predate Phase 35 (confirmed via `git log -p --follow` on each file — the vulnerable lines
were introduced in earlier commits and Phase 35's diff does not touch the surrounding logic).
Neither counts toward `threats_open` for this phase. Both are reported in full because the audit
brief explicitly asked not to let "pre-existing" make a real issue invisible.

### A. `trail_dropdown.dart:203-207` — "un-download" falls through to a real server DELETE

```dart
Future<void> _deleteTrail(BuildContext context, Trail trail) async {
    if (trail.isLocal) {
      Navigator.of(context).pop();
      ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
    }                                              // <-- no return / no else

    final router = GoRouter.of(context);

    try {
      await ref.read(trailSaveProvider.notifier).deleteTrail(trail);   // unconditional
      ...
```

`trailSaveProvider.deleteTrail` (`trail_save_provider.dart:198-200`) is an unconditional
`DELETE /trail/{id}` HTTP call — it has no internal `isLocal` guard. The `if (trail.isLocal)` block
has no `return`/`else`, so **every** call to `_deleteTrail` falls through to this real server
delete, regardless of the `isLocal` branch above it. This contradicts the commit's own stated
intent ("load bearing — routing delete to `trailLibraryProvider.deleteTrail` (un-download) *rather
than* a server delete") — the code does both, not one or the other.

**Reachability is real, not theoretical.** `TrailDropdown` is only instantiated from
`trail_detail_screen.dart:98`, fed by `trailProvider(id)`. `TrailNotifier.build()`
(`trail_provider.dart:17-98`) falls back to the on-device ObjectBox cache (`entity.toModel()`,
which hardcodes `isLocal: true`) inside a bare `catch (_)` on **any** exception from the network
fetch — not gated on `onlineStatusProvider`/connectivity. A transient error while the device is
fully online and connected (a dropped mid-request connection, a momentary 5xx, a slow GPX file
fetch) is enough to route `TrailDetailScreen` onto the cached, `isLocal: true` copy while the
device can still reach the server for a subsequent request.

**Concrete impact:** a user viewing their own downloaded trail, in a state where the device is
online but the earlier detail fetch happened to fail and fell back to cache, taps Delete (shown
unconditionally for any `isLocal` trail — `_allowDelete` returns `true` before even checking
ownership, `trail_dropdown.dart:170-172`) intending to free device storage. The un-download step
succeeds, giving the appearance of the intended outcome, but the code then also fires a live,
authenticated `DELETE /trail/{id}` against the server, which — for a trail the user actually owns —
succeeds and permanently deletes it server-side. This matches exactly the failure mode the audit
brief named ("a mistake here deletes a user's trail from the server when they meant to remove a
download") and the standing project rule to never purge server/account data via a
scoping/local-cleanup action.

Severity: **HIGH** (irreversible data loss for the trail's actual owner), but **not a Phase 35
regression** — `git log -p --follow -- app/lib/components/trail/trail_dropdown.dart` shows both
the missing `return` and the unconditional `trailSaveProvider.deleteTrail` call predate the
`isOffline`→`isLocal` rename by several commits; Phase 35 changed only the two identifier tokens.
Recommend an immediate follow-up fix (add `return;` after the `trailLibraryProvider.deleteTrail`
call, or restructure as `if (trail.isLocal) { ...; return; }`) — not made here, as implementation
files are read-only for this audit.

### B. `tag_provider.dart:55` — unescaped user input in a PocketBase filter string

```dart
final response = await api.get("/tag?filter=name~'$name'");
```

The audit brief asked this to be audited properly as the one genuine injection surface in the
diff — it is, though (per the brief's own hint) the line predates Phase 35 (`git log --oneline --
app/lib/provider/trail/tag_provider.dart` → introduced in `340e239c "adds tag filtering"`; Phase 35
only changed the surrounding error-handling, confirmed via the `21c4b1ee` diff hunk which shows
the `api.get(...)` line unchanged/re-indented, not rewritten).

**No encoding happens anywhere in the path:**
- Client: Dio's `RequestOptions.uri` getter (`dio-5.9.2/lib/src/options.dart:628-643`) concatenates
  `baseUrl + path` as a raw string and only calls `Uri.parse(url).normalizePath()` — verified live
  (`dart run` against the pinned `dio` version) that `'`, `&`, `?`, space and non-ASCII all survive
  into the request either literally or percent-encoded *as opaque bytes*, but `Uri.parse` does
  **not** re-interpret or reject a `name` value that breaks the intended `filter=name~'...'`
  structure — e.g. `name = "x'&sort=-created&perPage=1000000"` produces a wire request whose query
  string, once split on `&` by the server's own URL parsing, contains extra `sort`/`perPage`
  key-value pairs alongside the truncated `filter`.
- Server: `RecordListOptionsSchema.filter` (`web/src/lib/models/api/base_schema.ts:14`) is
  `z.string().optional()` — no character allowlist, no length cap — and is forwarded verbatim to
  PocketBase (`web/src/lib/util/api_util.ts:51-68`, `event.locals.pb.collection('tags').getFullList({filter: ...})`).
- PocketBase itself (vendored at `pocketbase@v0.38.0`, confirmed by reading
  `apis/record_crud.go:56-121`) documents this class of risk explicitly in its own source
  comments: the client-supplied `filter` executes in the same query as the collection's `ListRule`
  ("the List API rule acts also as filter and executes in a single run with the client-side
  filters. This is by design"), and its `RecordFieldResolver`'s allowed-fields pattern
  (`core/record_field_resolver.go:105`, `` `^\@collection\.\w+(\:\w+)?\.[\w\.\:]*\w+$` ``) permits
  `@collection.<other>.<field>` joins from the same client-supplied filter text — not just from
  admin-authored rule strings. PocketBase's own mitigation is a randomized throttle on repeated
  empty-result filters (lines 104-121), explicitly called "not a full guarantee."

**Impact, scoped to this endpoint:** low in practice. `tags` has `listRule: "@request.auth.id != \"\""`
(`db/migrations/1742411270_updated_tags.go`) — any authenticated user can already list every tag
unfiltered, and the collection carries no privacy-sensitive field (`id`, `name`, `created`,
`updated` only per `db/migrations/1742409454_created_tags.go`), so narrowing/broadening the `name~`
match via injection discloses nothing beyond what's already fully enumerable. The `&`
parameter-splice at most lets the same authenticated user set `perPage`/`sort`/`expand` on their
own request — not a privilege boundary, since they could call `/tag` directly with those params
themselves. The theoretically more serious angle — using an injected `@collection.<other>.<field>`
clause as a boolean oracle against a collection the caller does *not* otherwise have visibility
into (trails, summit_logs, users) — is real as a **class** of risk in this codebase's pattern (the
same raw-interpolation shape recurs, out of this phase's diff, at
`app/lib/provider/profile/profile_counts_provider.dart:16-17`), but was not further exploited or
proven beyond PocketBase's own documented, throttle-mitigated acknowledgment of the risk, since
doing so would require probing a running server, which is out of scope for a static code audit.

**`debugPrint` check (explicitly requested):** `debugPrint('tag search failed for "$name": $e')`
(`tag_provider.dart:70,74`) logs only the user-typed search string and `DioException.toString()`,
whose format (`dio-5.9.2/lib/src/dio_exception.dart:237-289`) is a generic, templated
human-readable description of the failure type/status code — no headers, cookies, or auth tokens
are included. Confirmed not a secrets leak, consistent with the existing WR-12 logging pattern
already accepted in `trail_import_util.dart` (Phase 34).

**Verdict: not a Phase 35 regression.** Not counted in `threats_open`. Flagged here per the audit
brief's explicit instruction, since it is a real (if PocketBase-acknowledged, low-severity-for-this-endpoint)
input-validation gap that a future endpoint reusing the same raw-interpolation pattern against a
privacy-sensitive collection would inherit. Recommend using PocketBase's parameterized filter
builder (`pb.filter('name ~ {:name}', {'name': name})`-equivalent, or a Dio `queryParameters` map
so `Uri` encodes `name` as a single opaque value) at this call site and at
`profile_counts_provider.dart:16-17`, and adding a server-side character allowlist to
`RecordListOptionsSchema.filter` if it must remain a free-form string.

---

## Conclusion

**threats_open: 0** for Phase 35 itself — every change this phase made to previously-existing
security-relevant logic (the extension guard, the route topology, the flag rename across 9
consumers) either strengthens or exactly preserves the prior posture; no mitigation was weakened
and no new unmitigated attack surface was introduced by this diff.

Two pre-existing findings were surfaced by close reading of the exact lines the audit brief
pointed at and are reported in full above rather than dismissed as out of scope:

- **Finding A** (`trail_dropdown.dart:203-207`, HIGH): the "un-download" delete path falls through
  to an unconditional server-side delete. Real, reachable, potentially destroys a user's own trail
  data. Predates Phase 35. Recommend prompt follow-up fix.
- **Finding B** (`tag_provider.dart:55`, LOW for this endpoint / informational as a pattern):
  unescaped user input in a PocketBase filter string, no encoding at any layer. Predates Phase 35
  (line unchanged by this commit). Low impact on the `tags` collection specifically, but the
  raw-interpolation pattern recurs elsewhere in the codebase and should not be copied into a
  privacy-sensitive collection's filter construction.

No `unregistered_flag` items — this phase's `SUMMARY.md` has no `## Threat Flags` section to
reconcile (the phase was executed without a plan, so there is no executor threat-flag mechanism to
check against).
