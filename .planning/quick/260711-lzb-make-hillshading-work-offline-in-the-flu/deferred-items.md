# Deferred Items — Quick Task 260711-lzb

Out-of-scope discoveries found during execution, not fixed (per executor scope boundary).

| Item | Detail |
|------|--------|
| Stale generated doc-comment IDs in 4 riverpod `.g.dart` files | Running `dart run build_runner build --delete-conflicting-outputs` (required by Task 2) regenerated `glyph_sprite_cache_provider.g.dart`, `map_style_json_provider.g.dart`, `navigation_stats_provider.g.dart`, and `trail/map_cluster_search_provider.g.dart` with doc-comment text that dropped stale planning-ID references (e.g. `(GLYPH-04, D-08)`, `(STYLE-02/03/04)`, `(D-17)`, `(CLUS-01/04/05)`) — a byproduct of a `riverpod_generator` version already installed but not previously re-run against these files. Unrelated to this task's `files_modified` scope, so reverted via `git checkout --` to keep the Task 2 commit scoped. No functional change; safe to regenerate and commit separately whenever these files are next touched. |
