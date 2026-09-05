---
phase: quick-260808-vct
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/models/trail.dart
  - app/lib/util/trail/filter.dart
  - app/lib/provider/trail/trail_library_provider.dart
  - app/lib/i18n/app_en.arb
  - app/lib/components/trail/trail_quick_filter_bar.dart
  - app/lib/routes/library_screen.dart
  - app/lib/routes/profile_trail_map_screen.dart
  - app/lib/provider/profile/profile_trails_provider.dart
  - app/test/util/trail/filter_test.dart
autonomous: false
requirements: [QUICK-260808-vct]

must_haves:
  truths:
    - "An 'Available offline' chip renders as the second chip in the quick filter bar, immediately right of 'Sort'"
    - "Tapping the chip toggles it on/off and it renders in the active (primaryContainer) style while on, using the same buildChip helper as every other chip"
    - "With the chip on, the profile trails list shows only trails whose data is already on this device"
    - "With the chip on, the profile trails list stops paginating the server list (no infinite-scroll spinner, no partial pages)"
    - "The chip's label comes from AppLocalizations, not a hardcoded English string"
    - "Reset clears the chip along with every other filter"
    - "The offline chip is never rendered on the profile trail MAP screen, where server-side clustering cannot express device-local library membership"
    - "A signed-out account, or another account's downloaded trails, are never surfaced by the filter"
  artifacts:
    - path: "app/lib/models/trail.dart"
      provides: "TrailFilter.offlineOnly flag, deliberately excluded from toFilterText()"
      contains: "offlineOnly"
    - path: "app/lib/provider/trail/trail_library_provider.dart"
      provides: "downloadedTrailIdsProvider — the single named id-set view of the account-scoped offline library"
      contains: "downloadedTrailIds"
    - path: "app/lib/util/trail/filter.dart"
      provides: "offlineOnly clause in applyTrailFilter, via an injected downloadedIds set"
      contains: "downloadedIds"
    - path: "app/lib/components/trail/trail_quick_filter_bar.dart"
      provides: "Offline toggle chip in second position + showOfflineChip opt-out"
      contains: "showOfflineChip"
    - path: "app/lib/i18n/app_en.arb"
      provides: "available_offline localized label"
      contains: "available_offline"
    - path: "app/test/util/trail/filter_test.dart"
      provides: "Unit coverage for the offlineOnly clause"
      contains: "offlineOnly"
  key_links:
    - from: "app/lib/components/trail/trail_quick_filter_bar.dart"
      to: "trailFilterProvider(filterId).notifier"
      via: "updateFilter((f) => f.copyWith(offlineOnly: !f.offlineOnly))"
      pattern: "offlineOnly"
    - from: "app/lib/provider/profile/profile_trails_provider.dart"
      to: "trailLibraryProvider"
      via: "device-only list source when offlineOnly is on"
      pattern: "trailLibraryProvider"
    - from: "app/lib/provider/trail/trail_library_provider.dart"
      to: "currentAccountId(store)"
      via: "trailLibraryProvider's existing account scoping (inherited, not re-implemented)"
      pattern: "currentAccountId"
---

> **SUPERSEDED — read SUMMARY.md for what actually shipped.**
>
> This plan's client-side design (source inversion in `ProfileTrailsNotifier`, `totalPages`
> pinned to 1, map screen opted out, `offlineOnly` never emitted into `toFilterText()`) was
> built, reviewed and then replaced before anything was pushed. The delivered implementation
> filters SERVER-side with an `id IN [...]` whitelist.
>
> What this plan got wrong:
> - `id` is a filterable attribute on the trails index (`db/main.go:413`) and both search
>   endpoints are POST, so a server-side id whitelist was available the whole time.
> - The map opt-out was presented as a system constraint. It was a consequence of choosing
>   client-side filtering; with the clause evaluated server-side, clusters are aggregated over
>   the already-filtered set and the map works.
> - Pinning `totalPages` abandoned pagination and downgraded text search to a local substring
>   match — both avoided entirely by filtering server-side.
> - The offline resilience it was optimizing for was illusory: the network fetch runs first
>   either way, and a non-own handle rethrows before any device-only path is reached.
>
> The threat model below is still broadly valid, but T-vct-03 is inverted — the id set IS now
> sent to the server, deliberately, narrowed to the profile's author because
> `/profile/{handle}/trails` proxies the request body to the origin instance for a federated
> actor. See SUMMARY.md's Security table.

