---
title: Trail plugins
description: Configure trail synchronization, imports, category mappings, and trail sending
---

Trail plugins connect Wanderer to services that store planned routes or
completed activities. Depending on their capabilities, they can import planned
trails, import completed activities, include provider photos and waypoints, or
send a Wanderer trail back to the provider.

## Common trail plugin settings

Open **Settings → Plugins**, select a trail plugin, and configure the options it
exposes. The available controls depend on its manifest and capabilities.

### Planned and completed trails

Plugins that distinguish between planned routes and completed activities let
you enable either source independently. Disable a source when you do not want
that kind of provider item imported.

Some plugins also offer a start date. Use it to ignore older provider items,
reduce the initial import, or avoid duplicates when older trails already exist
in Wanderer.

### Privacy

When a provider exposes source privacy, a plugin can offer two behaviors:

- preserve the visibility of the provider item; or
- discard the provider visibility and apply your Wanderer trail privacy
  settings.

Review this setting before the first synchronization, especially if your
provider contains public activities.

### Category mapping

Trail plugins can report provider-specific activity categories. Their settings
let you map those values to Wanderer categories and subcategories, for example a
provider gravel activity to **Biking / Gravel**.

Changing a mapping can affect trails imported earlier. When applicable,
Wanderer offers to update the category of affected imported trails without
changing their other data.

### Automatic merging

When **Auto-merge** is enabled, an imported trail is merged only when Wanderer
finds exactly one clear existing match. This can reduce duplicates when the same
activity is already present from another source.

### Sending trails

Plugins with trail-send support appear in the **Send to…** dialog on a trail.
Only connected and enabled plugins that advertise the required capability are
listed.

## Strava

The Strava plugin imports routes, activities, and their available photos.

:::caution[A Strava subscription is required]
With Strava's June 2026 Developer Program update, accessing the Strava API as a
"Standard Tier" developer requires an active Strava subscription. Because each
Wanderer user connects with their own Client ID and Client Secret, everyone
using this plugin counts as a Standard Tier developer and is subject to this
requirement.

- **New developers:** subscription required since **June 1, 2026**.
- **Existing developers:** subscription required from **June 30, 2026**.
- Active developers without a subscription are granted **3 months free** to
  transition — redeem the offer from your
  [Strava API settings dashboard](https://www.strava.com/settings/api).

Your personal data export and device or wearable integrations are not affected;
only programmatic API access is. For details see Strava's
[Developer Program update](https://communityhub.strava.com/insider-journal-9/an-update-to-our-developer-program-13428)
and [API FAQ](https://communityhub.strava.com/developers-knowledge-base-14/strava-api-faq-12906).
:::

### Create an application in Strava

Visit [Strava's API settings](https://www.strava.com/settings/api) and create an
API application. Your setup should resemble the following:

![Strava API Application](../../../../assets/guides/strava_api_app.png)

### Connect the plugin

1. Copy the **Client ID** and **Client Secret**.
2. Open **Settings → Plugins** and select Strava.
3. Enter the Client ID and Client Secret.
4. Choose whether to synchronize routes, activities, or both.

![Wanderer Strava plugin](../../../../assets/guides/wanderer_integration_strava.png)

5. Select **Save & connect**.
6. On Strava's authorization page, keep all required permissions selected and
   select **Authorize**.
7. After returning to Wanderer, enable the plugin.

If you later change the Client ID or Client Secret, reconnect the plugin. Other
settings can be saved without repeating the OAuth flow.

## komoot

The komoot plugin imports planned and completed tours, including available
photos and waypoints.

1. Open **Settings → Plugins** and select komoot.
2. Enter your komoot email address and password.
3. Choose the sources and optional start date offered by the plugin.
4. Save the settings and enable the plugin.

## Hammerhead

The Hammerhead plugin imports planned routes and completed activities and can
send Wanderer trails to Hammerhead.

1. Open **Settings → Plugins** and select Hammerhead.
2. Enter your Hammerhead email address and password.
3. Choose whether to synchronize planned tours, completed tours, or both.
4. Optionally set a start date to avoid importing older duplicates.
5. Save the settings and enable the plugin.

Once connected and enabled, Hammerhead is also available in the **Send to…**
dialog for compatible trails.
