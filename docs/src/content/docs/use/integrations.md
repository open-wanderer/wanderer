---
title: Integrations
description: How to set up third-party integrations with wanderer.
---

You can automatically sync trails to <span class="-tracking-[0.075em]">wanderer</span> at regular intervals using the third-party integration feature. Currently, we support three providers: **Strava**, **komoot** and **hammerhead**.

It is important to note that synchronization only works from the provider to <span class="-tracking-[0.075em]">wanderer</span> and not the other way around. Additionally, if a trail has already been synced to <span class="-tracking-[0.075em]">wanderer</span>, subsequent changes made in the provider will not be transferred unless the trail is deleted in <span class="-tracking-[0.075em]">wanderer</span>. Hammerhead also supports manual uploads from a trail's action menu, which is separate from the nightly sync.

## Strava Integration

### Creating an App in Strava

Before integrating Strava with <span class="-tracking-[0.075em]">wanderer</span>, you need to create an API application in Strava. Visit [Strava's API settings](https://www.strava.com/settings/api) and follow the steps to create a new API application. Your setup should resemble the following:

![Strava API Application](../../../assets/guides/strava_api_app.png)

### Setting Up the Integration

1. Copy the **Client ID** and **Client Secret**.
2. Go to the integrations page in <span class="-tracking-[0.075em]">wanderer</span>'s settings.
3. Click the settings button for the Strava integration.
4. Enter your **Client ID** and **Client Secret**.
5. Choose whether you want to sync routes, activities, or both.

![wanderer Strava Integration](../../../assets/guides/wanderer_integration_strava.png)

6. Save the settings and toggle the integration on.
7. You will be redirected to Strava's authorization page. Keep all checkboxes selected and click **Authorize**.
8. You will then be redirected back to <span class="-tracking-[0.075em]">wanderer</span>. The Strava integration is now active.

## komoot Integration

The komoot integration requires only your komoot username and password:

1. Open the komoot settings from the integrations menu.
2. Enter your komoot credentials.
3. Save the settings.
4. Toggle the integration on. It will become active immediately.

Your planned and completed trails will now sync with <span class="-tracking-[0.075em]">wanderer</span>.

## Hammerhead Integration

The Hammerhead integration requires your Hammerhead account details:

1. Open the Hammerhead settings from the integrations menu.
2. Enter your Hammerhead email and password.
3. Choose whether you want to sync planned tours, completed tours, or both.
4. (Optional) Set an "ignore trails before" date to avoid syncing duplicates if your Hammerhead account is already connected to other services.
5. Save the settings and toggle the integration on. It will become active immediately after a successful login.

## Sync Interval

By default, trails are synced every night at **02:00 AM**. You can modify this schedule using the `POCKETBASE_CRON_SYNC_SCHEDULE` [environment variable](/run/environment-configuration#pocketbase).

:::note
Please set a reasonable sync interval. Both Strava and komoot impose usage limits on their APIs. Exceeding these limits may result in rejected requests or account suspension.
:::
