---
phase: 32-on-demand-polygon-fetch-seed-slimming
plan: 06
subsystem: infra
tags: [git, filter-branch, history-rewrite, force-push, pocketbase]

# Dependency graph
requires:
  - phase: 32-on-demand-polygon-fetch-seed-slimming
    plan: 32-02
    provides: Migration reads the plain-JSON regions_seed.json catalog, not the .gz
  - phase: 32-on-demand-polygon-fetch-seed-slimming
    plan: 32-03
    provides: D-07 equivalence check already consumed the .gz as its oracle; no longer needed
provides:
  - db/migrations/initial_data/regions_seed.json.gz deleted from the working tree and purged from feature/app's entire git history
  - Local backup tag backup/pre-seed-purge-20260728 (escape hatch, not pushed to origin)
  - feature/app force-pushed (--force-with-lease); origin now matches the rewritten local history exactly
  - .planning/todos/completed/2026-07-28-purge-regions-seed-blob-from-git-history.md (moved from pending/, resolution recorded)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["git filter-branch --index-filter targeting a single literal path (never a glob) plus --prune-empty, verified via an empty diff against a pre-rewrite backup tag before any force-push"]

key-files:
  created: []
  modified:
    - db/migrations/initial_data/regions_seed.json.gz (deleted, working tree and all history)
    - .planning/todos/completed/2026-07-28-purge-regions-seed-blob-from-git-history.md (moved + resolution appended)

key-decisions:
  - "Task 2 (blocking checkpoint:decision) pre-resolved by the developer as purge-and-push before this executor ran; recorded verbatim below with the re-checked facts"
  - "Backup tag backup/pre-seed-purge-20260728 deliberately kept local-only, never pushed to origin -- pushing it would keep the old ~55MB blob reachable on the remote and defeat the purge"
  - "Verification used a genuine fresh git clone rather than trusting local git count-objects alone, since the retained local backup tag keeps the old blob locally reachable and masks any local size reduction until that tag is eventually deleted"

patterns-established:
  - "For any future git history purge on this repo: tag first, verify HEAD-vs-tag diff is empty before pushing, force-with-lease (never bare --force), and prove the result via an actual fresh clone, not just local count-objects (which lies while a backup tag exists)"

requirements-completed: [SLIM-01, SLIM-04]

# Metrics
duration: ~30min
completed: 2026-07-28
---

# Phase 32 Plan 06: Purge regions_seed.json.gz from git history Summary

**Deleted the retired 57MB gzip seed artifact from feature/app's working tree and its entire git history via `git filter-branch --index-filter` over 133 commits, verified byte-identical HEAD content against a local backup tag, force-pushed with `--force-with-lease`, and confirmed a genuine fresh clone shrinks from a 268.00 MiB to a 198.14 MiB pack while still migrating up to the identical 1306-row region catalog.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-07-28T10:15:00Z (approx.)
- **Completed:** 2026-07-28T10:47:00Z
- **Tasks:** 3 (Task 2 was a pre-resolved checkpoint:decision, no code/history action)
- **Files modified:** 2 (`db/migrations/initial_data/regions_seed.json.gz` deleted; the folded todo moved+edited)

## Accomplishments
- Confirmed both consumers that had kept the `.gz` alive through the phase were finished with it: plan 32-02's migration reads only the plain-JSON `regions_seed.json`, and plan 32-03's D-07 equivalence check already consumed the `.gz` as its oracle and recorded its result
- Deleted `db/migrations/initial_data/regions_seed.json.gz` from the working tree; verified zero remaining references to the literal string `regions_seed.json.gz` anywhere in `db`/`web`/`docs`, `go build ./...` and `go test ./...` both green, and a fresh (network-free beyond pre-existing Meilisearch/ORIGIN setup) `migrate up` still reseeding the identical 1306-row catalog (153 group / 1153 leaf, `catalog_commit` populated, `region_geometry` empty)
- Recorded Task 2's pre-resolved decision (`purge-and-push`) with the two re-checked facts, per the orchestrator's pre-resolution
- Rewrote `feature/app`'s history over `b1665219^..HEAD` with `git filter-branch --index-filter 'git rm --cached --ignore-unmatch db/migrations/initial_data/regions_seed.json.gz' --prune-empty`, tagged a local-only backup first, verified the rewritten `HEAD`'s tree is byte-identical to the backup tag's tip (`git diff` empty), confirmed zero `regions_seed.json.gz` blobs remain reachable from `HEAD` (only the plain-JSON path's blob does), force-pushed with `--force-with-lease` pinned to the pre-rewrite remote tip, reclaimed local `refs/original/*` + expired the reflog + ran `git gc --prune=now`, and proved the result end-to-end against a genuine fresh clone (blob absent, plain-JSON intact, `migrate up` reproduces the identical 1306-row catalog, pack size 268.00 MiB -> 198.14 MiB)

