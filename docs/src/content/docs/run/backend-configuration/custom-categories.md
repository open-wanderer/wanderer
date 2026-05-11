---
title: Custom categories
description: How to create custom trail categories
---

<span class="-tracking-[0.075em]">wanderer</span> uses categories to classify what kind of activity a trail belongs to. 
Out of the box you get: Biking, Canoeing, Climbing, Hiking, Skiing and Walking. 
However, you can adapt these categories to your needs or add completely new ones.

## Modifying categories

![Pocketbase Categories](../../../../assets/guides/pocketbase_categories.png)

In the PocketBase admin panel, click on the `categories` table in the list on the left side. 
All existing categories will be listed here. 
To edit one simply click on the row, edit the data you want to change, and click "Save". 
To delete a category check the box at the beginning of the row and click "Delete selected". 
To create a new category click the "New record" button in the top right corner, give your new category a name and a background image, and click "Save".

## Category settings

Categories can optionally define additional settings in the `settings` JSON field.
This field may be left empty.
When no settings are configured, <span class="-tracking-[0.075em]">wanderer</span> uses the built-in defaults.

Currently, the following setting is supported:

```json
{
  "wp_merge_enabled": true,
  "wp_merge_radius": 50
}
```

`wp_merge_enabled` controls whether geotagged photos are grouped into waypoint clusters.
Set it to `false` to create one waypoint per photo.

`wp_merge_radius` controls how close geotagged photos have to be to each other, in meters, before they are grouped into the same waypoint when adding waypoint photos to a trail.
Set it to `0` to only merge photos with the exact same coordinates, or increase the value to merge photos across a wider area.