<objective>
Add an "Available offline" toggle chip to the trail quick filter bar as the second chip
(immediately right of "Sort"), filtering trail lists down to trails whose data is already
held on this device.

Purpose: A hiker at a trailhead with no signal wants to see, at a glance, which of the
trails in front of them will actually open. Today that answer only exists on the Library
tab, which is a separate destination and loses the profile list's search/sort context.

Output: `TrailFilter.offlineOnly` flag, a `downloadedTrailIdsProvider` view over the
existing account-scoped offline library, the chip itself, and device-only list sourcing
in `ProfileTrailsNotifier`.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md

@app/lib/components/trail/trail_quick_filter_bar.dart
@app/lib/util/trail/filter.dart
@app/lib/provider/trail/trail_library_provider.dart
@app/lib/provider/profile/profile_trails_provider.dart
@app/lib/routes/library_screen.dart
</context>

<discovery_findings>
Investigation results the executor should NOT re-derive:

**Source of truth for "offline available"** — `TrailEntity.savedByUserIds` (a `List<String>`
of account ids holding the trail in their offline library), read through
`trailLibraryProvider` in `app/lib/provider/trail/trail_library_provider.dart`. That provider
already queries `savedByUserIds.containsElement(currentAccountId(store))` and already returns
`const []` for a signed-out account. **Reuse it. Do not query `TrailEntity` directly, and do
not invent a new marker.**

**Do NOT use `Trail.isLocal` as the primary signal.** Its doc comment in
`app/lib/models/trail.dart` explicitly forbids gating badges/visibility on it, because
`TrailEntity.toModel()` hardcodes it `true` for every cached row and it cannot distinguish
one account's capture from another account's download. It is acceptable only as a secondary
"this row came off the device" predicate inside `applyTrailFilter`, where every caller
already passes account-scoped ObjectBox rows.

**Chip construction** — `trail_quick_filter_bar.dart` has a local `buildChip({label, icon,
active, onPressed})` closure inside `build()` that produces the `ActionChip` with
`chipBg`/`chipLabelColor`. Every chip goes through it. The new chip MUST use it too — that
is what satisfies "no bespoke one-off styling". Every other chip opens a bottom sheet; this
one is a plain toggle, which is the only behavioural difference.

**Localization** — `app/l10n.yaml`: arb-dir `lib/i18n`, template `app_en.arb`, 14 locales,
and a committed `untranslated_messages.json` that is expected to grow when a new English-only
key lands. Regenerate with `flutter gen-l10n`. Labels are read as `l10n.<key>` from
`AppLocalizations.of(context)!`.

**Three surfaces mount `TrailQuickFilterBar`:**
| Surface | filterId | Source | offlineOnly behaviour |
|---|---|---|---|
| `library_screen.dart:119` | `'library'` | `trailLibraryProvider` (already downloaded-only) | tautological — every row passes |
| `profile_trail_screen.dart:107` | `profile_trail_{handle}` | `profileTrailsProvider` (network page + local merge, paginated) | the real work |
| `profile_trail_map_screen.dart:690` | `profile_trail_{handle}` | `mapTrailSearchProvider` + `mapClusterSearchProvider` | cannot be honored — see below |

**Why the map screen opts out.** `mapClusterSearchProvider` posts `filterText` to
`/search/trails/cluster` and gets back server-side aggregated clusters, not trails. Device-local
library membership cannot be expressed in a Meilisearch filter string, and a cluster count
cannot be intersected client-side. Honoring the chip for individual trails but not for clusters
would make the map's filtering silently change with zoom level. So the map passes
`showOfflineChip: false` and never renders an affordance it cannot honor. The map and list
share a filterId, so `offlineOnly` may be `true` while the map is mounted — that is fine and
matches today's behaviour for every `TrailFilter` field absent from `toFilterText()`.

