---
title: Plugins
description: How to set up third-party provider plugins with wanderer.
---

Plugins add optional functionality that is not built into the core application.
Once an administrator has installed a plugin, it appears in the plugin settings
page where users can configure and enable it.

Plugin installation and self-hosted connector trust settings are administrator
tasks. See [Plugin installation](/run/installation/plugins) for runtime bundle
and connector details.

## Strava Plugin

:::caution[A Strava subscription is required]
With Strava's June 2026 Developer Program update, accessing the Strava API as a
"Standard Tier" developer requires an active Strava subscription. Because each
<span class="-tracking-[0.075em]">wanderer</span> user connects with their own
Client ID and Client Secret, everyone using this plugin counts as a Standard
Tier developer and is subject to this requirement.

- **New developers:** subscription required since **June 1, 2026**.
- **Existing developers:** subscription required from **June 30, 2026**.
- Active developers without a subscription are granted **3 months free** to
  transition — redeem the offer from your
  [Strava API settings dashboard](https://www.strava.com/settings/api).

Your personal data export and device/wearable integrations are **not** affected;
only programmatic API access is. A free (non-subscriber) Strava account can no
longer use this plugin once the transition period ends. For details see Strava's
[Developer Program update](https://communityhub.strava.com/insider-journal-9/an-update-to-our-developer-program-13428)
and [API FAQ](https://communityhub.strava.com/developers-knowledge-base-14/strava-api-faq-12906).
:::

### Creating an App in Strava

Before integrating Strava with <span class="-tracking-[0.075em]">wanderer</span>, you need to create an API application in Strava. Visit [Strava's API settings](https://www.strava.com/settings/api) and follow the steps to create a new API application. Your setup should resemble the following:

![Strava API Application](../../../assets/guides/strava_api_app.png)

### Setting Up the Plugin

1. Copy the **Client ID** and **Client Secret**.
2. Go to the plugins page in <span class="-tracking-[0.075em]">wanderer</span>'s settings.
3. Click the settings button for the Strava plugin.
4. Enter your **Client ID** and **Client Secret**.
5. Choose whether you want to sync routes, activities, or both.

![wanderer Strava Plugin](../../../assets/guides/wanderer_integration_strava.png)

6. Click **Save & connect**.
7. You will be redirected to Strava's authorization page. Keep all checkboxes selected and click **Authorize**.
8. You will then be redirected back to <span class="-tracking-[0.075em]">wanderer</span>.
9. Toggle the plugin on. It is now active.

If you later change the Client ID or Client Secret, reconnect the plugin. Other
settings can be saved without repeating the OAuth flow.

## komoot Plugin

The komoot plugin requires only your komoot username and password:

1. Open the komoot settings from the plugins menu.
2. Enter your komoot credentials.
3. Save the settings.
4. Toggle the plugin on. It will become active immediately.

Your planned and completed trails will now sync with <span class="-tracking-[0.075em]">wanderer</span>.

## Hammerhead Plugin

The Hammerhead plugin requires your Hammerhead account details:

1. Open the Hammerhead settings from the plugins menu.
2. Enter your Hammerhead email and password.
3. Choose whether you want to sync planned tours, completed tours, or both.
4. (Optional) Set an "ignore trails before" date to avoid syncing duplicates if your Hammerhead account is already connected to other services.
5. Save the settings and toggle the plugin on. It will become active immediately after a successful login.

:::note
This page still describes provider setup at a high level. Provider-specific details depend on the installed plugin's manifest and capabilities.
:::
