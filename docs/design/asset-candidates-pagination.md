# Design: Cursor pagination and request budgets for asset candidates

Status: Implemented on 2026-08-05. The current implementation and operational behavior are documented in [`asset-plugins.md`](asset-plugins.md).
Affects: `asset_library.v1`, `RuntimeSession`, the photo library picker, auto-attach.

## 1. Starting point

Before this change, an asset plugin's candidate search was a single, indivisible WASM call with no way to continue.

- The capability input had neither a cursor nor a search limit.
- The output carried `hasMore` only as a bare hint, without continuation state.
- The host did not paginate plugin candidates. For plugins, `hasMore` had no consumer; only the internal wanderer library had a load-more flow.
- Immich therefore scanned up to 250 pages of 250 assets each, so up to 62,500 assets, synchronously in one call.
- There was no upper bound on the number of host HTTP requests per plugin call. `db/pluginsystem/host_http.go` limited only bytes per response.
- `assetIds` was not checked for length before the plugin call. An emptiness check existed only in the `import-to-waypoint` and `import-to-target` branches; the plain `import` path had neither check.

The statelessness of WASM execution is not an obstacle here. The repository already contains the right convention: the sync capability passes an opaque `state` back and forth (`plugins/sdk/types.go`, `ListInput` and `ListOutput`) and the host threads it through a batch loop (`db/routes/plugin_system_sync.go:295`). The asset contract should adopt that same convention rather than introduce a second cursor concept.

## 2. Capability contract

### 2.1 Input

The existing import limits stay unchanged. Search budgets get their own block because they concern a different level: `limits` are write limits at import time, `search` are plugin-enforced read budgets during the search.

Important for everything that follows: `limits` is **not** consistently host-enforced. `assetPluginPhotoImportLimits(hostConfig, false)` returns `nil` (`db/routes/plugin_system_assets.go:1813`), so manual imports run without any per-target limit. The importer's aggregate media budget also applies only in the `input.StorageMode == "copy"` branch; in `link_private` the media counter is never incremented. A manual `link_private` import is therefore completely uncapped today.

```json
{
  "limits": {
    "maxPhotosPerTrail": 100,
    "maxPhotosPerWaypoint": 10,
    "maxPhotosPerSummitLog": 20
  },
  "search": {
    "maxItems": 100,
    "maxScannedItems": 2500,
    "maxProviderRequests": 10
  },
  "state": {
    "page": 2
  },
  "request": {
    "action": "candidates"
  }
}
```

A dedicated type `AssetSearchLimits` is introduced for this. `importer.PhotoImportLimits` stays untouched — the importer never reads the search fields and should not carry them.

| Field | Meaning |
| --- | --- |
| `search.maxItems` | Maximum number of candidates returned. |
| `search.maxScannedItems` | Maximum number of **evaluated** provider assets per call. Strict, see section 2.4. |
| `search.maxProviderRequests` | Soft, plugin-side request budget for `candidates`. |
| `state` | Plugin continuation state only, treated as opaque by the host. |

`search.maxProviderRequests` is explicitly **not** the security enforcement. It exists so the plugin can return in a controlled way with `state + hasMore` instead of running into a hard runtime limit. Enforcement is described in section 3.

### 2.2 Output

```json
{
  "candidates": [],
  "state": { "page": 12 },
  "hasMore": true,
  "stats": { "scannedItems": 2500 }
}
```

`stats.scannedItems` is self-reported and serves observability, not enforcement. It makes visible how much scan budget a call consumed.

For the `import` action a normative rule is added that did not exist before:

> The IDs in `photos` and `omittedAssetIds` together form an **exact partition** of the requested `assetIds`. In detail:
>
> 1. Every ID in `photos` and every ID in `omittedAssetIds` must have been requested.
> 2. Within each of the two sets, every ID appears at most once.
> 3. No ID may appear in both sets.
> 4. The union of both sets must equal the set of requested IDs — no more, no less.
>
> Any violation is a protocol error.

Without this rule the re-sorting in section 5.3 would not be definable: it maps photos to a candidate rank via `externalId` and needs the guarantee that this mapping is unambiguous and complete. So far this was nowhere in the contract; it merely held in practice because `assetsByID` walks the IDs in order.

Rule 3 is the case that would slip through without being stated explicitly: a plugin that reports an ID as both imported *and* omitted leaves it open whether the photo should be persisted. Rule 4 turns the two individual rules from section 5.3 — nothing unrequested, nothing silently missing — into a single checkable statement.

### 2.3 State contract rules

- `hasMore=true` requires a non-empty continuation state.
- The continuation state must differ from the input state.
- The comparison is done over canonically serialized or hashed JSON state, not over map ordering.
- A violation is a protocol error. It must **not** be silently treated as `hasMore=false` — that is exactly what would let auto-attach work incompletely without anyone noticing, reproducing the very class of bug this rework is meant to remove.
- Server-side batch loops additionally remember all state hashes seen and thereby detect cycles such as `A → B → A`.
- The batch cap remains as a last safety net, analogous to `defaultPluginSyncMaxBatches`.