**Pagination interaction (the load-bearing detail).** `ProfileTrailsNotifier` mixes in
`PagedLoadMore`; `profile_trail_screen.dart:_onScroll` only calls `loadNextPage()` when
`pos.maxScrollExtent > 0`. So naively post-filtering each 20-hit server page against the
downloaded set is a correctness bug, not just a perf one: if page 1 filters down to zero rows
there is no scroll extent, auto-load never fires, and downloaded trails sitting on page 3 are
unreachable forever. The fix is to invert the source rather than post-filter: when
`offlineOnly` is on the list is built entirely from device rows (which are complete and
already in memory) and `totalPages` is pinned to the current page so `hasMore` is false.

**Resolving "downloaded trails authored by THIS profile"** — `Trail.author` holds the author's
actor record id. For the signed-in hiker's own handle it is `authProvider`'s `user.actorId`.
For another hiker's handle, take it off the network page already fetched:
`TrailSummary.summaryAuthorActorId`. No extra network call and no new identity machinery is
needed — another hiker's profile already hard-fails offline (`_fetchAndMerge` rethrows
`DioException` when `!_isOwnHandle`), so a rendered other-handle list always has network rows
to read the actor id from.

**Pre-existing bug, DO NOT FIX in this plan:** `_isAnyActive` omits `_isDistanceActive`, so a
distance-only filter shows no Reset button. Out of scope. Note it in the SUMMARY; do not touch it.
</discovery_findings>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add offlineOnly to the filter state, the downloaded-id view, and the client-side clause</name>
  <files>
app/lib/models/trail.dart
app/lib/util/trail/filter.dart
app/lib/provider/trail/trail_library_provider.dart
app/lib/i18n/app_en.arb
app/test/util/trail/filter_test.dart
  </files>
  <behavior>
Tests to add to `app/test/util/trail/filter_test.dart` (plain `Trail(...)` fixtures, no Store —
match the existing fixture style in that file):
- With `offlineOnly: false`, a trail whose id is absent from `downloadedIds` and whose
  `isLocal` is false is KEPT (flag off changes nothing).
- With `offlineOnly: true` and an empty `downloadedIds`, a trail with `isLocal: false` is DROPPED.
- With `offlineOnly: true`, a trail whose id IS in `downloadedIds` is KEPT even when
  `isLocal: false`.
- With `offlineOnly: true` and an empty `downloadedIds`, a trail with `isLocal: true` is KEPT
  (a row read off this device is offline-available by construction).
- With `offlineOnly: true`, the clause composes with an existing clause: a downloaded trail
  whose difficulty is excluded by `filter.difficulty` is still DROPPED.
  </behavior>
  <action>
Four coordinated edits, then codegen.

1. `app/lib/models/trail.dart` — add `@Default(false) bool offlineOnly,` to the `TrailFilter`
   freezed factory (the class starting at the `abstract class TrailFilter with _$TrailFilter`
   declaration). Place it next to the other boolean flags (`completed`, `liked`) so it reads
   with its peers. Then add a short comment inside `toFilterText()` — near the `completed`
   clause — stating that `offlineOnly` is deliberately NOT emitted, because Meilisearch indexes
   server-side trail documents and has no field for this device's library membership; it is a
   client-side-only clause. `TrailFilter` has no `fromJson`/`toJson`, so no serialization
   concerns. `@Default(false)` means `buildDefaultTrailFilter` needs no change and
   `resetFilter()` clears the flag for free — verify both, do not add redundant wiring.

2. `app/lib/provider/trail/trail_library_provider.dart` — add a `@riverpod` function provider
   `Set<String> downloadedTrailIds(Ref ref)` that watches `trailLibraryProvider`, maps to
   `t.id`, drops empty ids (unsynced captures have a blanked server id and are handled by the
   `isLocal` branch instead), and returns a set. Follow the `Ref`-first parameter style used by
   `localTrail`/`ownLiveCapture` in `app/lib/provider/trail/local_trail_provider.dart`. Document
   that account scoping is INHERITED from `trailLibraryProvider` and must never be
   re-implemented here — that provider already returns `const []` for a signed-out account,
   which is what keeps one account's downloads from leaking into another's filter.

