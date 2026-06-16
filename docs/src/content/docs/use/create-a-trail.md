---
title: Create or edit a trail
description: How to create or edit a trail by uploading or drawing a trail using Valhalla
---

## What is a trail?

In <span class="-tracking-[0.075em]">wanderer</span>, a trail is a digital route that includes GPS data and descriptive metadata like name, difficulty, category, photos, and waypoints. Trails can be explored by others and searched in the app.

## Create a trail

To start, click the  <button class="h-10 text-white rounded-lg px-4 py-2 mx-2 bg-primary font-semibold transition-all hover:bg-primary-hover focus:ring-4 ring-zinc-400 leading-none">+ New Trail</button> button in the top right corner.

### Step 1: Pick a route

Each trail must begin with a route. There are two ways to provide one:

#### Upload a file

Click the **Upload file** button to select a GPS file. Accepted formats are **GPX**, **FIT**, **TCX**, or **KML**.

After uploading:

- The map centers on the route
- Elevation profile and speed (if available) are rendered
- Distance, elevation gain/loss, and other metadata are extracted
- The form fields on the left will be partially prefilled with data extracted from the file

#### Draw a route

Click the **Draw a route** button to manually define a route on the map. While in drawing mode:

- Click on the map to place route anchors
- <span class="-tracking-[0.075em]">wanderer</span> will automatically route between anchors using the [Valhalla routing engine](https://github.com/valhalla/valhalla)
- You can drag anchors to reposition them
- Use the top-left menu to change routing mode (e.g. walking, cycling)
- Use the waypoint button (location marker icon) in the route editing toolbar to show or hide existing waypoint markers on the map. Hiding them also disables right-clicking to create new waypoints.
- To remove an anchor, click on it and then click the red trash icon

If you disable Valhalla routing, straight lines will be used between anchors instead.

To finish drawing, click **Stop drawing**.

:::tip
<span class="-tracking-[0.075em]">wanderer</span> uses a public, donation-financed Valhalla server by default. Please consider supporting it at [https://www.fossgis.de/verein/spenden/](https://www.fossgis.de/verein/spenden/).
:::

#### Trail anchor list

While drawing or editing a route, the anchor list shows the route's start, intermediate, and finish anchors in order. It helps you inspect and adjust the structure of the route without relying only on the map:

- Each segment shows distance and elevation stats for the route section that follows the anchor.
- Hover an anchor in the list to highlight its marker on the map.
- Drag intermediate anchors in the list to reorder the route sequence.
- Delete an anchor from the list to remove it from the route.

### Step 2: Fill out trail details

#### Basic Info

- **Name** – Required. Every trail needs a name.
- **Location** – Autofilled if available in the uploaded file.
- **Date** – Defaults to today.
- **Description** – Use the editor to describe your trail in as much detail as you want.
- **Distance / Duration / Elevation** – These are automatically calculated but can be manually adjusted if needed.
- **Tags** – Add descriptive tags to help categorize and search for your trail (e.g. forest, sunset, dog-friendly). Start typing to add a tag and press Enter to confirm.
- **Difficulty** – Select the trail's difficulty (e.g. Easy, Moderate, Hard)
- **Category** – Choose the activity type (e.g. Hiking, Cycling)

#### Visibility

Toggle the **Private** switch if you do not want the trail to be visible to others. When set to private, only you will be able to view and access this trail.

:::note
Creating a public trail will automatically publish that trail to all your followers.
:::

### Step 3: Add Waypoints

Waypoints are points of interest along the trail.

- Click **+ Add Waypoint** to add one manually. It will appear centered on the map and can be dragged to another location.
- When you are not editing the route, click on the map to open a popup with a **Create waypoint** button at that location.
- While drawing or editing the route, right-click on the map to open the same popup without placing a route anchor. This works only while waypoint markers are visible (see the waypoint toggle in the route editing toolbar).
- Each waypoint can have a name, description, icon, and photos.
- Use Font Awesome icons for map markers. You can browse them at [fontawesome.com](https://fontawesome.com/search?q=share&o=r&m=free).

Alternatively, click **From Photos** to upload photos with GPS metadata. Waypoints will be created automatically based on the photo locations.

### Step 4: Add Photos & Videos

You can attach photos and videos to the trail itself. These will be shown in the trail's detail view. If you upload more than one, you can select one to be the trail’s thumbnail in the overview.

### Step 5: Add to Summit Book

If you've completed this trail yourself, you can log a summit book entry.

- Click **+ Add Entry**
- Upload a separate GPS file or just log the date of your completion
- You can add multiple summit entries over time without creating duplicate trails

To learn more about summit logs visit the [dedicated section](/use/summit-logs) of the documentation.

### Step 6: Save the trail

When you're done, click   <button class="h-10 text-white rounded-lg px-4 py-2 mx-2 bg-primary font-semibold transition-all hover:bg-primary-hover focus:ring-4 ring-zinc-400 leading-none">Save Trail</button> to persist your trail to the database. This will also re-index it for search and display it in your trail list.

## Edit an existing trail

Open a trail and choose **Edit** from its menu. If the trail already has route data, the editor starts in drawing mode so you can continue adjusting the existing route immediately.

Route editing works like drawing a new route, with a few differences:

- Click **Stop editing** to leave drawing mode. This protects the route from accidental changes when you click the map and restores left-clicking the map to create waypoints. Metadata, photos, waypoints, summit entries, and lists can be edited regardless of whether drawing mode is active.
- Click **Edit route** to return to route editing mode later.
- Existing waypoints remain visible while editing the route. Use the waypoint toggle in the route editing toolbar to hide them if they get in the way (this also disables right-click waypoint creation).
- Left-click the map to add or extend route anchors.
- Right-click the map to create a waypoint at that location without placing a route anchor.
- Use **Reset route** if you want to discard the current route and upload or draw a new one.

Saving an edited trail updates its route data, metadata, waypoints, photos, and search index.
