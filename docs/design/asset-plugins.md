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

## Photo attachment and file API

`PUT /api/v1/assets` uploads files as local photo assets, links the caller's existing photos, and makes independent local copies of readable photos owned by another author. Targets are trails, waypoints, or summit logs; a child target must belong to the supplied trail. Trails and waypoints require the trail's author or a current edit share. Summit-log photos require the entry's author or the trail's author. Copy lookup checks current source access and resolves reusable copies through private provenance. Concurrent requests share a copy operation.

`DELETE /api/v1/assets/{id}` with a target removes only its matching link. Removing a photo is idempotent: an unknown asset or an asset not linked to the requested target returns `assetDeleted: false`. After a matching link is removed, an orphaned asset is deleted only when the caller owns it. Authors can withdraw their own linked photos even after their edit access ends. Without a target the caller must own the asset and its links are removed with it. `PATCH /api/v1/trail/{id}/thumbnail` marks a linked photo or clears all marks.

`GET /api/v1/assets/{id}/file` rechecks asset access on every request, including conditional requests, and redirects internally to PocketBase's local file handler. The BFF forwards share tokens, authorization and validators, preserving bodyless 304 responses and cancellation. Downloads keep PocketBase's file-token, thumbnail and range handling.

The supplied image and Compose configuration default to `BODY_SIZE_LIMIT=1G`. Asset batches have a fixed 1 GiB aggregate request limit, including multipart framing; increasing `BODY_SIZE_LIMIT` does not raise it. Error details retain completed attachments and file counts after partial writes. When building a saved record's response fails, the error identifies the committed record without publishing incomplete photo relations. Trail card queries request at most 100 IDs and return at most three photo links per trail, with the marked real photo ahead of generated route previews. Stored-file card URLs keep the file name and extension and carry any share token. Card asset expands retain all public scalar values; their metadata contains only `generated`, keeping unrelated metadata and private provenance out of card payloads.

Failed attachment cleanup checks current links in a transaction and removes only an owned orphan. A copy that another request has already reused stays linked to its successfully saved targets.
