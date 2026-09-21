---
title: Custom map tiles & assets
description: How to configure where the mobile app loads map tiles, fonts, and sprites from
---

When the <span class="-tracking-[0.075em]">wanderer</span> mobile app renders an online map, it does not hardcode any tile server. Instead it asks your instance where to load map data from via the `/api/v1/map/style-sources` endpoint, which returns three URLs:

- **Tiles** — the vector tile template the base map is rendered from
- **Glyphs** — the fonts used for map labels (place names, street names)
- **Sprites** — the icon set used for map symbols (route shields, POI icons)

This gives you, the instance operator, full control over the map stack your app users consume — including pointing everything at self-hosted services.

## Defaults

Without any configuration, tiles are loaded from the [Protomaps API](https://protomaps.com/) and glyphs/sprites from Protomaps' public asset host. The Protomaps API requires an API key:

| Environment Variable | Description                                                                 | Default                                     |
| -------------------- | --------------------------------------------------------------------------- | ------------------------------------------- |
| PROTOMAPS_API_KEY    | API key for the Protomaps tile API; get one at [protomaps.com](https://protomaps.com/) | |
| TILE_SERVER_URL      | Vector tile URL template; overrides the Protomaps API entirely when set     | Protomaps API                               |
| MAP_ASSETS_URL       | Base URL for glyphs and sprites                                             | https://protomaps.github.io/basemaps-assets |

All three variables are set on the `web` service.

## Self-hosting tiles

Set `TILE_SERVER_URL` to any tile server that serves [Protomaps basemap](https://docs.protomaps.com/basemaps/downloads)-flavored vector tiles, using `{z}/{x}/{y}` placeholders:

```yaml
services:
  web:
    environment:
      TILE_SERVER_URL: https://tiles.example.com/tiles/{z}/{x}/{y}.mvt
```

When `TILE_SERVER_URL` is set, `PROTOMAPS_API_KEY` is not used.

## Self-hosting glyphs & sprites

Set `MAP_ASSETS_URL` to a host serving the [basemaps-assets](https://github.com/protomaps/basemaps-assets) directory layout. The app expects fonts under `<MAP_ASSETS_URL>/fonts/{fontstack}/{range}.pbf` and sprites under `<MAP_ASSETS_URL>/sprites/v4/`:

```yaml
services:
  web:
    environment:
      MAP_ASSETS_URL: https://assets.example.com/basemaps-assets
```

The simplest way to self-host these is to clone the `basemaps-assets` repository and serve it with any static file server.

:::note
These settings affect the *online* map in the mobile app. Offline maps come from downloaded [region archives](/run/backend-configuration/region-catalogue) and cached assets instead and work independently of these variables.
:::
