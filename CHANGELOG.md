# v0.18.3
## Bug fixes
- Fixes bug that prevented waypoints from being update or deleted in v0.18.2
- Return properly formatted error message when accessing a protected API route
- Fixes logo for OIDC 2 & 3 providers (thanks @wolffshots)
- Fixes bug that caused dropdown actions from a feed item on the homepage to cause a 404 error

## Maintenance
- Updates pocketbase to v0.30.0

# v0.18.2
## Features
- Adds `dedup` command to pocketbase. This command allows an admin to quickly identify duplicate trails and delete them. Use the `--dry-run` flag to only log duplicate trails without deleting them. To execute the command run `docker exec -it wanderer-db ./pocketbase dedup --dry-run`.
- Adds option to only sync strava activities after a certain date
- Singificant performance improvements for instances with larger userbases
- Greatly improved initial indexing speed when starting wanderer
## Bug fixes
- Fixes permission issues for public trails
- Fixes bug that caused trails to be duplicated multiple times (to clean up see the `dedup` command above)
- Fixes link to "New Trail" from empty profiles
- Fixes link when opening a trail from the map searchbar
- Sorting by difficulty no longer sorts by difficulty alphabetically
- Fixes strava integration stopping after only one page

# v0.18.1
## Bug fixes
- Fixes permission issues that prevented federation from working properly
- Trail categories are properly displayed in lists
- Fixes issue that prevented lists from saving

## Documentation
- Improves docs for updating on bare-metal installations

## Translation
- Adds Basque and Russian translations (thanks @aisaivia & @jeffscrum)

# v0.18.0

