# Photo assets: technical reference

The `assets` collection stores photos separately from their links to trails, waypoints and summit logs. The schema defines external identities, copy lineage and media provenance.

## Data model

Migration `db/migrations/1791800003_created_assets.go` defines the collections and indexes.

| Field | Meaning |
| --- | --- |
| `type` | Currently only `photo`. |
| `file` | Local PocketBase file. |
| `storage_mode` | `copy` or `link_private`. |
| `author` | The photo author's ActivityPub actor. |
| `external_provider`, `external_id` | Stable external identity within an author's library. |
| `taken_at`, `lat`, `lon` | Photo capture time and coordinates. |
| `metadata` | Public photo metadata, including the content hash and generated route preview marker. |
| `source_metadata` | Hidden provenance for internal persistence and author exports. |
| `remote_status` | `available`, `missing`, or `inaccessible`. |
| `remote_checked_at`, `remote_missing_since`, `remote_error` | Remote media status details. |

The partial unique index on `(author, external_provider, external_id, type)` prevents duplicate provider photos for one author. Author, library ordering and copy lineage queries have dedicated indexes.

| Link collection | Target field | Unique index |
| --- | --- | --- |
| `trail_assets` | `trail` | `(asset, trail)` |
| `waypoint_assets` | `waypoint` | `(asset, waypoint)` |
| `summit_log_assets` | `summit_log` | `(asset, summit_log)` |

An asset may have multiple targets. Each relation permits one link between the same asset and target. Trail links also mark the selected thumbnail.

Only successful photo file responses and their redirects use `Cache-Control: private, max-age=3600`; error responses use `no-store` and are not cached. A viewer who already loaded a photo may keep seeing their cached copy for up to one hour after access is revoked. Every new request checks current access immediately. Shared caches never store these responses. Successful photo responses retain their validators and `Vary` headers.

## Persistence and deletion

Uploaded photos receive a SHA-256 hash when written. A transient file identity avoids hashing the same upload twice, and replacing a file invalidates its previous hash.

Lifecycle hooks collect photo links before deleting a trail, waypoint or summit log, then delete photos without remaining links. This also covers cascades and deletes that committed before an after-delete hook reported an error. Photos linked elsewhere survive.

## Existing photos and target permissions

The upgrade keeps existing trail, waypoint and summit-log photos as assets, with legacy URLs resolved under the current access rules. Migrated waypoint photos belong to the waypoint's trail author when that owner resolves; otherwise they keep the waypoint author's ownership.

Trail and waypoint photo links can be edited by the trail's author or a user with a current edit share. Creating a waypoint does not grant separate photo rights after the share ends. Summit-log photo links can be edited only by the entry's author or the trail's author. Asset authors can still withdraw their own photos by unlinking or deleting them.

Server-generated route-placeholder SVGs are generated assets. Real photos take precedence in thumbnails and previews, and a trail with only its placeholder still shows it. Placeholders are not attached to outgoing federation activities.

## Contributed photos

A photo belongs to its contributor, who may withdraw it by unlinking or deleting it. Before that contributor's account is deleted, photos still linked to another actor's target move to the owner of the oldest surviving link to another owner's target. A trail or waypoint uses the trail's author as owner; a summit-log entry uses its own author. Other links remain intact. If the recipient already owns the same external identity and both photos have the same nonempty content hash, links are moved to that existing asset, duplicate links are combined and thumbnail marks are retained. Otherwise the contributed photo itself changes owner and keeps its bytes and links. A conflicting external identity is retained only in hidden provenance. Photos linked only to the deleted actor's own targets cascade normally.

The record update API does not let non-superusers replace asset files, including through file modifiers. Metadata edits remain available to the author, and internal writes can still replace stored media.