### 2.4 Continuation within a provider page

Provider page size and `search.maxItems` are independent of each other. An Immich page returns 250 assets, `maxItems` may be 100. A state carrying only `page` would lose 150 already-found candidates in that case, because the next continuation starts on the following page. The same problem arises when `search.maxScannedItems` runs out mid-page.

The contract therefore specifies: **the state represents a position within the provider page.**

```json
{
  "state": {
    "page": 12,
    "offset": 100
  }
}
```

Two alternatives were rejected:

- *Count provider pages atomically.* Forces `maxItems >= provider page size` and thereby makes the limit useless as soon as a provider returns large pages.
- *Allow bounded overshoot of `maxItems`.* Moves the problem into the response size and makes the limit unreliable for the caller.

Two rules follow:

- `maxItems` is a hard upper bound on returned candidates. A plugin that found more matching photos on a page returns exactly `maxItems` and writes the remaining position into the state.
- `maxScannedItems` is **strict** and counts the provider assets actually evaluated. If the budget runs out in the middle of a provider response, the plugin stops there and writes the position into the state.

`offset` counts **raw provider assets**, not candidates found. The candidate count is a function of filtering and is unsuitable as a resume point.

Both rules mean that a continuation reloads the partially consumed provider page and skips forward to `offset`. That is the deliberately accepted price:

- A stateless plugin cannot cache a partially consumed page. The `maxItems` case forces the reload anyway.
- This makes both stop reasons share the same mechanism and the same code path — instead of two semantics with different resume behaviour.
- The cost is one additional provider request per continuation boundary inside a page.

The overhead cannot be given as a single number, nor as a closed formula. It depends on the provider page size, the number of matches, the **position** of the matches within the page, and on whether the scan budget additionally ends mid-page. A page may need to be reloaded even when all its matches fit into a single response — namely to evaluate its remaining assets at all.

Instead of a formula, three concrete cases at 250 assets per provider page:

| Case | Loads of this page |
| --- | --- |
| No matches on the page | 1 |
| 100 matches within the first 100 assets, `maxItems` 100 | 2 — the second load evaluates the remaining 150 assets |
| 250 matches, `maxItems` 100 | 3 — returning 100, 100 and 50 |

The worst case is therefore a track whose photos sit densely in a few provider pages — exactly the pattern of a day hike. That is acceptable because it is a small absolute number of requests, but it should be kept in mind when choosing `maxItems` relative to the provider page size: a `maxItems` close to the page size makes refetches rare.

The rejected alternative would have been an atomic page budget counting fetched instead of evaluated assets, allowed to overshoot by at most one provider response. It saves the refetch, but would correctly have had to be named `maxFetchedItems`, and it would have created two different continuation mechanisms side by side, because the `maxItems` case still needs the refetch.

The internal structure of the state stays provider-specific and opaque to the host. `page`/`offset` is the Immich shape, not a contract schema.

## 3. Hard request budget in the runtime

### 3.1 Why not in `RequestPolicyContext`

`RequestPolicyContext` (`db/pluginsystem/policy.go:19`) is fixed at `OpenSession` and is therefore session-wide. For asset calls that would be harmless, because `callAssetPlugin` opens its own session per action (`db/routes/plugin_system_assets.go:1126`). Sync breaks the assumption: the same session serves both the `list` batches (`plugin_system_sync.go:310`) and every `detail` call (`:369`). A session-wide budget would have to be sized for the more expensive of the two and would be ineffective for the other.

### 3.2 Contract change

Both interfaces in `db/pluginsystem/runtime.go` change, not just `RuntimeSession`:

```go
type RuntimeCallOptions struct {
    MaxHostRequests int
}

type Runtime interface {
    Call(
        ctx context.Context,
        plugin LocalPlugin,
        export string,
        input []byte,
        policy RequestPolicyContext,
        options RuntimeCallOptions,
    ) ([]byte, error)

    OpenSession(
        ctx context.Context,
        plugin LocalPlugin,
        policy RequestPolicyContext,
    ) (RuntimeSession, error)
}

type RuntimeSession interface {
    Call(
        ctx context.Context,
        export string,
        input []byte,
        options RuntimeCallOptions,
    ) ([]byte, error)

    Close(ctx context.Context) error
}
```

`Runtime.Call` forwards the options to the session it opens internally. `OpenSession` stays unchanged — the budget is deliberately not session-wide (section 3.1).

The scope is small: there are only two real `Call` implementations (`db/pluginsystem/worker.go:60` and `:159`), plus the `UnavailableRuntime` stubs and exactly one test file with fakes.

### 3.3 Responsibilities

| Level | Carries |
| --- | --- |
| Session | Manifest, connector policy, auth, absolute runtime ceilings. |
| Call | Action- or export-specific `MaxHostRequests`. |
| Worker | Local request counter inside `workerRuntimeSession.call()`. |