3. `app/lib/util/trail/filter.dart` — add an optional named parameter
   `Set<String> downloadedIds = const {}` to `applyTrailFilter`, and add the clause inside the
   existing `.where` predicate: when `filter.offlineOnly` is true, drop the trail unless
   `trail.isLocal` is true OR `downloadedIds` contains `trail.id`. Update the function's
   doc comment, which currently enumerates exactly which clauses mirror `toFilterText()` and
   which are deliberately skipped — add `offlineOnly` as a clause that exists ONLY here and has
   no server-side counterpart, and record why `isLocal` is legitimate at this call site
   (it answers "did this row come off the device?", which is precisely offline availability;
   the ban in its own doc comment is on gating destructive actions, badges and tab visibility
   on it, none of which this is). Keep the file dependency-light: no Riverpod, no ObjectBox
   imports — the id set is injected by the caller, which is why it is a parameter and not a
   provider read.

4. `app/lib/i18n/app_en.arb` — add key `available_offline` with value `Available offline`.
   Do NOT reuse the existing `offline` key (line ~372): it means "the device has no
   connectivity" and this chip means "this trail's data is on the device"; collapsing the two
   is the exact conflation the `Trail.isLocal` rename in `models/trail.dart` was made to undo.
   Follow the surrounding entry formatting (the file pairs each key with an `@key` description
   block where its neighbours do — match whatever the adjacent keys do).

Then run codegen and l10n generation:
`dart run build_runner build --delete-conflicting-outputs` (regenerates `trail.freezed.dart`
and `trail_library_provider.g.dart`) followed by `flutter gen-l10n` (regenerates
`app_localizations*.dart`). Commit the regenerated `lib/i18n/untranslated_messages.json` —
`l10n.yaml` documents that its growth is intentionally visible in the diff, so a 13-locale
increase here is expected and must not be suppressed.
  </action>
  <verify>
    <automated>cd app && flutter test test/util/trail/filter_test.dart test/models/trail_filter_test.dart && flutter analyze lib/models/trail.dart lib/util/trail/filter.dart lib/provider/trail/trail_library_provider.dart</automated>
  </verify>
  <done>
`TrailFilter` carries `offlineOnly` defaulting to false; `toFilterText()` still emits no
clause for it (asserted by the existing `test/models/trail_filter_test.dart` staying green);
`downloadedTrailIdsProvider` compiles and derives from `trailLibraryProvider`;
`applyTrailFilter` honors `offlineOnly` against an injected `downloadedIds` set with all five
new tests passing; `available_offline` resolves through `AppLocalizations`.
  </done>
</task>

<task type="auto">
  <name>Task 2: Render the chip in second position and wire the three consuming surfaces</name>
  <files>
app/lib/components/trail/trail_quick_filter_bar.dart
app/lib/routes/library_screen.dart
app/lib/routes/profile_trail_map_screen.dart
app/lib/provider/profile/profile_trails_provider.dart
  </files>
  <action>
1. `app/lib/components/trail/trail_quick_filter_bar.dart`:
   - Add a `final bool showOfflineChip;` field defaulting to `true` in the const constructor.
     Document that it exists solely for surfaces that cannot honor a device-local clause.
   - Add `bool _isOfflineActive(TrailFilter filter) => filter.offlineOnly;` alongside the other
     `_is*Active` predicates, and include it in `_isAnyActive` so Reset appears when the chip is
     the only thing on. Do NOT alter `_isAnyActive`'s existing pre-existing omission of
     `_isDistanceActive` — that is a separate bug and out of scope.
   - In `build()`, compute `offlineActive` next to the other `*Active` locals, and insert the
     new chip into the `Row` between the Sort chip and the Categories chip, preserving the
     `const SizedBox(width: 8)` spacing pattern exactly. Wrap it so nothing renders when
     `showOfflineChip` is false (a collection-`if` around both the chip and its trailing
     spacer — the bar must not gain a double gap when hidden).
   - The chip goes through the existing `buildChip` helper: `label: l10n.available_offline`,
     `icon: Icons.cloud_done_outlined` (outlined Material glyph, matching the
     `category_outlined`/`landscape_outlined` family already in the bar), `active: offlineActive`.
   - Unlike every other chip, `onPressed` opens no sheet — it toggles:
     `ref.read(trailFilterProvider(filterId).notifier).updateFilter((f) => f.copyWith(offlineOnly: !f.offlineOnly))`,
     guarded by the same `filter != null ? ... : () {}` pattern the neighbouring chips use.

