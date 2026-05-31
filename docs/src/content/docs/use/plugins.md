---
title: Plugins
description: How to set up third-party provider plugins with wanderer.
---

Third-party providers are provided by locally installed WASM plugins. First-party
plugins are bundled in the wanderer repository under `plugins/`. Runtime plugin
bundles live as direct child directories below `data/plugins` and are shown in
the plugin settings once their `plugin.json` manifest has been discovered.

Plugin capabilities decide which actions are available. A provider may support route listing, activity listing, media import, or sending a route from a trail's action menu.

There is no built-in plugin store. Install plugins by downloading a plugin
bundle from the GitHub release assets, extracting it, and copying the extracted
plugin directory into `data/plugins`.

Community plugins can be discovered through the project's public plugin list,
for example a GitHub discussion maintained outside the application.

Plugins with username/password or API-key auth can be saved and enabled
directly. OAuth plugins have a separate connection step: save the settings,
complete the provider authorization flow, and then enable the plugin.

## Strava Plugin

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