The effective limit is the minimum of the absolute runtime ceiling and the call budget. A missing or invalid call budget does **not** mean unlimited; it falls back to a safe runtime default — the same pattern `host_http.go:251` already uses for `maxBytes <= 0`.

Both values are kept as named constants in `db/pluginsystem`, next to the existing runtime limits:

| Constant | Value | Role |
| --- | --- | --- |
| `DefaultMaxHostRequestsPerCall` | 64 | Applies when a call path sets no budget. |
| `AbsoluteMaxHostRequestsPerCall` | 512 | Ceiling that no call budget can exceed. |

The absolute ceiling must permit the largest legitimate call. That is `import` with 208 requests (section 3.6); 512 leaves headroom without being unlimited.

The default of 64 covers every path today except large imports. `import` must therefore set its budget explicitly — it can exceed 64. A forgotten budget on a new path thus produces a clear error instead of silent unboundedness.

### 3.4 Worker implementation

- Initialize the counter inside `workerRuntimeSession.call()`.
- Count every `workerMessageHostHTTPRequest`.
- On overflow, fail the **entire call** with a dedicated budget error.
- Reset the counter on the next `Call` of the same session.

The third point is a deliberate decision. Merely returning an error to the plugin for the N+1 host request would let the plugin swallow that error and return a successful partial result — exactly the silent incompleteness the state rules in section 2.3 also rule out. A budget overflow is a host error, not a plugin signal.

A session field reset per `Call` is race-free because `workerRuntimeSession.Call` serializes via `callMu` (`worker.go:159`). Passing it as a parameter is still preferable because it is harder to misuse.

### 3.5 Derivation per action

The values are derived from the actual request behaviour of the existing plugins. The derivation is deliberately kept in the document so the numbers stay justifiable later instead of looking like grown magic.

What is counted is the level the worker sees: every `workerMessageHostHTTPRequest`, that is, every host request the plugin triggers during the export. What the host does **after** the export is not counted, and neither is every network hop: redirects are handled inside a single `ExecuteHostRequest` and produce no additional worker RPC.

| Call | Actual need | Budget | Derivation |
| --- | --- | --- | --- |
| `check` | 2 | 4 | `/api/users/me` plus probe search. |
| `candidates` | 10 | 24 | `maxScannedItems` 2500 ÷ 250 provider page size = 10 pages, plus refetches at continuation boundaries (section 2.4). Must exceed the soft auto-attach budget of 16 (section 5). |
| `thumbnail` | 0 | 4 | The export only builds a `MediaRef` (`plugins/immich/immich.go:101`). The fetch then happens host-side via `importer.FetchPhotoMedia`. Pure future headroom. |
| `import` | `len(assetIds)` | `len(assetIds) + 8` | One GET per ID (`plugins/immich/immich.go:167`). Capped at 208 by the limit from section 3.6. |
| Sync `list` | 1–2 | 8 | Komoot and Strava send one list request per batch. |
| Sync `detail` | 1–3 | 16 | Strava needs three: activity, streams, photos (`plugins/strava/strava.go:75`, `:81`, `:90`). |
| Session refresh | 1 | 4 | One token request, for example `login()` in Hammerhead. |
| Trail send | 0–1 | 8 | The export only builds a plan; the transfer then happens host-side via `ExecuteHostRequest` (`db/routes/plugin_system_send.go:117`). The single request occurs only when `userIDForUpload` falls back to `login()` for lack of a token var (`plugins/hammerhead/hammerhead.go:142`). Mostly future headroom. |

For `thumbnail` and trail send the budget is deliberately higher than the actual need. Both exports could load provider metadata in the future; a budget of 0 would forbid that extension only at runtime. The documented actual need nevertheless stays 0 and 0–1 respectively, so that nobody later infers real behaviour from the budget.

The soft `search.maxProviderRequests` for `candidates` always sits below the hard budget of 24: at 10 for the picker, at 16 for auto-attach (section 5). The gap is intentional — the plugin should return in a controlled way with `state + hasMore` at the soft value and never reach the hard limit. Reaching it means something is wrong.

The hard budget is therefore a per-call value, not a per-action constant: it must exceed the soft budget of the respective caller. 24 covers both of today's callers.

`import-to-waypoint` and `import-to-target` are already normalized to `action="import"` before the plugin call and need no separate budget branch.

The call paths outside assets and sync must not be forgotten — each of them needs an explicit budget or a defined safe default:

- Session refresh in `injectSessionAuth`, which can run via both `Session.Call` and `Runtime.Call` (`db/pluginsystem/auth_injection.go`).
- Trail send (`db/routes/plugin_system_send.go:117`).

A call path without a budget falls back to the runtime default per the rule in section 3.3, not to "unlimited".

A flat budget across all actions is not possible: `import` resolves every asset ID with its own GET (`plugins/immich/immich.go:167`), so it needs up to `len(assetIds)` requests in one call, while `candidates` gets by with a single-digit number.

### 3.6 Rejecting oversized `assetIds` up front

