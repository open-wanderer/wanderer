---
title: Create a trail
description: How to create a trail by uploading or drawing a trail using Valhalla
---

## What is a trail?

In <span class="-tracking-[0.075em]">wanderer</span>, a trail is a digital route that includes GPS data and descriptive metadata like name, difficulty, category, photos, and waypoints. Trails can be explored by others and searched in the app.



## Create a trail

To start, click the  <button class="h-10 text-white rounded-lg px-4 py-2 mx-2 bg-primary font-semibold transition-all hover:bg-primary-hover focus:ring-4 ring-zinc-400 leading-none">+ New Trail</button> button in the top right corner.



## Step 1: Pick a route

Each trail must begin with a route. There are two ways to provide one:

### Upload a file

Click the **Upload file** button to select a GPS file. Accepted formats are **GPX**, **FIT**, **TCX**, or **KML**.

After uploading:

- The map centers on the route
- Elevation profile and speed (if available) are rendered
- Distance, elevation gain/loss, and other metadata are extracted
- The form fields on the left will be partially prefilled with data that extracted from the file

### Draw a route

Click the **Draw a route** button to manually define a route on the map. While in drawing mode:

- Click on the map to place waypoints
- <span class="-tracking-[0.075em]">wanderer</span> will automatically route between points using the [Valhalla routing engine](https://github.com/valhalla/valhalla)
- You can drag points to reposition them
- The anchor list next to the map shows start, intermediate, and finish points with segment distance and elevation stats
- Hover an item in the anchor list to highlight its marker on the map
- Reorder intermediate anchors from the list to adjust the route sequence
- Use the top-left menu to change routing mode (e.g. walking, cycling)
- To remove a point, click on it and then click the red trash icon

If you disable Valhalla routing, straight lines will be used between points instead.

To finish drawing, click **Stop drawing**.

:::tip
<span class="-tracking-[0.075em]">wanderer</span> uses a public, donation-financed Valhalla server by default. Please consider supporting it at [https://www.fossgis.de/verein/spenden/](https://www.fossgis.de/verein/spenden/).
:::



## Step 2: Fill out trail details

### Basic Info

- **Name** – Required. Every trail needs a name.
- **Location** – Autofilled if available in the uploaded file.
- **Date** – Defaults to today.
- **Description** – Use the editor to describe your trail in as much detail as you want.
- **Distance / Duration / Elevation** – These are automatically calculated but can be manually adjusted if needed.
- **Tags** – Add descriptive tags to help categorize and search for your trail (e.g. forest, sunset, dog-friendly). Start typing to add a tag and press Enter to confirm.
- **Difficulty** – Select the trail's difficulty (e.g. Easy, Moderate, Hard)
- **Category** – Choose the activity type (e.g. Hiking, Cycling)

### Visibility

Toggle the **Private** switch if you do not want the trail to be visible to others. When set to private, only you will be able to view and access this trail.

:::note
Creating a public trail will automatically publish that trail to all your followers.
:::


## Step 3: Add Waypoints

Waypoints are points of interest along the trail.

- Click **+ Add Waypoint** to add one manually. It will appear centered on the map and can be dragged to another location.
- Each waypoint can have a name, description, icon, and photos.
- Use Font Awesome icons for map markers. You can browse them at [fontawesome.com](https://fontawesome.com/search?q=share&o=r&m=free).

Alternatively, click **From Photos** to upload photos with GPS metadata. Waypoints will be created automatically based on the photo locations.



## Step 4: Add Photos & Videos

You can attach photos and videos to the trail itself. These will be shown in the trail's detail view. If you upload more than one, you can select one to be the trail’s thumbnail in the overview.



## Step 5: Add to Summit Book

If you've completed this trail yourself, you can log a summit book entry.

- Click **+ Add Entry**
- Upload a separate GPS file or just log the date of your completion
- You can add multiple summit entries over time without creating duplicate trails

To learn more about summit logs visit the [dedicated section](/use/summit-logs) of the documentation.



## Step 6: Save the trail

When you're done, click   <button class="h-10 text-white rounded-lg px-4 py-2 mx-2 bg-primary font-semibold transition-all hover:bg-primary-hover focus:ring-4 ring-zinc-400 leading-none">Save Trail</button> to persist your trail to the database. This will also re-index it for search and display it in your trail list.
