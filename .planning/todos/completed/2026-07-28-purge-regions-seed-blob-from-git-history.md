---
created: 2026-07-28T00:00:00.000Z
title: Purge the 55MB regions_seed.json.gz blob from git history
area: repo
resolves_phase: 32
files:
  - db/migrations/initial_data/regions_seed.json.gz
---

## Problem

`db/migrations/initial_data/regions_seed.json.gz` is ~54.65 MB and lives in pushed
history on `feature/app` as of 2026-07-28. Phase 32 replaces it with a sub-100 KB
geometry-free catalog — but **deleting the file does not shrink the repository**.
The blob stays in history forever, so every clone keeps paying for it.

Purging requires a history rewrite plus a force-push. That is cheap right now, while
`feature/app` is effectively a single-owner branch, and expensive after it merges to
`main` — at which point the blob is permanent for every contributor.

There is precedent: the 730 MB uncompressed `regions_seed.json` was purged the same
way on 2026-07-28 (commits `67d056c9`/`490a685f`, since rewritten). That purge did
**not** require a force-push because the blob existed only in unpushed commits. This
one will, because the `.gz` is already on the remote.

## Solution

Sequence matters — do this **after** Phase 32 lands and **before** `feature/app`
merges to `main`.

1. Confirm Phase 32 is merged into `feature/app` and the `.gz` is no longer read by
   any code path (`grep -rn regions_seed`).
2. Find every commit that introduced or touched the blob:
   `git log --oneline --all -- db/migrations/initial_data/regions_seed.json.gz`
3. Tag a backup of the current tip before touching anything.
4. Rewrite the range with an index-filter, as with the earlier purge:
   ```
   FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch --index-filter \
     'git rm --cached --ignore-unmatch db/migrations/initial_data/regions_seed.json.gz' \
     --prune-empty -- <first-bad-commit>^..HEAD
   ```
   Consider `git-filter-repo` instead if the range spans most of the branch — it is
   substantially faster, though it wants a fresh clone.
5. Verify: HEAD's tree is unchanged apart from the intended deletion, and no blob
   over ~1 MB remains in the rewritten range.
6. Coordinate the force-push. Anyone who has pulled `feature/app` needs a heads-up —
   their local branch will have diverged and needs a reset, not a merge.
7. Reclaim local space: expire reflogs and `git gc --prune=now`.

## Verification

- `git rev-list --objects <branch> | git cat-file --batch-check` reports no blob
  above ~1 MB in the branch's history.
- A fresh `git clone` of the branch is dramatically smaller than before.
- `migrate up` against an empty `pb_data` still produces the identical 1306-row
  catalog (153 group / 1153 leaf, zero orphaned parents) from the new slim seed.

## Risk

Force-pushing rewritten history is disruptive to anyone tracking the branch. If
`feature/app` has gained other contributors by the time this runs, weigh the ~55 MB
against the coordination cost — the blob is a permanent tax, but a botched rewrite
loses work. The backup tag from step 3 is the escape hatch.

## Resolution (2026-07-28, plan 32-06)

Completed as `purge-and-push`. `git filter-branch --index-filter` over
`b1665219^..HEAD` (133 commits, 2 pruned as empty), backup tag
`backup/pre-seed-purge-20260728` created locally (not pushed) before the rewrite,
`HEAD` tree verified byte-identical to the backup tag's tip, force-pushed with
`--force-with-lease=feature/app:3667e058...` (old remote tip). A fresh clone of
`feature/app` confirmed the `.gz` gone, `regions_seed.json` intact, and
`migrate up` producing the identical 1306-row catalog (153 group / 1153 leaf,
`region_geometry` empty). Fresh-clone `.git` size dropped from a 268.00 MiB
pre-rewrite pack to 198.14 MiB. Full detail in
`.planning/phases/32-on-demand-polygon-fetch-seed-slimming/32-06-SUMMARY.md`.

Anyone tracking `feature/app` must `git fetch origin && git reset --hard
origin/feature/app` (not merge/pull) — local history diverged for the 94
commits that were already on the remote.