The host rejects oversized `assetIds` lists before calling the plugin.

The obvious justification — "surplus IDs would be discarded by the importer anyway" — does **not** hold. It applies only to auto-attach. For manual imports `PhotoLimits` is `nil` (`db/routes/plugin_system_assets.go:1813`, called with `enforce=false` at `:1688`), and the aggregate media budget applies only in the `copy` branch. A list of 5,000 IDs in `link_private` mode would be processed in full today: 5,000 provider requests in the plugin, 5,000 records in the importer.

The limit is therefore defined as a **standalone request limit**, independent of `PhotoImportLimits`, with a fixed maximum of **200 IDs per request**. It simultaneously bounds the runtime budget derived from `len(assetIds)` in section 3.5 — otherwise that budget would itself be unbounded — yielding at most 208 requests there.

200 is far above any realistic manual selection in the picker and therefore barely restricts today's freedom. For comparison: `util.DefaultPluginMaxImportMediaItems` is 20, but applies only in `copy` mode. For `link_private` this limit is the first effective bound at all.

The check belongs in `pluginSystemAssetCall`, before the action branching, so that it also covers the plain `import` path — which today does not even have the emptiness check of the two `import-to-*` branches.

**Duplicates in `assetIds`** are stably deduplicated at the same place, that is, keeping the first occurrence. `assetsByID` walks the IDs in order without inspection (`plugins/immich/immich.go:167`) and returns two `Photo` entries with an identical `externalId` for an ID passed twice. Under the partition rule from section 2.2 that would be a protocol error — correctly detected, but caused by the host itself. Deduplication must run before the length check so that a list does not fail the 200 limit because of duplicates.

The alternative would be to deliberately enforce `PhotoImportLimits` for manual imports as well. That is a real behaviour change for users who today intentionally select more photos than the per-target limit allows, and therefore does not belong in this rework as a side effect. If it is wanted, it needs its own design.

## 4. Cursor handling towards the browser

The raw plugin state must not reach the browser. **Decided: the host keeps it in a short-lived server-side store and hands the browser only a random ID.**

The alternative, a signed or encrypted token, is rejected. It would survive restarts and work across instances, but would still need a server-side batch counter against cycles (section 4.2) — which removes its main advantage, statelessness. The store covers cycle protection without an extra construct, keeps the ID constantly small, and never lets the provider state leave the process, which means the encryption question does not have to be answered but disappears.

The price is in-process state: after a backend restart, "load more" fails. That is the same model wanderer already accepts and documents for the thumbnail cache and remote jobs. The picker must handle this case cleanly — an expired or unknown ID means: restart the search, not show an error.

The store entry must be bound to at least:

- user or actor,
- plugin ID,
- instance ID,
- normalized search parameters,
- **fingerprint of the effective configuration and authentication**,
- expiry time.

The binding is checked on load-more. If one of the values differs — for example because the user moved the time range or changed the instance configuration — the entry is discarded and pagination starts over.

Binding to user or actor is not just hygiene: without it, a guessed cursor ID would be a way to page through someone else's candidate results.

The fingerprint must include the **authentication**, not only the configuration. If the API key of the same Immich instance is swapped without `url` or any other configuration field changing, the cursor otherwise still points at a pagination running against a different library and a different user context — page 12 of something else entirely. This is implemented as a hash over the effective configuration plus the auth secrets, or alternatively via an instance revision that increments on every change to `plugin_instances`. Raw auth material is not stored, only its hash.

The store also needs its own resource limits because it is fillable from outside. The values are modelled on the thumbnail cache (512 entries, 64 MiB) but smaller, because an entry only holds metadata:

| Limit | Value |
| --- | --- |
| Entries globally | 512, evicting the least recently used |
| Total memory | 8 MiB |
| Entries per user | 8 |
| Plugin `state` per entry | 4 KiB; a larger state is a protocol error |
| Batches per entry | 50 |
| Lifetime | 10 minutes since last use |
| Cursor entropy | 128 bits from a cryptographic source |

The lifetime is sliding, not absolute: it suits a user loading more repeatedly in the picker and clears abandoned searches quickly. 50 batches times `maxItems` 100 yields 5,000 candidates per search — far more than a picker sensibly displays, and still finite.

### 4.1 Browser contract

The host never returns the plugin `state` to the browser, only the field `cursorId`.

**This requires a dedicated response type.** The plugin endpoint today serializes `pluginAssetLibraryOutput` — the raw plugin response — directly to the client (`db/routes/plugin_system_assets.go:761`). As soon as that struct carries a `state` field, the provider state lands in the browser without any further action. That is exactly what section 4 is meant to prevent.

The design therefore specifies a separate browser DTO, for example `pluginAssetCandidatesResponse`, with exactly these fields:

| Field | Origin |
| --- | --- |
| `candidates` | Plugin output |
| `hasMore` | Plugin output |
| `takenAfter` | Plugin output |
| `hasTimestamps` | Plugin output |
| `cursorId` | Host, from the store |
| `restartRequired` | Host |