## Features
- Adds feed on homepage for logged in users
- Adds customizable "About" section to the homepage (read [here](https://wanderer.to/run/frontend-configuration/about) to learn more)
- New Maps: OpenHikingMap, CyclOSM
- New map overlays: hiking, biking, mountain biking & skiing routes
- New POI overlays: e.g. grocery stores, campsites, water sources etc.
- Complete overhaul of the route drawing/editing feature
- Adds option to crop routes when editing
- Adds undo/redo to route editing
- Adds option to manually recalculate elevation data when editing a route
- Adds automated elevation correction when uploading trails/summit logs directly
- Directions to a trail head are now provided by OpenStreetMap
- Non-pulbic trails can now be shared with guests via a public share link
- Adds route previews on lower zoom levels in the world map
> [!NOTE]
To display the previews for each track on the world map, wanderer computes encoded polylines for each track on startup. Depending on the amount of trails in your instance, it can take several minutes until all trails are fully indexed and searchable.
- 3D terrain is now also available in the world map
- Trail category is now displayed in search results

## Bug fixes
- Custom categories with spaces are now handled correctly
- Avoid reuploading all trails when updating a list
- Fixes build dependencies for building from source (thanks @slothful-vassal)
- Add headers to nominatim requests to comply with usage policy
- Reset pagination when updating filters

## Thanks

A big thanks goes to @cugu for doing a lot of GitHub house keeping and submitting various, helpful PRs while I was gone!
Another big shoutout has to go to @vcoppe and [gpx.studio](https://github.com/gpxstudio/gpx.studio). Their code was a huge help implementing the various map overlays and GPX editing functionality that was added in this patch.


# v0.17.2
## Features
- Trails in the map view can now be sorted
- Adds ogp metadata tags for SEO
- Public profiles are now accessible by anonymous users
- A trail's direction can now be reversed in the editor
- Adds localization for the calendar component (thanks @james-geiger)

## Bug fixes
- Waypoint descriptions are now properly formatted
- The summit log table on the statistics page shows data again
- Fixes bug that caused lists to disappear when having more than 5 lists
- Removing the hillshading URL is now possible
- Clicking on a category on the homepage links to the correct trails page again
- Fixes bug that caused trail to disappear when switching map style to OpenTopoMap
- Trails are now correctly marked as "(Not) Completed" when adding (deleting) a summit log
- Allow links to other sites when hosting a private instance

# v0.17.1

## Features

- Adds batch actions for trails. You can now select multiple trails from the list and add them to a list, for example. Big thanks to @slothful-vassal for the PR
- If a trail has more than two tags they are now toggleable on trail cards for less visual clutter. Thank to @briannelson95 for the PR
- The wanderer.to homepage now contains a dedicated "Servers" section where public instances are listed

## Bug fixes
- Searching for a trail on the homepage does no longer result in 404
- An actor's username and preferred username are no longer switched
- Accesing your own private profile no longer throws an error
- Reverse geocoding location lookup now also properly works when uploading a trail through the API
- Both trail's and summit log's duration is now stored in seconds for consistency
- All federated requests are now properly signed
- Fixes GPX parser to deal with empty tags
- Fixes bug that caused only 30 entries to be displayed in the statistics
- The "Add to list" button is available again when creating a trail
- Fixes bug that prevented saving lists with a large number of trails
- Fixes "Copy link" button when sharing lists
- Adds missing namespace to activitypub actor endpoint

# v0.17.0
> [!CAUTION]
This release contains breaking changes. They are marked with a ⚠️.  
**Please update to version v0.16.5 first before updating to v0.17.0.**

## Configuration
Check the reopsitory's [`docker-compose.yml`](https://github.com/open-wanderer/wanderer/blob/main/docker-compose.yml) for a valid configuration.

- ⚠️ The PocketBase environment variable `POCKETBASE_ENCRYPTION_KEY` is now required. It requires a valid 32 character AES key as its value. To generate a key, run `openssl rand -hex 16`.
- ⚠️ The PocketBase environment variable `ORIGIN`is now required. It must be set to the public IP or hostname (including the port) of your wanderer frontend and must equal the value set for the frontend's `ORIGIN` environment variable.

## Features
- Adds federation
- Adds rich text editor for descriptions and comments
  
## Docs
- Adds documentation for federation
- Restructures the documentation in three distinct parts (for users, admins & developers) for better separation of concerns

## Translation
- New language: Russian (thanks @jeffscrum)

# v0.16.5

## Features
- Further performance improvements when showing large amount of trails on the map
- Elevations are now recalculated when importing trails leading to improved elevation gain/loss calculation

## Bug fixes
- Tags are now properly displayed for trails on the front page
- Direct links to a list now properly load the trails in the list
- Fixes bug that caused trails with waypoints being rejected by the upload API
- Fixes duration for summit logs imported from komoot
- Fixes performance issues when loading trails with summit logs or waypoints
  
# v0.16.4
## Security
> [!CAUTION]
Fixes a critical vulnerability where, in rare cases, registered users could temporarily inherit another user's session. This was caused by an incorrectly scoped PocketBase instance being shared across concurrent requests on the server.

Impact:
  - Affects all versions prior to v0.16.4
  - Risk of temporary user session mix-up during concurrent requests

Fix:
  - Authentication is now correctly isolated per request
  - Session handling is fully secured on both client and server

Action Required:
  - Please update to v0.16.4 immediately and restart your wanderer instance to apply the fix

# v0.16.3
## Features
- Adds option to add waypoints directly by uploading photos with EXIF data
- Performance improvemtents when loading acitivities
- Major performance improvements when displaying multiple tracks on the map
- Adds ENV variables to configure pocketbase SMTP settings

## Bug fixes
- Fixes untranslated trail difficulty in table view
- Fixes wrong file extension when exporting trails on mobile
- Completed tours synced from komoot are now also marked as completed in wanderer
  
## Docs
- Updates ENV variables section to reflect changes mentioned above

# v0.16.2
## Features
- Adds various settings for route calculations
- Trails with no photos will now have an autogenerated route preview as the thumbnail
- Pressing "M" in the map view will hide the trail
- Reduces data when loading lists (thanks @slothful-vassal) 
- Trails are no longer automatically marked as completed upon creation. You will need to create a summit log manually to do so
- If SMTP settings are present, new users will be asked to confirm their email address
  
## Bug fixes
- Fixes bug that prevented totals from getting updated when creating a new route
- Comments in GPX files are now ignored when importing from the client side
- Fixes issue that prevented oAuth registered users from saving their settings
- Fixes bug that prevented new users from commenting
- The stats page now correctly shows more than 30 activities

## Docs
- Fixes allowed values for trail difficulty in API reference
- Fixes env var descriptions
- Updates oAuth docs to reflect changes in PocketBase


# v0.16.1
## Features
- Trail filter settings are now saved when you visit a trail and come back
- Trail descriptions can now be up to 10000 characters long
  
## Bug fixes
- Fixes error in the KML file parser
- Fixes error that caused trails to disappear from the map when switching styles
- Fixes route point numbering when deleting in-between route points
- Fixes bug that caused integration secrets to be encrypted multiple times (thanks @Kami)
- Improves error handling when encountering unexpected data while syncing integrations
- Fixes a bug that allowed unregistered users to access the list creation page
- strava activities with no or empty GPS data are now ignored (thanks @dyuri)

## Docs
- Multiple updates by the community to increase clarity and update outdated info (thanks @huggenknubbel, @Kami)

# v0.16.0
> [!CAUTION]
This release contains breaking changes. The necessary migrations will happen automatically.
**Please update to version v0.15.2 first before updating to v0.16.0.**

## Maintenance
- Updates to PocketBase v0.26.1
- Bumps required go version to >= 1.23.0

## Features
- Introduces tags for trails
- Adds support for KMZ files
- Waypoints can now be created by clicking on the map when creating a new trail
- Adds support for videos

## Bug fixes
- Fixes map trail bounding box to include public and shared trails
- Fixes bug that caused orphan waypoints and summit logs
- The default language is now set correctly after registering
- Non-highlight photos from komoot are now synced correctly
- Fixes bug that prevented newly created trails from being saved again
- Fixes calculation of the total trail distance
- Waypoint map markers are now removed correctly when editing a trail
- Trails with no category can now be added to lists
- Trail shares are now persisted correctly in meilisearch
- Fixes pagination when a URL parameter is present

# v0.15.2
## Features
- Password fields now have a hint if you surpass the maximum password length (72)

## Bug fixes
- Fixes bug that prevented strava activities with heartrate data being imported
- Fixes bug that prevented users from creating new summit logs
- Fixes unclear error messages when saving integrations
- Fixes login issues for wanderer instances hosted via http


# v0.15.1
## Features
- You can now choose to sync only completed or planned tours from komoot
- Toast messages now stack

## Bug fixes
- Fixes bug that prevented the komoot integration from toggling on
- Fixes bug that caused orphaned summit logs
- Fixes bug that prevented users from saving an updated trail


# v0.15.0
## Features
- Integrations: you can now sync your strava and komoot trails directly with wanderer. [Learn more](https://wanderer.to/guides/integrations/).
- Updates the trail details view to give a clearer idea of the trail's course
- Improved trail import dialog
- Duplicate detection when importing trails
- Adds option to import a trail directly from a URL
- Significant preformance improvements when displaying trails in the map view (allows for up to 500 trails to be displayed at once)
- Improves calculation of total elevation gain and loss by apllying a smoothing function (thanks @gri38)

## Bug fixes
- Fixes a bug that caused batch import jobs to fail if a filename contained a comma
- Fixes trail card height issues
- Fixes bug that caused some trails to be hidden from the list view
- Fixes response headers to be <4kB to prevent crashing default reverse proxy configs
  
# v0.14.0
> [!IMPORTANT]
> This release introduces significant updates, including the migration of the frontend to [Svelte 5](https://svelte.dev/blog/svelte-5-is-alive). While the migration has been rigorously tested, there is a possibility that some features may not function as expected. We encourage you to report any issues you encounter. Additionally, the location search functionality has been transitioned from a locally hosted meilisearch index to nominatim. This upgrade offers substantially improved location search capabilities within wanderer. As a result, the custom meilisearch docker image (`flomp/wanderer-search`) is now deprecated. You can safely replace it with the official meilisearch image (`getmeili/meilisearch:v1.11.3`) in your `docker-compose.yml`.

## Maintenance
- Migrates to Svelte 5
- Removes the cities500 index from meilisearch

## Features
- Switches location search to nominatim
- Lists are now fuzzy searchable from the searchbar on the frontpage, in the map view and in the list view

## Bug fixes
- Fixes a bug that caused new users to not be able to save their settings
- The default language is now set to the browser language after registering a new user
- Moves the user biography out of the auth cookie to decrease cookie size

# v0.13.2

## Security
- Fixes potential XSS attack vector in waypoint and summit log map popups

## Features
-  Adds page loading bar
-  Improves route editor interface
-  Adds location search to trail create and edit form
-  You can now focus the map on your geolocation
-  Updates max. photo size for waypoints, summit logs and trails to 20MB

## Bug fixes
- Fixes bug that caused new users to not be redirected after registering
- Fixes bug that caused map coordinates to not wrap properly
- Fixes bug that caused new users to not be able to create lists
- Clicking on an icon in the waypoint icon picker now picks the correct icon
- Fixes bulk auto-upload
- Deleting an account now properly cascade deltes all associated objects from the database

## Translations
- New translation: Spanish (thanks to @xccose)
- Updated translations (thanks to all contributors)

# v0.13.1

## Features
- Improves loading speed of the home page
- Improves server side rendering of certain components

## Translations
- Various additions across multiple languages (thanks to all contributors)

## Bug fixes
- Fixes a bug that hid the map on some sites on mobile devices
- Fixes responsive layouts on mobile devices

# v0.13.0
## Features
- Adds a profile page with timeline, trails and stats of the respective user
- Adds notifications (on the website and via email) for various events (e.g. a new comment on your trail). Notifications can be toggled in the settings.
- You can now follow other users
- Settings page overhaul:
  - Clearer setting sections
  - Option to add a personal biography
  - New privacy settings page
  - New notification settings page
- Lists can now be public
- Lists can now be filtered and sorted
- Trails can now be displayed as a table (thanks @tofublock)
- Minor quality of life improvements:
  - more descriptive error messages
  - uniform empty states

## Bug fixes
- Fixes bug that caused trails to not be displayed for new users
- Fixes various bugs related to improper API data validation

## Docs
- Improves and updates API reference
- Improves clarity of "From source installation" guide

# v0.12.0
> [!CAUTION]
This release contains breaking changes. Most migrations will happen automatically, but you will need to take action in two places that will be clearly marked ⚠️ further down.

## Maintenance
- Updates to meilisearch version 0.11.3. 
- ⚠️ meilisearch indices are not compatible across minor versions. This means you will need to rename or delete your [`data.ms`](https://github.com/open-wanderer/wanderer/blob/8635de78b9f1510e2316b08e605b175a2615f4db/docker-compose.yml#L19) folder on your host system to force meilisearch to rebuild the index on the next start (note that this can take a little while). 

## Features
- Adds password reset email function for users (see [docs](https://wanderer.to/guides/authentication/#forgot-your-password) for more info)
- You can now add photos to your summit logs
- Complete rewrite of the map logic switching from raster to vector tiles

⚠️ Custom raster tilesets added via `Settings -> Display -> Tilesets` will no longer work. You will have to delete them for the map to show correctly. Tileset URLs must now point to a valid `style.json` describing a vector tileset (see [docs](https://wanderer.to/guides/customize-map/#custom-map-styles) for more info).
- 3D Terrain and Hillshading are now available (see [docs](https://wanderer.to/guides/customize-map/#terrain--hillshading) fore more info)
- Adds two more default map styles: CARTO Light & Dark
- Adds loading animations for trail lists

## Bug fixes

- Fixes bug that caused a new trail to be created instead of updated when uploading a new GPS data source to an existing trail
- Fixes bug that caused trails to throw a 404 error when they had summit logs created before v0.11.0
- Waypoint markers can now also be dragged after a trail was saved
- Fixes issue with GPX export when using Google Chrome (thanks [@tofublock](https://github.com/tofublock))

## Miscellaneous
As the number of contributors to this project continues to grow (which I’m very happy about), I’ve set up a [Discord channel](https://discord.gg/MdpybUHc) for more direct communication. If you’re interested in helping with wanderer, feel free to join!

# v0.11.0
## Features
- Other user's profiles can now be viewed
- The summit log author is now listed in the summit log table
- The trail author is now listed for every trail
- Trails can now be filtered by author
- Users will now receive an error message when they try to upload a photo that is too large
- Updated 3D Model on front page
  
## Bug fixes
- Waypoints and summit logs of shared trails are now properly displayed
- Fixes missing translation for trail categories
- Fixes events in profile calendar

# v0.10.1
## Features
- Adds elevation loss to trails. Please note that trails created before this version will have a default elevation loss of 0. Edit & save to update.
  
## Bug fixes
- Fixes bug that caused auto-added summit logs to not have distance, durtaion etc.
- Fixes error in the auto-upload feature
- Fixes access permissions for profile page
- A summit log is now also created automatically when you upload a trail directly
- Fixes link to the API documentation in footer
  
# v0.10.0
## Features
- A new summit log entry is now added automatically when uploading a new GPX file for a new or existing trail
- GPX files can now be attached to summit logs
- Adds a new profile page with filterable statistics derived from summit log GPX data
- When importing a GPX file the filename will be used as a fallback
  
# v0.9.0
## Features
- Complete visual overhaul of the list page
- Lists can now be shared with other users
- Updates visual style for waypoints

## Bug fixes
- Fixes a bug that caused categories not to load properly on page reload
- Fixes icon picker suggestions for waypoints
- Fixes a bug that would prevent public and shared trails from showing up in the overview


# v0.8.2
## Features
- The pagination is now available at top and bottom
- HEIC image format is now supported
- Exporting only a trail without summit logs or photos will create a single file instead of a ZIP folder
- The current page is now remembered when navigating back to the trail overview

## Bug fixes
- Fixed map height when viewing a trail in detail view

# v0.8.1
## Features
- Public and shared trails can now be exported and printed

## Bug fixes
- Correctly adds the xmlns header to exported GPX files
- Fixes detail view for shared and public trails
- Fixes bug that caused the default category to be re-applied when editing a trail


# v0.8.0
## Features
- The settings page layout got a complete visual overhaul
- You can now change your email and password from the web UI
- You can import GPX, KML, TCX, and FIT files directly through the UI or using the API
- Exporting all trails at once is now possible on the settings page
- Additional custom map tile sets can now be added via the settings page

## Bug fixes
- Added multiple missing translations
- The hotline metric selector icons are now radio buttons instead of checkboxes


# v0.7.3
## Features
- Adds support for FIT files
  
## Bug fixes
- Fixes display issue for the filter panel in the map view

## Translations
- adds Italian translation (thanks to [lukasitaly](https://github.com/lukasitaly))
  
## Bug fixes
- Icons in dropdown menus are now displaying properly again
- The trail id is now returned when using the upload API endpoint

# v0.7.2
## Bug fixes
- Icons in dropdown menus are now displaying properly again
- The trail id is now returned when using the upload API endpoint

# v0.7.1
## Features
- A warning is now displayed if the ORIGIN environment variable is misconfigured
  
## Bug fixes
- Fixes login for http connections
- Trails are now properly sorted across all pages

# v0.7.0
## Features
- Trail sort and sort direction are now remembered through a page reload
- You can now pick between OpenStreetMaps and OpenTopoMaps tiles
- The gradient track line can now display altitude, slope, and speed
- A waypoint's coordinates can now be inferred from a photo's EXIF data

## Docs

- Added a guide for custom categories


# v0.6.1
## Bug fixes
- Fixes a bug that would show wrong comments under a trail
- Fixes an overflow issue in list views
- Settings are now created properly when signing up with OAuth
- Fixes a bug that caused properly named trails to get a generic name

# v0.6.0
## Features
- You can now share trails with other users by selecting "Share" from the trail contextmenu. You can set the permission level as "View" or "Edit".
- Adds missing translations

## Bug fixes
- Fixed a bug that prevented comments from showing up
- GPX files without a name in the metadata section will now receive a generic name when uploaded through the API

# v0.5.1
## Features
- You can now export trails as GPX or GEOJson files. Optionally you can include photos and summit book entries of the trail. This replaces the "Download GPX" function in previous versions.

## Bug fixes
- Fixed a bug that would prevent users from creating multiple summit log entries without reloading the page
- wanderer now takes the `<rte>` tag into account when displaying a trail on the map
- A trail's date attribute is now the current date by default

# v0.5.0

## ⚠️ Breaking changes ⚠️
- This version updates the index pattern of the meilisearch index. Please delete your `data.ms` folder before launching wanderer. The indices will be rebuilt on launch. Otherwise trail filtering will no longer work.

## Features
- Trails can now be filtered by date
- Elevation, slope and speed graphs are now also visible when creating a new trail
- When creating a new trail you now have the option to create a new route from scratch without uploading a GPX file. Press the "Draw a route" button and plan your new route directly in wanderer. We use [valhalla](https://github.com/valhalla/valhalla) and their associated free [hosted service](https://gis-ops.com/global-open-valhalla-server-online/) to calculate the routes. To activate the feature make sure to set the PUBLIC_VALHALLA_URL environment variable on you wanderer-web service. See the current [docker-compose.yml](https://github.com/open-wanderer/wanderer/blob/main/docker-compose.yml) for a working configuration.

## Bug fixes
- Uploaded trails will now have a date if it can be parsed from the file

# v0.4.0
## Features

- Trails can now be printed to PDF. Select "Print" from the menu when viewing a trail.
- Trails can now have a date.
- You can now comment on trails
- Adds a setting to focus the map on all trails instead of a specific location

## Bug fixes
- fixed a bug that would show a wrong date for summit logs for certain time zones (now really)
- fixed a bug that prevented waypoints from showing up in public trails

# v0.3.2
## Bug fixes

- Fixed a bug that caused a 500 Internal Error to appear when viewing trails without an account

## Translations

- improves Dutch translation (thanks to [Vistaus](https://github.com/Vistaus))


# v0.3.1
## Features

- Max values for elevation gain and distance filters are now dynamically calculated based on your longest trail
- Disabling username/email & password auth in PocketBase is now reflected in wanderer's login UI

## Bug fixes

- Fixed a bug that prevented import trails from appearing in map view

## Translations

- adds Simplified Chinese translation (thanks to [icyleaf](https://github.com/icyleaf))

## Dependencies

- updates PocketBase to v0.22.7

# v0.3.0
## Features

- Trails can now be added to a list while editing or creating a trail. The trail must be saved at least once to add it to a list.
- wanderer now has an auto-upload folder. GPX files in this folder will be autmatically uploaded and converted to a trail. Read the [docs](https://github.com/open-wanderer/wanderer/wiki/API#auto-upload-folder) for more information.
- addded support for TCX and KML files. Note that this feature is still experimental. Please report any issues you encounter.
- added OAuth support. Read [here](https://github.com/open-wanderer/wanderer/wiki/OAuth) how to enable providers.
  
## Bug fixes

- fixed a bug that would show a wrong date for summit logs for certain time zones
- added client side validation for usernames

## Translations
- adds French translation (thanks to [seb2020](https://github.com/seb2020))
- adds Hungarian translation (thanks to [sszemtelen](https://github.com/sszemtelen))


# v0.2.1
## Bug fixes

- summit book dates now show in the correct format for the current locale
- fixed a bug that would overwrite trail names and descriptions when editing a trail

## Translations

- adds Dutch translation (thanks to [yves-bonami](https://github.com/yves-bonami))
- adds Polish translation (thanks to [ludrol](https://github.com/ludrol))
- adds Portuguese translation (thanks to [poVoq](https://github.com/poVoq))

# v0.2.0
## Features

- when creating/editing a trail you can now drag & drop photos into the photo section to upload them
- you can now attach photos to waypoints
- you can now edit the "Distance", "Elevation Gain" and "Est. Duration" fields
- waypoint markers can now be moved with drag & drop
- lists can now be displayed as a map showing all trails contained in the list
- you can now prevent users from signing up by setting the `DISABLE_SIGNUP` environment variable to `true`
- you can now upload GPX files via the API to create trails. Check the [documentation](https://github.com/open-wanderer/wanderer/wiki/API#upload-trails) for more info.
- the city index now includes states

> Note: for city states to show up in your search you have to delete your data.ms folder if you already have a previous installation of wanderer. The indices will then be rebuilt on startup.

## Docs

-  added complete API documentation

# v0.1.1
## Bug fixes

- fixed a bug that would prevent trails longer than 20km from being displayed
- added BODY_SIZE_LIMIT env variable to docker compose to allow for bigger file uploads
- fixed a bug that caused only 5 trails to be shown at a time
- fixed a bug that would cause waypoints not to be deleted from the backend
- updated the default docker-compose.yml to include a secure MEILI_MASTER_KEY
- the default location field now sets the value correctly after clicking on a search result
  
## Docs

- updated the docs to include BODY_SIZE_LIMIT

# v0.1.0 
- Initial release