## Task Commits

Each task was committed atomically:

1. **Task 1: Delete the gzip artifact and prove nothing reads it** - originally committed as `6566c128` (feat). This commit was later folded away (pruned as empty) by Task 3's `--prune-empty` history rewrite, because once the rewrite removed the `.gz` from the earlier introducing commit (`b1665219`) onward, this commit's own "delete the already-filtered-out file" diff became a no-op relative to its filtered parent. The deletion itself is fully preserved in the rewritten history (baked in from `b1665219` onward) — only the redundant standalone commit object disappeared. Still recoverable via the backup tag if ever needed.
2. **Task 2: Decide whether to rewrite and force-push published history** - pre-resolved by the developer as `purge-and-push` before this executor ran (no commit; decision-only task, no history/tag/push action taken during this task itself, per its own acceptance criteria)
3. **Task 3: Rewrite the range, verify, and force-push** - `6a342cde` (chore) + follow-up `5f7a6fd6` (docs, folding in a resolution-note edit that was mistakenly left unstaged in the first commit)

**Plan metadata:** commit to follow (docs: complete plan)

## Task 2 Decision Record (pre-resolved)

**Returned option id:** `purge-and-push` (recorded verbatim, per the orchestrator's pre-resolution)

Re-checked facts, gathered and reported before the decision was made:
- Rewrite range `git rev-list --count b1665219^..HEAD` = **132 commits** (planning-time estimate was ~100/101)
- `git rev-list --count origin/feature/app..HEAD` = **38 commits** ahead of the remote (planning-time value was 7)
- `git merge-base --is-ancestor b1665219 origin/feature/app` = **true** — the blob is published, so a force-push is required
- Of the 132 commits in the rewrite range, **94 are already on the remote** and would receive new SHAs — this blast radius was stated to the developer before they chose
- `git count-objects -vH` before any work: `size-pack: 268.00 MiB`, `count: 246`

No tag was created, no history was rewritten, and no push occurred during Task 2 itself — all of that happened in Task 3, gated on this recorded decision.

## Files Created/Modified
- `db/migrations/initial_data/regions_seed.json.gz` - Deleted from the working tree (Task 1) and purged from every commit in `feature/app`'s history (Task 3, `b1665219^..HEAD`)
- `.planning/todos/completed/2026-07-28-purge-regions-seed-blob-from-git-history.md` - Moved from `pending/`; appended a `## Resolution (2026-07-28, plan 32-06)` section recording the option chosen, the rewrite/push/verification specifics, and the reset-not-merge instruction for anyone tracking the branch

## Decisions Made
- Kept the backup tag `backup/pre-seed-purge-20260728` strictly local — never pushed it to `origin`. Pushing it would have kept the ~55MB blob reachable on the remote via that tag ref, defeating the entire purpose of the purge for anyone who later clones or fetches tags.
- Used `git filter-branch` (not `git-filter-repo`) since `git-filter-repo` was not installed and 133 commits was well within `filter-branch`'s practical range (rewrite completed in ~27 seconds); installing a new tool for this run wasn't warranted per Rule 3's package-install exclusion, and the plan explicitly framed `git-filter-repo` as an optional alternative "if the range has grown large enough."
- Verified the purge's real-world effect via an actual fresh `git clone` (both for blob absence and for the `migrate up` acceptance criterion) rather than relying solely on local `git count-objects`, because the retained local backup tag keeps the old blob reachable locally and would otherwise make local size numbers misleadingly unchanged.
- Force-pushed using `--force-with-lease=feature/app:<old-remote-sha>` (the concrete pre-rewrite remote tip, fetched immediately beforehand) rather than a bare `--force-with-lease`, for the strongest possible guarantee against clobbering a concurrent remote update.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 3's own commit initially omitted a staged content edit**
- **Found during:** Task 3, post-rewrite cleanup
- **Issue:** After `git mv`-ing the todo file into `completed/`, its content was edited (adding the `## Resolution` section) via a direct file write, but the edit wasn't re-staged before the `chore(32-06)` commit — so that commit only recorded the rename, not the resolution text.
- **Fix:** Staged and committed the missing content in a small immediate follow-up commit (`5f7a6fd6`), before this local-only commit was ever pushed.
- **Files modified:** `.planning/todos/completed/2026-07-28-purge-regions-seed-blob-from-git-history.md`
- **Verification:** `git diff` against the working tree is now clean; `git show 5f7a6fd6` contains exactly the intended addition.
- **Committed in:** `5f7a6fd6`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Cosmetic-only slip in the todo file's committed content, caught and corrected before any push; no impact on the history rewrite, verification, or force-push, all of which completed correctly on the first attempt.

## Note on a stale planning-time fact (not a deviation, no fix needed)

The plan's `verified_facts` stated `git log --oneline --all -- db/migrations/initial_data/regions_seed.json` "returns nothing" and Task 3's own re-confirmation instruction expected the same. By the time this plan executed, plans 32-01 through 32-05 had already landed, so that path now has real history (introduced by 32-01's `d6c0bafd`). This is expected drift from the planning-time snapshot (only the two facts explicitly marked "re-check" were guaranteed to still hold) and posed no actual risk: the index-filter targets the literal `.gz` path only, an exact string match with no glob, so a shared/adjacent history on the plain-JSON path can never be caught by it regardless of whether that path has zero or non-zero prior history. Confirmed after the rewrite: the plain-JSON blob (`a43aa9cf...`) is the only `regions_seed` object reachable from `HEAD`.

## Issues Encountered
- Docker Desktop was not running at the start of this plan; started it and ran a throwaway local Meilisearch container (`v1.11.3`, port 17700) to satisfy earlier, unrelated migrations in the same `migrate up` chain — identical one-time setup step 32-02 already documented. Torn down after verification. A second throwaway container (port 17701) was used later for the post-rewrite fresh-clone `migrate up` proof and also torn down.
- `git-filter-repo` was not installed locally; used `git filter-branch` instead (see Decisions Made above) — completed in ~27 seconds for 133 commits, well within acceptable range, no installation needed.

## User Setup Required
None - no external service configuration required for this plan's own change. (The two temporary Meilisearch containers used only for local/fresh-clone verification have already been torn down.)

## Next Phase Readiness
- `db/migrations/initial_data/regions_seed.json.gz` no longer exists anywhere in `feature/app` — working tree or history. Fresh clones no longer pay the ~55MB tax.
- **Anyone tracking `feature/app` must reset, not merge:** `git fetch origin && git reset --hard origin/feature/app`. History diverged for the 94 commits (of the 132-commit rewrite range) that were already on the remote before this push; a `git pull` or `git merge` will produce spurious conflicts/duplicate commits instead of a clean update.
- Backup tag `backup/pre-seed-purge-20260728` remains local-only on this machine as the escape hatch. It can be deleted (`git tag -d backup/pre-seed-purge-20260728`) once the rewritten `feature/app` has been confirmed good in normal use for a few days; deleting it (followed by another local `git gc --prune=now`) will also finally shrink this machine's own local `.git` pack, which currently still reports ~268 MiB locally because the tag keeps the old blob reachable here.
- The folded todo (`.planning/todos/pending/2026-07-28-purge-regions-seed-blob-from-git-history.md`) is now in `.planning/todos/completed/` with the resolution recorded — nothing further pending from it.
- Phase 32 (on-demand-polygon-fetch-seed-slimming) is now fully complete: all 6 plans executed, the seed is geometry-free, and the one artifact the phase couldn't shrink by editing code (the historical blob) has been retired by history rewrite.
- No blockers identified.

---
*Phase: 32-on-demand-polygon-fetch-seed-slimming*
*Completed: 2026-07-28*

## Self-Check: PASSED
