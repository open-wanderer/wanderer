---
title: Mobile app support
description: What your instance needs so users can connect the wanderer mobile app to it
---

The <span class="-tracking-[0.075em]">wanderer</span> [mobile app](/app) is in public beta. It talks to your instance through the same API the web interface uses, plus a handful of endpoints (region catalogue, health probe, navigation, and more) that are **not in the regular release yet**. Until the app is merged into the main release line, an instance only supports the app if it runs the `-app` builds.

## Run the `-app` images

Both images carry a matching `-app` tag next to every regular release, e.g. `v0.21.0-app`. Pin **both** services to the same tag; the `web` image hosts the API proxy the app relies on, so mixing a `-app` backend with a regular frontend does not work.

```yaml
services:
  db:
    image: flomp/wanderer-db:v0.21.0-app
    # ...
  web:
    image: flomp/wanderer-web:v0.21.0-app
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

## Optional: offline map regions

The app can download map regions for offline use, but only the ones you enable. Nothing is built or served until you do; see [Region catalogue](/run/backend-configuration/region-catalogue). Without any enabled region, users can still download trails and navigate them offline, just without a base map.

## What users see when an instance isn't ready

The app does not check the instance version up front. On an instance without the `-app` images, users can log in, but the map stays empty and recording, navigation, file import, settings, and offline regions fail with errors. The [Getting started](/app/getting-started#choose-your-instance) page tells them to ask their administrator, which is you.

Users pick your instance from the public server list on wanderer.to or type its address; there is nothing to register on your side.