`state` and `stats` are explicitly **not** included. The `default` branch must no longer pass the plugin output through unchanged.

This must not be confused with `assetLibraryResponse` (`db/routes/plugin_system_assets.go:173`): that type belongs to the internal wanderer photo library and stays untouched. Both endpoints deliver candidates to the same picker, but they are separate response types.

The search response of the plugin endpoint therefore looks like this:

```json
{
  "candidates": [],
  "hasMore": true,
  "cursorId": "8f3c…",
  "restartRequired": false
}
```

The load-more request sends `cursorId` **in addition to the unchanged search parameters**. The parameters are not redundant: the host compares them against the binding stored in the entry and thereby detects the case where the client changed filters but still sent the cursor. Without them it would have to trust the client to reset on every filter change.

| Question | Decision |
| --- | --- |
| Field name towards the browser | `cursorId`, named identically in response and follow-up request |
| Search parameters in the load-more request | yes, complete; they serve the binding check |
| `restartRequired` | normal `200` response with an empty candidate list and without `cursorId` |
| Entry on `hasMore=false` | deleted immediately |

**`restartRequired` is deliberately not an error status.** A `4xx` would surface in the picker as an error message even though an expired cursor is a normal, expected state — for instance after a backend restart or after ten minutes of inactivity. The picker should discard the old results and quietly search again instead of confronting the user with an error.

Deleting immediately on `hasMore=false` keeps the store small and prevents an exhausted cursor from being used again later.

### 4.2 Cycle protection in the interactive path

The cycle protection from section 2.3 relies on state hashes seen so far and works only in server-side batch loops that hold those hashes across the loop. The picker has no such loop: every "load more" is its own request. A signed, stateless token carries no history, so a plugin could still produce `A → B → A` and the browser would keep loading indefinitely.

The store from section 4 solves this along the way: its entry holds, besides the current state, the set of state hashes already seen. The cycle rule from section 2.3 thereby applies in the interactive path just as it does in a server-side loop — that was the decisive argument for the store.

State size, lifetime and the batch cap per entry are specified in the limits table in section 4.

## 5. Missing consumers

**Picker.** `PhotoLibraryPickerModal` needs, per plugin, the cursor ID issued by the host, `hasMore` and `loading` — **not** the plugin `state`, which never leaves the process (section 4). Candidates loaded afterwards must be deduplicated and re-sorted again through `uniqueCandidates`. Without that, every cursor is inconsequential.

On the `restartRequired` signal (section 4.1) the picker must discard the candidates already displayed before restarting the search. A restart that builds on a half-filled list would mix results from two different pagination runs.

**Auto-attach.** Needs a server-side batch loop with unambiguous behaviour when the cap is reached: a visible error instead of silent partial results.

**Auto-attach does not use the picker's `search` values.** That is not a detail but a precondition for preserving reach.

A batch ends as soon as *one* of the two limits applies: `maxItems` candidates found or `maxScannedItems` assets evaluated. With the picker values (100 and 2500), a batch in a dense result set ends after roughly 100 evaluated assets. 25 batches would then reach 2,500 instead of 62,500 examined assets — one twenty-fifth of today's reach. Guaranteeing 62,500 assets would require 625 batches in the extreme case.

Auto-attach therefore sets `maxItems` **equal to** `maxScannedItems`:

| Parameter | Picker | Auto-attach |
| --- | --- | --- |
| `maxItems` | 100 | 2500 |
| `maxScannedItems` | 2500 | 2500 |
| `maxProviderRequests` | 10 | 16 |
| Batch cap | 50 (store) | 25 (`defaultAssetAutoAttachMaxBatches`) |

Since the number of candidates can never exceed the number of evaluated assets, `maxItems` is no longer binding. Only then does this hold:

> 25 batches × 2500 evaluated assets = 62,500 — exactly the reach auto-attach has today with 250 pages of 250 assets.

This equation has two preconditions that must be stated, because violating them produces the same silent reduction in reach as the picker values:

1. The provider returns full pages. At 250 assets per page a batch covers 2500 assets with exactly ten requests. If the provider returns shorter pages, the same asset count needs more requests.
2. `maxProviderRequests` does not bind before the scan budget. With the picker value of 10, the limit for Immich sits exactly on the ten pages required — without any headroom. Auto-attach therefore raises this budget too, so that the scan budget stays the binding limit rather than the request count.

If a batch ends early for one of these reasons, nothing breaks: the loop continues with the next batch. It then reaches 62,500 only after more than 25 batches — and thereby runs into the cap, which per section 5 produces a visible error. That is exactly as intended: the reduction is reported instead of happening silently.

Step 4 of the rollout therefore changes the *distribution* of the work, but not the *reach*. Had the cap instead been set to `defaultPluginSyncMaxBatches` (100), reach would have silently quadrupled.