2. `app/lib/routes/profile_trail_map_screen.dart` (line ~690) — pass `showOfflineChip: false`
   to `TrailQuickFilterBar`, with a comment giving the reason recorded in
   `<discovery_findings>`: `/search/trails/cluster` returns server-side aggregated clusters that
   cannot be intersected against a device-local id set, so honoring the chip would make
   filtering silently depend on zoom level.

3. `app/lib/routes/library_screen.dart` — in `_filtered`, pass the downloaded id set through to
   `applyTrailFilter` (`ref.watch(downloadedTrailIdsProvider)`, read in `build()` and threaded
   into `_filtered`, matching how `filterAsync.value` is already threaded). Every row here comes
   from `trailLibraryProvider`, so the clause is tautological and the visible list will not
   change — wire it anyway so there is exactly one filtering path, and say so in a comment so a
   future reader does not "clean up" the apparently-dead argument.

4. `app/lib/provider/profile/profile_trails_provider.dart` — this is the real behaviour change.
   - Thread `downloadedIds` into `_readOwnLocal`'s existing `applyTrailFilter(local, filter)`
     call, reading `ref.read(downloadedTrailIdsProvider)` in the same style the method already
     uses `ref.read` for `objectBoxProvider` (a `ref.watch` here would reintroduce the rebuild
     storm the file's comments document at length — use `ref.read`).
   - In `_fetchAndMerge`, after the existing fetch/`DioException` block, add an `offlineOnly`
     branch taken when `filter?.offlineOnly == true`. In that branch, build the returned
     `ProfileTrailsState` from DEVICE ROWS ONLY rather than from `mergeOwnTrails`:
     resolve `authorActorId` as `authProvider`'s `user.actorId` when `_isOwnHandle`, else the
     first non-null `summaryAuthorActorId` among the freshly fetched `networkTrails`; take
     `ref.read(trailLibraryProvider)` narrowed to rows whose `author` equals that
     `authorActorId` (empty list when it is null — never fall back to an unnarrowed library
     read, which would put another profile's trails on this profile); narrow those rows by the
     search query with the existing `filterOwnTrailsByQuery(rows, q)`; union them with the
     `local` list already passed in; dedupe on `id` when non-empty and `localId` otherwise
     (the same key `profile_trail_screen.dart` uses for its `ValueKey`); then run
     `applyTrailFilter(deduped, filter, downloadedIds: ...)` so the remaining clauses and the
     sort still apply.
   - In that same branch set `page: 1` and `totalPages: 1` so the inherited
     `hasMore => page < totalPages` is false. This is what stops `_onScroll` from calling
     `loadNextPage()` and is why no change to `PagedLoadMore`, `canLoadMore` or `appendPage` is
     needed. Add a comment explaining that post-filtering paginated server pages was rejected
     because a zero-match first page yields no scroll extent, so auto-load never fires and
     deeper downloaded trails become unreachable.
   - Leave `ProfileTrailsState.offline` FALSE in this branch. It is documented as "decided from
     the fetch outcome itself", and `profile_trail_screen.dart` renders the
     `own_trails_offline_banner` from it — reusing it here would tell an online hiker they have
     no connection.
   - The network fetch itself still runs unchanged (it is what supplies `authorActorId` for a
     non-own handle, and a `DioException` there must keep its current rethrow-for-other-handles
     behaviour).
  </action>
  <verify>
    <automated>cd app && flutter analyze lib/components/trail/trail_quick_filter_bar.dart lib/routes/library_screen.dart lib/routes/profile_trail_map_screen.dart lib/provider/profile/profile_trails_provider.dart && flutter test</automated>
  </verify>
  <done>
`flutter analyze` is clean on all four files and the full `flutter test` suite passes. The
quick filter bar renders Sort, then "Available offline", then Categories. The offline chip is
absent on the profile trail map screen. `ProfileTrailsNotifier` returns a device-sourced,
non-paginating state when `offlineOnly` is set, without setting `offline: true`.
  </done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <what-built>
An "Available offline" toggle chip in the trail quick filter bar, second position, right after
"Sort". It filters trail lists down to trails whose data is already on this device, sourced from
the existing account-scoped offline library (`TrailEntity.savedByUserIds`). It is hidden on the
profile trail map screen, where server-side clustering cannot express device-local membership.
  </what-built>
  <how-to-verify>
Build and install the app yourself (this plan never runs `flutter build` or `adb install`), then:

1. Open your own profile's trails list. Confirm the chip order reads: Sort, Available offline,
   Categories, Difficulty, Distance, Elevation gain, Date, Completion status.
2. Confirm the chip is visually identical to its neighbours (same height, radius, avatar size,
   inactive/active colours) — it must look like it was always there.
3. Tap it. Confirm it turns active (filled `primaryContainer`) and the list narrows to trails
   you have downloaded or recorded on-device. Confirm the "Reset" text button appears.
4. Scroll to the bottom of the filtered list. Confirm there is NO trailing infinite-scroll
   spinner and no further pages load.
5. Tap "Reset". Confirm the chip goes inactive and the full list returns.
6. Turn on airplane mode, toggle the chip on, and confirm every trail it shows actually opens.
7. Open another hiker's profile trails list (while online), toggle the chip, and confirm you see
   only their trails that you have downloaded.
8. Open the map view from a profile trails list (the map icon in the app bar). Confirm the
   offline chip is NOT present there.

One product decision to confirm: the chip is currently also shown on the **Library** tab, where
it is inert — every trail on that tab is downloaded by definition, so toggling it changes
nothing. Say "hide it on Library" if you would rather it not render there, and it will be given
`showOfflineChip: false` at that call site too.
  </how-to-verify>
  <resume-signal>Type "approved", or describe what is off (chip order, styling, filtering behaviour, or the Library-tab decision)</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| account A ↔ account B on one device | One `TrailEntity` row is shared by every account that downloaded it; only `savedByUserIds` separates them |
| signed-out ↔ signed-in | A signed-out read of the trail box would expose every account's downloaded (including private) trails |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-vct-01 | Information disclosure | `downloadedTrailIdsProvider` | mitigate | Derives strictly from `trailLibraryProvider`, which already filters on `savedByUserIds.containsElement(currentAccountId(store))` and returns `const []` for a null account. The new provider adds no query of its own — account scoping is inherited, never re-implemented. |
| T-vct-02 | Information disclosure | `ProfileTrailsNotifier` offlineOnly branch | mitigate | The library read is narrowed by `Trail.author == authorActorId`; when `authorActorId` is null the branch contributes an EMPTY list. An unnarrowed fallback would render another profile's downloaded trails on this profile and is explicitly forbidden in the task action. |
| T-vct-03 | Information disclosure | `TrailFilter.toFilterText()` | mitigate | `offlineOnly` is deliberately not emitted into the Meilisearch filter string, so the set of trails this device holds is never disclosed to the server. Guarded by the existing `test/models/trail_filter_test.dart`. |
| T-vct-SC | Tampering | package installs | accept | No new npm/pub/cargo dependencies are added by this plan; nothing to audit. |
</threat_model>

<verification>
- `cd app && flutter analyze` reports no new issues
- `cd app && flutter test` passes, including the five new `offlineOnly` cases in
  `test/util/trail/filter_test.dart`
- `cd app && flutter gen-l10n` runs clean and `lib/i18n/untranslated_messages.json` shows
  `available_offline` newly untranslated in the 13 non-English locales (expected growth)
- `grep -v '^#' app/lib/models/trail.dart | grep -c offlineOnly` returns a non-zero count
- No `flutter build` or `adb install` was run at any point
</verification>

<success_criteria>
- The quick filter bar renders "Available offline" as the second chip, right after "Sort", built
  through the existing `buildChip` helper with no bespoke styling
- The chip's label is resolved via `AppLocalizations` (`l10n.available_offline`), never hardcoded
- Toggling the chip narrows the profile trails list to device-held trails and pins
  `hasMore` to false so server pagination stops
- `ProfileTrailsState.offline` is untouched by the chip, so the "you're offline" banner never
  appears because of it
- Reset clears `offlineOnly` along with every other filter
- The chip does not render on the profile trail map screen
- Full `flutter analyze` and `flutter test` are clean
</success_criteria>

<output>
Create `.planning/quick/260808-vct-add-a-new-chip-to-the-filter-bar-that-fi/260808-vct-SUMMARY.md` when done.
Record in it: the `_isAnyActive`/`_isDistanceActive` pre-existing bug observed but deliberately
not fixed, and the user's answer on whether the chip should also be hidden on the Library tab.
</output>
