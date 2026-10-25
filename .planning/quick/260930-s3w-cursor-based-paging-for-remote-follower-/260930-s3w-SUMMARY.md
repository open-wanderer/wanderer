---
phase: quick-260930-s3w
plan: 01
status: complete
subsystem: federation
tags: [activitypub, paging, cursor, go, sveltekit, flutter]
requirements: [CUR-01, CUR-02, CUR-03, CUR-04, CUR-05, CUR-06]
key-files:
  modified:
    - db/federation/actor.go
    - db/federation/collection_test.go
    - db/routes/remote_profile_follow.go
    - db/routes/activitypub.go
    - web/src/lib/stores/profile_store.ts
    - web/src/routes/profile/[handle]/users/[type]/+page.svelte
    - web/src/routes/api/v1/profile/[handle]/follows/+server.ts
    - app/lib/provider/profile/profile_follows_provider.dart
    - app/lib/provider/profile/profile_follows_provider.freezed.dart
    - app/lib/provider/profile/profile_follows_provider.g.dart
  created:
    - app/test/provider/profile_follows_cursor_test.dart
---

# Quick 260930-s3w: Cursor-based paging for remote follower lists

Remote follower/following pages after page 1 now cost one remote request via a validated cursor (the previous response's `next`), instead of re-walking the collection from its root on every page.

## Commits

| Task | Commit | Checkout | Branch | Message |
| ---- | ------ | -------- | ------ | ------- |
| 1 | 9361ee36 | CODE_WT (scratchpad/cursor-paging) | fix/federation-cursor-paging | fix(federation): page remote follower lists with cursors |
| 2 | 1303a41e | CODE_WT (scratchpad/cursor-paging) | fix/federation-cursor-paging | fix(web): send the follower list cursor when loading more |
| 3 | eeac540a | APP_WT (wanderer) | feature/app | fix(app): page follower lists with the server cursor |

Nothing was pushed.

## What changed

- **Backend (actor.go):** `ErrInvalidCursor`, `CollectionNext(collectionURL, page, n)` and `FetchCollectionCursor(app, ctx, collectionURL, cursor)`. `collectionLink` and `FetchCollectionPage` are unchanged. The cursor must be absolute, non-opaque, without userinfo, and match the collection's scheme, host (incl. port) and escaped path. The request uses the parsed URL's `String()`.
- **Routes:** `RemoteProfileFollowsList` uses the cursor path when `cursor` is set, answers an invalid cursor with 400, and always returns a `next` key (empty string on the last page). `ActivitypubActorFollow` does the same and rewrites `collection.Next` to the resolved URL (nil on the last page).
- **Web:** `profile_follows_index(..., cursor?)` sends `cursor` when non-empty. The followers/following page decides has-more by `next` when the key exists (else `page < totalPages`), and advances `pagination.page` only after a successful fetch. Swagger JSDoc lists `cursor`.
- **Flutter:** `ProfileFollowsState` gained `next` and `cursorPaging`. `hasMore` is `cursorPaging ? next != null : page < totalPages`. `appendPage` sends the cursor and page.

## Decisions

- **FetchCollectionCursor signature:** it takes no page number `n` (D-02 said "e.g."). `CollectionNext` takes the page number, because it is only needed when resolving the following link, and it serves both fetch paths. Every returned `next` passes through `collectionLink`, so it always passes cursor validation.
- Used `'cursor': ?cursor` (null-aware map element) in the Dart query map instead of `if (cursor != null)`, because the project's `use_null_aware_elements` lint flags the latter. SDK is ^3.11.5, so the syntax is supported.

## Verification

- Go (`GOTOOLCHAIN=go1.26.1`): `go test ./federation/ ./routes/ -count=1` passes; `go vet` clean. New tests: one request per cursor page in all four server styles (mastodon, gotosocial, wanderer, wanderer-old) with the same items as the walk and totalItems 7, `next` empty after the last page, zero requests for all seven rejected cursors (host, path, scheme, relative, port, userinfo, escaped path), and `CollectionNext` empty cases. The existing `TestFetchCollectionPage*` tests stay green.
- Web: `svelte-kit sync` and `svelte-check --threshold error`: 0 errors, 0 warnings.
- Flutter: the new `profile_follows_cursor_test.dart` (3 tests) and the unchanged `paged_load_more_test.dart` pass. Full `flutter test` passes (1191 tests, 1 skipped). `flutter analyze` reports 13 issues, all info-level and pre-existing in other files (vendor/tiptap_flutter, entities, etc.); none are in the touched files.
- `git status --short -- db web` in APP_WT is empty.

## Deviations from Plan

**1. [Rule 3 - Blocking] Unrelated generated file reverted.** `dart run build_runner` also rewrote `app/lib/provider/welcome/recent_servers_provider.g.dart`; it was reverted with `git checkout` so the app commit stays scoped. The `.g.dart` of the follows provider did change (hash line) and is committed.

Otherwise the plan was executed as written.

## Known Stubs

None.

## Threat Flags

None. The new cursor input surface is the one covered by T-s3w-01 to T-s3w-03 and is mitigated and tested.

## Self-Check: PASSED

- Commits 9361ee36 and 1303a41e on fix/federation-cursor-paging, eeac540a on feature/app: found in git log.
- New test files exist: db/federation/collection_test.go (extended), app/test/provider/profile_follows_cursor_test.dart.
