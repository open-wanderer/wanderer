# Deferred Items — Phase 17 (Navigation on MapLibre)

Pre-existing `flutter analyze` findings, out of scope for 17-02 (none touch files
this plan modified: `trail_layer.dart`, `trail_detail_map_screen.dart`,
`map_screen.dart`, `tracelet_position_source.dart`, `map_compass.dart`,
`pm_tile_provider.dart`). All are warnings/info, zero errors.

- `lib/components/trail/trail_dropdown.dart:126` — dead_code warning
- `lib/routes/settings_categories_screen.dart:551` — use_build_context_synchronously info
- `lib/routes/settings_subcategories_screen.dart:523` — use_build_context_synchronously info
- `lib/util/icon_util.dart` — ~30 deprecated_member_use info (Font Awesome icon renames)
- `test/models/feed_item_test.dart:3` — unused_import warning
