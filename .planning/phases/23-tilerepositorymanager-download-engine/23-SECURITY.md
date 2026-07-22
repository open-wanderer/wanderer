---
phase: 23-tilerepositorymanager-download-engine
audited: 2026-07-22
auditor: gsd-security-auditor
asvs_level: 1
block_on: high
threats_total: 8
threats_closed: 8
threats_open: 0
---

# Phase 23: TileRepositoryManager Download Engine — Security Audit

**Scope:** Verification-only. Every threat below was authored at plan time (`register_authored_at_plan_time: true` for all six 23-0X-PLAN.md files). This audit does not scan for new vulnerabilities — it verifies that each declared mitigation is actually present in the implemented code (or, for `accept` dispositions, that the acceptance rationale still holds against the real implementation).

## Threat Verification

| Threat ID | Category | Component | Disposition | Status | Evidence |
|-----------|----------|-----------|-------------|--------|----------|
| T-23-01 | Tampering | region `id` → file path / request path | mitigate | **CLOSED** | `app/lib/util/region_file_path.dart:20` (`regionIdPattern = ^[a-z0-9][a-z0-9_-]*$`), `:27-36` (`assertValidRegionId` throws `ArgumentError` on mismatch). Called at every path/request-path-building call site in `app/lib/services/tile_repository_manager.dart:103,188,264,279,325` (`startVectorDownload`, `startDemDownload`, `pauseRegion`, `resumeRegion`, `deleteRegion`) *before* any `regionVectorPath`/`regionDemPath`/`regionStorageDir`/`_requestPathFor` call. SvelteKit proxies retain the identical server-side allow-list unchanged: `web/src/routes/api/v1/regions/[id]/download/+server.ts:5-7` and `download-dem/+server.ts:5-7` (`RegionIdSchema` regex, parsed before the upstream fetch URL is built). |
| T-23-02 | Tampering / DoS | downloaded `.part` archive integrity | mitigate | **CLOSED (code-level)** — see caveat below | `app/lib/services/tile_repository_manager.dart:435-445` (`_isValidPmTiles` calls `PmTilesArchive.fromFile`, catches `CorruptArchiveException`/`UnsupportedError` → `false`). Gated before promotion at `:146-150` (vector) and `:231-235` (DEM): a failed validation deletes the `.part` file and sets `PackageStatus.error`; only a valid archive reaches `partFile.renameSync(finalPath)` + `PackageStatus.downloaded`. |
| T-23-03 | Denial of Service | oversized/mismatched declared size, low-storage device | mitigate | **CLOSED (code-level)** — see caveat below | `app/lib/util/disk_space_util.dart:58-65` (`hasEnoughSpace`: `freeBytes == null → false`; default `safetyMultiplier: 1.75`). Called in `tile_repository_manager.dart:116-124` (vector) and `:201-209` (DEM) *before* `Directory(...).createSync()` or any file write; on refusal the package is set `PackageStatus.error` and the method returns with zero bytes written. |
| T-23-05 | Tampering | persisted `PackageStatus`/`RegionStatus` `.code` | mitigate | **CLOSED** | `app/lib/models/region_status.dart:53-62`: `PackageStatus.paused(3)`/`error(4)` appended after `downloaded(2)` — `notDownloaded(0)`/`downloading(1)`/`downloaded(2)` untouched. `:35-45`: `RegionStatus.paused(4)`/`error(5)` appended after `updateAvailable(3)` — `0/1/2/3` untouched. `app/lib/entities/downloaded_tile_package_entity.dart:25-30`: `dbStatus` setter uses `PackageStatus.values.firstWhere((s) => s.code == value, orElse: () => PackageStatus.notDownloaded)` — an unrecognized future/corrupt code degrades safely rather than throwing or misdecoding. (`RegionStatus` itself is never persisted — `RegionEntity.status` is a getter-only computed value per `region_entity.dart:89-91`, so it has no `.code`-decode path to corrupt.) |
| T-23-07 | Denial of Service (orphaned storage) | region delete leaving orphaned files/rows | mitigate | **CLOSED** | `app/lib/services/tile_repository_manager.dart:324-366` (`deleteRegion`): cancels any in-flight `:vector`/`:dem` `CancelToken`s, then inside one `_store.runInTransaction(TxMode.write, ...)` removes both `DownloadedTilePackageEntity` rows (`packageBox.remove(obxId)`) and clears `region.vectorPackage.target`/`demPackage.target` to `null`; afterward (best-effort, existence-guarded) deletes the vector/dem final paths and both `.part` siblings, and the region storage dir if left empty. Both halves (DB rows + on-disk files) are handled explicitly since ObjectBox does not cascade a `ToOne` target's removal. |
| T-23-SC | Tampering (supply chain) | `disk_space_2` third-party native plugin | mitigate | **CLOSED** | `.planning/phases/23-tilerepositorymanager-download-engine/23-03-SUMMARY.md` "Legitimacy Decision (Task 1 — resolved)" section (lines 73-86) records live evidence gathered *before* the package was added: pub.dev registry/score API responses (150/160 pub points, 19,017 downloads/30d, MIT license), GitHub fork-provenance confirmation (`tom-ludwig/disk_space_2` forked from unmaintained `activcoding/Disk-Space`), and a verbatim read of both native plugin sources (`DiskSpace_2Plugin.kt`/`.swift`) confirming the entire native surface is three read-only `StatFs`/`NSFileManager` free/total-space queries — no network, file-write, reflection, or shell-out. `app/pubspec.yaml:14` (`disk_space_2: ^1.0.12`) was added only after this review, per the SUMMARY's stated task ordering (Task 1 decision-only gate → Task 2 `flutter pub add`). |
| T-23-04 | Information Disclosure | forwarded `Range` header (SvelteKit proxy → Go backend) | accept | **CLOSED (rationale confirmed accurate)** | Accept rationale: "Range offsets are byte ranges into a public (but auth-gated) region archive the user already has access to request in full; forwarding discloses nothing beyond what's already authorized." Confirmed against the real implementation: `db/main.go:229-236` shows `regionsGroup.Bind(apis.RequireAuth())` wraps both `/regions/{id}/download` and `/regions/{id}/download-dem` — the archive is genuinely auth-gated (any logged-in user), not actually public. `web/src/routes/api/v1/regions/[id]/download/+server.ts:54-58` forwards only the caller's own session Bearer token plus their own `Range` header — no cross-user or elevated access is introduced by the Range passthrough. Rationale holds. |
| T-23-06 | Denial of Service | resumed download re-requesting a stale/attacker-influenced offset | accept | **CLOSED (rationale confirmed accurate)** | Accept rationale: "The Range offset is computed from the app's own local `.part` file length (`File.length()`), never from server-supplied or user-supplied input." Confirmed: `app/lib/services/tile_repository_manager.dart:412-413` — `_downloadResumable` computes `alreadyDownloaded` via `partFile.existsSync() ? partFile.lengthSync() : 0` (local `dart:io` filesystem call), then `resumePlanFor(alreadyDownloaded)` derives the `Range: bytes=<offset>-` header from that local value alone. No server response field or user input feeds the offset. Rationale holds. |