Behind this sits a general point: as soon as `maxItems` is smaller than `maxScannedItems`, the search no longer returns "all matches out of `N` assets" but "the first `maxItems` matches in provider order". For the picker this is harmless — the user keeps paging and the display re-sorts anyway. For auto-attach it would be a silent change of selection: the best matches could sit beyond the stop boundary and would never be evaluated.

The upper bound of collected candidates is 62,500, corresponding to 313 import blocks of 200 IDs. Memory usage is the same as today, because today's single call also holds all matches from 62,500 assets in memory.

**Decided: auto-attach collects all candidates across all batches and imports once afterwards.**

The decisive reason is correctness, not robustness. `clusterWaypointPhotos` assigns each photo to the first cluster whose current centre lies within the radius, and updates that centre after every photo. The result therefore depends on the total set *and* the order. Batch-wise import would create different waypoints depending on the batch boundary — the same photo could land on an existing cluster once and get its own waypoint another time. That is not a robustness problem but incorrect clustering.

Collecting additionally avoids persisted partial results when a later batch fails. The memory cost is negligible: candidates are pure metadata, no media bytes, and their number is bounded by `maxItems` times the batch cap.

### 5.1 "Import once" does not mean "one plugin call"

`maxItems` times the batch cap can yield considerably more than 200 candidates, while section 3.6 limits every `import` call to 200 IDs. The two rules only appear to conflict: they concern different levels. The flow therefore separates plugin resolution from host-side persistence:

1. Collect candidates across all batches, deduplicate them and **sort them globally** (section 5.2).
2. Call plugin `import` in blocks of at most 200 IDs, in sort order.
3. Collect all returned `Photo` descriptors and **re-sort them into the global candidate order** (section 5.3), still without persisting.
4. Perform exactly **one** host-side import: global clustering, waypoint creation, limits, persistence.

Only step 2 is chunked. The clustering in step 4 sees the full set, which preserves the correctness argument above. The budgets from section 3.5 also apply per block, not to the total set.

### 5.2 Global sorting before the import

Sorting per batch is not enough. `sortMatches` sorts only the matches of the respective call (`plugins/immich/matching.go:44`), and `clusterWaypointPhotos` processes photos strictly in input order (`db/routes/waypoint_cluster.go:142`). Concatenated batches, each sorted on its own, would therefore again produce clustering dependent on `maxItems` and the batch boundary — exactly what section 5 is meant to exclude.

Auto-attach therefore re-sorts the entire candidate set before step 2 begins. The comparator is part of the contract because it determines the clustering result:

1. `distanceFromStart` ascending,
2. `distance` ascending,
3. `takenAt` ascending, with missing values or values not parseable as RFC 3339 sorted **after** all valid ones,
4. `assetId` ascending, byte-wise.

Stage 3 needs the explicit rule for missing values because `takenAt` is a string in the candidate contract and an empty string would otherwise sort before every valid timestamp — photos without a capture time would lead the clustering. Two invalid values count as equal; stage 4 then decides.

Stage 4 is not a cosmetic criterion but establishes the total order: without it, the ordering of identical metadata stays dependent on the batch boundary.

Deduplication happens before sorting, keyed on `assetId`. Within one auto-attach run for one plugin instance the provider is constant, so an additional provider key is unnecessary. This deliberately differs from the picker, which mixes results from several providers and therefore deduplicates on `(externalProvider, externalId)`.

Sorting by `distanceFromStart` processes photos in track order, which is the natural ordering for spatial clustering — neighbouring photos meet each other instead of jumping across the track.

This is a deliberate behaviour change: today the clustering receives the plugin order, that is `distance` ascending and, on ties, `takenAt` descending. It belongs in step 3 of the rollout and must be called out there as a behaviour change, not treated as a side effect.

### 5.3 Re-sorting the plugin responses

The global sorting from section 5.2 orders the *candidates*. What gets clustered, however, are the `Photo` descriptors the plugin returns from `import` — and their order is guaranteed nowhere. Nothing in the contract obliges a plugin to return `photos` in the order of the `assetIds` passed in; Immich only does so today because `assetsByID` walks the IDs in order (`plugins/immich/immich.go:167`). A plugin that parallelizes or groups could return any other order — and would thereby destroy the batch independence just established.

The host therefore establishes the order itself, after step 3 from section 5.1:

1. Map every `Photo.externalId` to the original `Candidate.assetId`.
2. Sort the photos by global candidate rank, not by response order and not by block order.
3. An `externalId` that appeared in no requested block is a protocol error — the plugin delivered something that was not requested.
4. A duplicated `externalId` is likewise a protocol error.
5. A requested but undelivered `externalId` is **also a protocol error** — unless the plugin explicitly reports it in `omittedAssetIds`.

Point 5 was a silent skip with an internal counter in an earlier version of this design. That was wrong, for two reasons.

First, it contradicts the underlying principle: no successful partial result without a visible signal. The same rule already applies to budget overflows (section 3.4) and to invalid state (section 2.3).

