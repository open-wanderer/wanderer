---
title: Region catalogue
description: How to offer downloadable offline map regions to mobile app users
---

The region catalogue lets users of the <span class="-tracking-[0.075em]">wanderer</span> mobile app download entire map regions — for example a country or a state — for fully offline use: browsing, recording, and navigating trails without any network connection.

As an instance administrator you decide which regions your server offers. For every *enabled* region, the backend pre-builds two [PMTiles](https://docs.protomaps.com/pmtiles/) archives and serves them to logged-in app users:

- a **vector archive** with the base map (streets, terrain features, labels), extracted from the [Protomaps daily build](https://build.protomaps.com) and clipped to the region's boundary polygon
- a **DEM archive** with elevation data for hillshading and elevation profiles, extracted from [Mapterhorn](https://mapterhorn.com)

:::note
The region catalogue is only useful in combination with the mobile app. If none of your users use the app, you can ignore this page entirely — nothing is built or downloaded until you enable a region.
:::

## The regions collection

Available regions come from a curated hierarchy (based on the [CoMaps](https://codeberg.org/comaps/comaps) region definitions) that ships with the backend and is seeded automatically into the `regions` collection on first start. The hierarchy covers the whole world, split into pieces of manageable size — larger countries are subdivided (e.g. `germany.bavaria`), smaller ones are a single region.

All regions start out **disabled**. Your instance only builds and serves archives for regions you explicitly enable.

## Managing regions

Open the region catalogue admin page at:

```
https://<your-instance>/region-catalog/
```

The page requires you to be logged in to the PocketBase dashboard as a superuser.

From here you can:

- **Enable / disable regions** — browse the hierarchy and toggle the regions your instance should offer. Enabling a region immediately caches its boundary geometry; the archives themselves are built by the next sync.
- **Sync now** — trigger an immediate build pass instead of waiting for the nightly schedule. The page shows whether a sync is currently running and when the next scheduled one will start.
- **Delete archives** — remove a region's built archives from disk, e.g. after disabling it.

## Builds & scheduling

A build pass runs automatically every night at 03:00 UTC and rebuilds archives that are outdated — because a newer Protomaps daily build is available, or because the region definition changed. While a region is being rebuilt, the previous archive keeps being served; the new file replaces it atomically only on success.

Two environment variables on the `db` service control the process:

| Environment Variable           | Description                                                              | Default    |
| ------------------------------ | ------------------------------------------------------------------------ | ---------- |
| REGION_ARCHIVE_CRON_SCHEDULE   | Cron expression (UTC) for the nightly build pass                         | 0 3 * * *  |
| REGION_ARCHIVE_EXTRACT_TIMEOUT | Maximum duration for a single archive extraction (Go duration string)    | 30m        |

Archives are stored in `pb_data/region_archives/`, one folder per region containing `vector.pmtiles` and `dem.pmtiles`. If you delete files there manually, the backend reconciles its records with the files on disk at the next start.

:::caution
Region archives can be large — from tens of megabytes for a small region to several gigabytes for large, densely mapped ones. Make sure the volume backing `pb_data` has enough free space before enabling many regions, and consider excluding `pb_data/region_archives` from your backups: the archives can always be rebuilt.
:::

Each build downloads only the enabled region's clipped extract (not the full planet file) from the upstream sources, but a region covering a whole country can still take a while — increase `REGION_ARCHIVE_EXTRACT_TIMEOUT` if builds of very large regions fail with a timeout.

## Troubleshooting

- **A region shows an error status** — check the `db` service logs for `[regions]` entries. Typical causes are an upstream download failure or a timeout; the next sync (or "Sync now") retries automatically.
- **Downloads require login** — region listing and archive downloads are only available to authenticated users. Anonymous visitors cannot download archives.
- **Re-seeding the catalogue** — the `seed-regions` CLI command of the `db` binary regenerates the seeded hierarchy from a pinned upstream commit. You normally never need to run it; it exists for maintainers updating the shipped catalogue.
