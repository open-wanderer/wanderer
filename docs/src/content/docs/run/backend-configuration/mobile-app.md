---
title: Mobile app support
description: What your instance needs so users can connect the wanderer mobile app to it
---

The <span class="-tracking-[0.075em]">wanderer</span> [mobile app](/app) is in public beta. It talks to your instance through the same API the web interface uses, plus a handful of endpoints (region catalogue, health probe, navigation, and more) that are **not in the regular release yet**. Until the app is merged into the main release line, an instance only supports the app if it runs the `-app` builds.

## Run the `-app` images

Both images carry a matching `-app` tag next to every regular release, e.g. `v0.21.2-app`. Pin **both** services to the same tag; the `web` image hosts the API proxy the app relies on, so mixing a `-app` backend with a regular frontend does not work.

:::caution[Minimum version: v0.21.2-app]
Current app versions require **`v0.21.2-app` or newer** for both images. Earlier `-app` builds (`v0.21.0-app`, `v0.21.1-app`) do not publish the NodeInfo document the app uses to check compatibility, so the app treats them as unsupported and keeps sign-in disabled. If you set up the app with an older `-app` tag, update both services.
:::

```yaml
services:
  db:
    image: flomp/wanderer-db:v0.21.2-app
    # ...
  web:
    image: flomp/wanderer-web:v0.21.2-app
    # ...
```

Then pull and restart:

```sh
docker compose pull
docker compose up -d
```

There are no `-app` git tags. If you run from source, the app-compatible backend is on the `feature/app` branch; see [App development](/develop/app-development).

The `-app` builds contain everything the regular release of the same version does, plus the app endpoints and their migrations. Treat them like any other upgrade: take a [backup](/run/backend-configuration/backup-server) first.

## Reachable over HTTPS

The app refuses plain `http://` connections to anything but `127.0.0.1`. Your instance must be reachable over HTTPS with a certificate the phone trusts; self-signed certificates do not work unless they are installed on the device.

## A map tile source

The app's online map loads tiles from wherever your instance tells it to. By default that is the Protomaps API, which needs an API key: set `PROTOMAPS_API_KEY` on the `web` service, or point `TILE_SERVER_URL` at your own tile server. Without either, the app's online map stays blank. See [Custom map tiles & assets](/run/backend-configuration/map-tiles).

## Optional: offline map regions

The app can download map regions for offline use, but only the ones you enable. Nothing is built or served until you do; see [Region catalogue](/run/backend-configuration/region-catalogue). Without any enabled region, users can still download trails and navigate them offline, just without a base map.

## What users see when an instance isn't ready

As soon as users pick an instance, the app reads its NodeInfo document at `/.well-known/nodeinfo/2.1` and compares the reported version against the minimum it supports. If the instance runs an older version, or no `-app` build at all, the app shows **Server not supported** with a link to this page, and keeps login and registration disabled until the instance is updated. The [Getting started](/app/getting-started#choose-your-instance) page tells them to ask their administrator, which is you.

To check what the app sees, open `https://<your-instance>/.well-known/nodeinfo/2.1`: `software.version` must be `0.21.2` or newer. A 404 means the instance runs an image without NodeInfo.

Older app versions do not perform this check. On an unsupported instance, their login fails with `type 'List<dynamic>' is not a subtype of type 'Map<String, dynamic>' in type cast` instead.

If everything else works but the map is blank, the instance has no [map tile source](#a-map-tile-source) configured.

Users pick your instance from the public server list on wanderer.to or type its address; there is nothing to register on your side.