Second, the justification was factually wrong. `photoFromAsset` discards only assets with an empty ID (`plugins/immich/immich.go:288`), and a candidate that was delivered earlier has a non-empty `assetId` by definition. The obvious real-world case — the asset was deleted between candidate search and import — already makes the entire GET, and with it the import call, fail in Immich (`assetsByID`). A legitimate silent omission therefore does not exist today.

**Explicit partial results.** So that a plugin can nevertheless handle the deletion case tolerantly in the future without losing the whole run, the `import` output may carry `omittedAssetIds`:

```json
{
  "photos": [],
  "omittedAssetIds": [
    { "assetId": "abc", "reason": "not_found" }
  ]
}
```

Omitting without an entry in `omittedAssetIds` remains a protocol error. This leaves exactly two outcomes — complete, or explicitly and visibly incomplete — but no silent third one.

**Visibility on both import paths.** `omittedAssetIds` belongs to the general `import` output and therefore concerns more than auto-attach:

| Path | Visibility |
| --- | --- |
| Auto-attach | Omitted IDs appear in the summary next to `imported`, and additionally in the log. |
| Manual import | The response becomes an envelope `{ imported, omitted }`. |

The manual path today responds with a bare array of imported results (`db/routes/plugin_system_assets.go:1672`). Without a change, an omission would be invisible there — the user selects twenty photos, receives eighteen, and learns nothing.

The alternative of failing manual imports on any omission was rejected. The user selected the photos explicitly; a single photo deleted in the meantime must not prevent the other nineteen. An envelope is equally visible but not destructive.

This is a format change to an existing response. Affected is `assets_import_plugin_links` in `web/src/lib/stores/asset_store.ts`, which today passes the array straight to `appendAssets`. The migration belongs in rollout step 3.

## 6. Consistency against Immich

Immich's `nextPage` is a page number, not a stable database cursor — the plugin parses it with `strconv.Atoi` and only checks monotonicity (`plugins/immich/immich.go:154`). Wanderer wraps it as an opaque continuation token but gains no snapshot guarantee from it.

This is documented as accepted weak consistency, not solved:

- The closed, mostly past time window from `takenAfter`/`takenBefore` keeps the result set largely stable during a search.
- Duplicates are caught anyway by the existing deduplication on `(externalProvider, externalId)`.
- What realistically remains are possible omissions when a photo is deleted or its EXIF date changed during pagination.

The strict scan semantics from section 2.4 reload partially consumed pages and are therefore exposed to this drift a second time: if the content of page 12 shifts between two calls, `offset` points at a different position. The consequence stays in the same order of magnitude — individual skipped or duplicated assets, the latter caught by deduplication.

For a photo candidate picker, that is the right order of magnitude.

## 7. Rollout

The order matters for safety, because only the last step changes behaviour:

1. Introduce `state`, `hasMore`, state validation, `AssetSearchLimits` and `RuntimeCallOptions`.
2. Equip the picker with the cursor ID, load-more, and `restartRequired` handling.
3. Equip auto-attach with the server-side batch loop, batch cap, global sorting (section 5.2) and import chunking (section 5.1).
4. Only then reduce the current effective scan limit of 62,500 to a smaller per-call budget.

Step 3 contains a deliberate behaviour change: the global sorting changes the input order of the clustering and thereby the waypoints created. It is unavoidable — without it the clustering would stay dependent on batch boundaries — but it belongs in the changelog and must not be treated as a silent side effect of pagination.

Atomic is best. If several deployments are needed, the first must retain the current high scan budget. Otherwise auto-attach silently finds fewer photos from the first deployment on: no error, no log, no UI signal.

## 8. Test matrix

Contract and pagination:

- Relevant candidates appear only after several empty batches.
- Auto-attach finds candidates beyond the first scan budget.
- A provider page with more matches than `maxItems` loses no candidates: the rest of the page appears in the following batch.
- A `maxScannedItems` exhausted mid-page stops there and evaluates no further asset in that call.
- Unchanged and cyclic state is rejected.
- A cycle `A → B → A` is detected across two separate picker requests as well, not only within a server-side loop.
- An expired or unknown store entry returns `restartRequired`; the picker then discards the displayed candidates and searches again instead of loading more onto the old list.
- A cursor ID belonging to another user is rejected.
- Swapping the API key of the same instance — without changing `url` or any other configuration field — invalidates the cursor and returns `restartRequired`.
- A load-more request with a valid `cursorId` but changed search parameters returns `restartRequired` instead of paging on.
- On `hasMore=false` the store entry is no longer findable afterwards.
- A plugin `state` larger than 4 KiB results in a protocol error.
- A store entry whose bound search parameters or configuration fingerprint no longer match is discarded.
- Auto-attach clusters over the total set of all batches: the same candidate set yields the same waypoints regardless of `maxItems` and batch boundary. The test must run the same set with at least two different `maxItems` values and expect identical waypoints.
- The comparator from section 5.2 produces a total order: candidates with identical `distanceFromStart`, `distance` and `takenAt` keep the same order across batch boundaries.
- Auto-attach with more than 200 candidates calls `import` in blocks but persists exactly once, and the clustering sees the total set.
- A plugin that returns the `photos` of every 200-ID block in **reverse** order produces the same waypoints as one that preserves the order of the `assetIds`.
- An `externalId` that was not requested, and a duplicated `externalId`, each result in a protocol error.
- A requested `externalId` that is neither delivered nor reported in `omittedAssetIds` results in a protocol error.
- An `externalId` reported in `omittedAssetIds` lets the import continue and appears in the auto-attach summary or in the `omitted` part of the manual envelope.
- An ID present in both `photos` and `omittedAssetIds` results in a protocol error.
- An ID in `omittedAssetIds` that was never requested results in a protocol error.
- Auto-attach examines 62,500 assets even at a dense match rate, not just `maxItems` times the batch cap — the test needs a fixture in which almost every scanned asset is a match.
- Duplicate `assetIds` in the request produce no protocol error but are removed before the plugin call; the length check applies to the deduplicated list.
- If the provider returns shorter pages than expected, an auto-attach batch ends at the request budget instead of the scan budget, and reaching the batch cap reports a visible error instead of silently finding fewer photos.
- The response of the plugin candidate endpoint contains neither `state` nor `stats` — verified on the serialized JSON, not on the Go type.
- Candidates without or with an invalid `takenAt` sort behind all valid ones in the comparator and do not change waypoints depending on the batch boundary.
- A `maxScannedItems` exhausted mid-page resumes at the raw asset index on the following call, not at the candidate index.
- Picker results loaded afterwards are correctly deduplicated and re-sorted.
- A reached batch cap produces a visible error instead of silent partial results.

Request budget:

- `check` can perform its two requests.
- `candidates` stops at the search budget with a resumable state.
- `import` with exactly the maximum permitted photo count does not exceed the runtime budget.
- An `assetIds` list above 200 is rejected before the plugin call, on all three import routes — including the plain `import` path — and also in `link_private` mode.
- A budget overflow fails the entire call; a plugin that ignores the host error cannot return a successful partial result.
- Session refresh and trail send complete successfully within their budget, including the case where trail send triggers a `login()` request for lack of a token var.
- A call without a set budget runs with `DefaultMaxHostRequestsPerCall`, not unlimited.
- A call budget above `AbsoluteMaxHostRequestsPerCall` is capped at the absolute limit.
- The request counter is reset between two calls of the same session.
- The next request above the action-specific limit is blocked.
- Two different exports within the same session receive different limits, and the counter starts at zero on every `Call`.

## 9. Decisions and implementation status

Decided and **no longer** open:

| Question | Decision | Section |
| --- | --- | --- |
| Passing `MaxHostRequests` | Call parameter, not a session field | 3.2 |
| Action-specific budget values | Table with derivation | 3.5 |
| `assetIds` upper bound | 200, standalone request limit | 3.6 |
| Picker cursor | Server-side store, random ID to the browser | 4 |
| Cycle protection in the browser path | State hashes seen, held in the store entry | 4.2 |
| Auto-attach batching | Collect, then persist once; `import` in blocks of 200 | 5, 5.1 |
| Clustering order | Global sorting, comparator is part of the contract | 5.2 |
| Order of plugin responses | Host re-sorts `photos` by `externalId` into candidate rank | 5.3 |
| Missing photos on import | Protocol error unless explicitly in `omittedAssetIds` | 2.2, 5.3 |
| `photos` + `omittedAssetIds` | Exact partition of the requested IDs | 2.2 |
| Visibility of manual omissions | Envelope `{ imported, omitted }` instead of a bare array | 5.3 |
| `search` values for auto-attach | `maxItems` = `maxScannedItems` = 2500, not the picker values | 5 |
| Browser response type | Dedicated DTO without `state`/`stats`, not the raw plugin output | 4.1 |
| Auto-attach batch cap | `defaultAssetAutoAttachMaxBatches = 25`; reach-preserving only together with `maxItems` = 2500 | 5 |
| Browser contract | `cursorId`, search parameters in the follow-up request, `restartRequired` as `200` | 4.1 |
| Cursor binding | Fingerprint over effective configuration **and** authentication | 4 |
| Store limits | 512 entries, 8 MiB, 8 per user, 4 KiB state, 50 batches, 10 minutes | 4 |
| Continuation within provider pages | State represents a position within the page | 2.4 |
| Scan semantics | Strict: evaluated assets, `offset` as raw asset index | 2.4 |
| Runtime default and ceiling | 64 and 512, named constants | 3.3 |

The reference documentation in `docs/design/asset-plugins.md` has been updated to describe the implemented contract, cursor lifecycle, runtime budgets, picker behavior, auto-attach batching, import envelope, and affected functions.

Deliberately outside this design:

- Enforcing `PhotoImportLimits` for manual imports as well (section 3.6). That is a noticeable behaviour change and needs its own design.
- The trail send budget is set conservatively without measuring the path. Verify it against reality on the first real run.
