---
title: Import/Export
description: How to import and export trails in wanderer
---

## Import

<span class="-tracking-[0.075em]">wanderer</span> supports bulk uploading of trails via an auto-upload folder. A file watcher automatically detects new files added to this directory and imports them into your library.

:::note
This feature is currently only available for Docker installations. Files added to the folder while the container is not running are ignored.
:::

:::caution
Successfully uploaded files will be deleted from the auto-upload folder.
:::

### Configuration

#### Environment variables

The following environment variable must be present in the `wanderer-web` docker container and set to a valid volume path (see below).

| Environment Variable | Description                    | Default      |
| -------------------- | ------------------------------ | ------------ |
| UPLOAD_FOLDER        | Path to the auto-upload folder | /app/uploads |


#### Volume
Make sure to mount the upload folder as a volume to your host system. The default `docker-compose.yml` already includes this volume. Ensure that the mapped value matches the one in the `UPLOAD_FOLDER` environment variable.

#### API token
The bulk upload process uses API tokens to authenticate requests and determine which user account the uploaded trails should be assigned to.

1. Create an API token: Follow the steps in the [Authentication section](/use/authentication/#api-tokens) to generate a new API token.
2. Prepare the folder structure: Create the folder: Inside your UPLOAD_FOLDER, create a sub-folder named exactly after your API token.
3. Upload: Move your trail files (e.g., .gpx, .fit, or .kml) into that sub-folder.

**Example structure**:
    `/app/uploads/wanderer_key_<...>/my_trail.gpx`


## Export

To export selected trails head over to `/trails` and select the trails you want to export. From the <span class="inline-block w-8 h-8 bg-primary rounded-full text-center text-white">⋮</span> menu select "Export". You can export the route data either in GPX or in GeoJSON format. Furthermore, you can choose whether you want to include the photos and the summit book of the trail. In any case, <span class="-tracking-[0.075em]">wanderer</span> will create a ZIP archive with all the data that is then downloaded.

You can also export all of your trails at once. To do so, head over to `/settings/export` and click "Export all trails". The other steps remain analogous to exporting a single trail.
