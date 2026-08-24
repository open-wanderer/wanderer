---
title: Asset plugins
description: Find, import, and link photos from external media libraries
---

Asset plugins connect Wanderer to external media libraries. They can search for
geotagged photos that match a trail or waypoint and import the selected results
as Wanderer photo assets.

## Where asset plugins are used

An enabled asset plugin can provide photo candidates in several workflows:

- after a completed trail is imported by a trail plugin;
- after a time-stamped GPS track is uploaded;
- while editing a trail, to create photo waypoints;
- while editing a waypoint that has coordinates.

Candidate matching is based on information such as track time and geographic
distance. The exact search controls depend on the plugin.

## Photo storage modes

Compatible asset plugins can offer two storage modes:

| Mode | Behavior |
| --- | --- |
| Store photos in Wanderer | Downloads selected files and stores them as local Wanderer assets. |
| Link remote references | Keeps private references to provider assets and fetches files on demand. |

Remote links can be used only for private content. If a trail becomes public,
Wanderer materializes its linked photos locally because anonymous viewers cannot
access private provider media.

If you disable an asset plugin while linked photos still exist, Wanderer asks
whether it should download the photos or remove the links first.

## Automatic attachment and limits

Depending on the plugin metadata, its settings can include:

- automatic attachment after trail-plugin imports;
- automatic attachment after GPS uploads;
- the maximum number of generated photo waypoints;
- the maximum number of photos per waypoint or trail;
- the imported image size.

Automatically generated photo waypoints are merged according to the trail
category's waypoint merge radius. Wanderer names new waypoints using nearby
OpenStreetMap points of interest, then reverse geocoding, and finally the photo
coordinate as fallbacks.

## Immich

The first-party Immich plugin searches an Immich library for geotagged photos
that match a trail or waypoint.

### Create an Immich API key

Create an API key in your Immich account settings and grant only these
permissions:

| Permission | Used for |
| --- | --- |
| `user.read` | Validate the key and identify your user for **Only my photos**. |
| `asset.read` | Search geotagged photos and read selected asset metadata. |
| `asset.view` | Load thumbnail previews. |
| `asset.download` | Download selected originals or previews. |

The plugin does not need write, delete, upload, album, library, or administrator
permissions.

### Configure the plugin

1. Open **Settings → Plugins** and select Immich.
2. Enter the Immich server URL and API key.
3. Adjust the time window, search radius, maximum waypoints, import size, and
   **Only my photos** setting as needed.
4. Choose the photo storage mode and automatic attachment options.
5. Save the settings and enable the plugin.

The Wanderer backend must be able to resolve and reach the Immich URL. Access
from your desktop browser alone is not sufficient. Administrators configuring a
self-hosted or private endpoint should also review
[Plugin installation](/run/installation/plugins/#self-hosted-asset-plugins).