## Accepted Risks Log

The following risks are formally accepted (not mitigated) for Phase 23, per the disposition declared in each threat's originating PLAN.md. Recorded here per this project's accepted-risk documentation convention.

1. **T-23-04 (Information Disclosure — forwarded `Range` header).** Accepted because the region archive endpoints require authentication (`apis.RequireAuth()`, confirmed in `db/main.go:230`) and a `Range` request only lets an already-authorized caller fetch a byte-subset of an archive they could otherwise fetch in full. No new disclosure surface is introduced by forwarding the header.
2. **T-23-06 (DoS — resumed download re-requesting a stale offset).** Accepted because the resume offset is derived exclusively from the local `.part` file's own byte length (`File.lengthSync()`), never from any server-supplied or user-supplied value, so there is no attacker-controllable input to this decision.

## Caveat: On-Device Confirmation Outstanding for T-23-02 / T-23-03

Both T-23-02's and T-23-03's mitigation plans (as declared in `23-04-PLAN.md` and repeated in `23-06-PLAN.md`) cite `on-device-confirmed per 23-06` as part of their evidence chain. The **code-level control is genuinely present** (see evidence above) and is what this audit verifies. However, the *on-device* confirmation itself has **not actually happened yet**:

- `.planning/phases/23-tilerepositorymanager-download-engine/23-VERIFICATION.md` (frontmatter `status: human_needed`) lists both "RESUME (TILE-02)" (→ T-23-02) and "DISK REFUSAL (TILE-03)" (→ T-23-03) under "Human Verification Required," explicitly stating no evidence exists that the on-device harness run has occurred.
- `.planning/phases/23-tilerepositorymanager-download-engine/23-UAT.md` shows all 5 human-check items, including tests #1 (RESUME) and #2 (DISK REFUSAL), with `result: [pending]` and `passed: 0`.

This is not a code gap — no implementation change is required — and does not change either threat's CLOSED status at the code-verification level this audit performs. It is flagged here so the claimed "on-device-confirmed" evidence is not silently treated as complete: the on-device human-check (physical-device Range-resume + real disk-refusal walkthrough via `app/test/services/tile_repository_manager_harness.dart`) should be completed before treating these two threats as fully closed end-to-end, consistent with 23-06-PLAN.md's own verification design.

## Unregistered Flags

None. No `23-0X-SUMMARY.md` file in this phase contains a `## Threat Flags` section (confirmed by grep across all six summaries), so no new attack surface was reported by the executor during implementation.

One item was reviewed for registration but judged not to warrant a flag: `app/test/services/tile_repository_manager_harness.dart` (added in 23-06) includes a "Backend base URL" text field wired to `apiProvider.notifier.updateBaseUrl`, letting a tester point the harness at an arbitrary backend. This is a debug-only driver, confirmed absent from `router_provider.dart`/`main.dart` (zero production references, per `23-06-SUMMARY.md`'s self-check and this audit's own grep), lives under `app/test/`, and is never shipped in the production app bundle's navigation graph. It is not a production attack surface and is not registered as a threat.

## Verification Commands Used

```
grep -rn "assertValidRegionId" app/lib/
grep -rn "regionVectorPath|regionDemPath|regionStorageDir" app/lib/
grep -rn "PmTilesArchive.fromFile" app/lib/services/tile_repository_manager.dart
grep -rn "hasEnoughSpace|runInTransaction" app/lib/services/tile_repository_manager.dart
grep -n "paused(3)|error(4)|paused(4)|error(5)" app/lib/models/region_status.dart
grep -n "disk_space_2" app/pubspec.yaml
grep -n "regions/:id/download|RequireAuth" db/main.go
grep -rn "Threat Flags" .planning/phases/23-tilerepositorymanager-download-engine/*.md   # 0 matches
```

## Summary

**Threats closed:** 8/8 (6 `mitigate` + 2 `accept`, all confirmed against implemented code/config, not documentation alone)
**Threats open:** 0
**Blockers:** none
**Caveat (non-blocking):** on-device human-check for T-23-02/T-23-03 declared-but-not-yet-executed (see above) — recommend completing before treating Phase 23 as fully done end-to-end, per the phase's own `human_verify_mode: end-of-phase` design.

---
*Audited: 2026-07-22*
*Auditor: gsd-security-auditor*
