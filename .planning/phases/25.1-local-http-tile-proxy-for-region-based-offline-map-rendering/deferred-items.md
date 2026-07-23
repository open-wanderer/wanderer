# Deferred Items — Phase 25.1

## 25.1-02: Pre-existing generated-file drift (out of scope, not committed)

Running `dart run build_runner build --delete-conflicting-outputs` for Task 2's
`tile_proxy_provider.g.dart` codegen also regenerated
`app/lib/provider/region/tile_repository_provider.g.dart` — its committed
version was stale relative to `tile_repository_provider.dart`'s own
already-committed doc comments (a doc-comment-only diff, no behavior change).
This file is not in this plan's `files_modified` list, so the regenerated
output was left uncommitted in the working tree rather than folded into a
25.1-02 task commit (SCOPE BOUNDARY — only auto-fix issues directly caused by
the current task's own changes). A future plan touching
`tile_repository_provider.dart`/`.g.dart` should pick this up naturally via
its own codegen step.
