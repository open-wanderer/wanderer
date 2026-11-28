import type { Asset } from "#lib/models/asset.ts";
import type { AttachedAsset } from "#lib/models/attached_asset.ts";
import { RecordIdValueSchema } from "#lib/models/api/base_schema.ts";
import { Collection, handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";
import { sharedAssetCopy } from "#lib/server/asset_copy_singleflight.ts";
import { assetCopyFilename } from "#lib/util/media_filename.ts";

interface AssetLinkTarget {
    trail?: string;
    waypoint?: string;
    summit_log?: string;
}

/**
 * @swagger
 * /api/v1/assets:
 *   put:
 *     summary: Upload or link photo assets
 *     description: >
 *       Uploads photo files as assets, links the user's own photos or copies readable photos from other authors, and attaches them to a
 *       trail, waypoint or summit log. Trails and waypoints require the trail's author or a current edit share.
 *       Summit logs require the entry's author or the trail's author.
 *     tags:
 *       - Assets
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             properties:
 *               files:
 *                 type: array
 *                 items:
 *                   type: string
 *                   format: binary
 *                 description: Photo files to upload.
 *               assetIds:
 *                 type: array
 *                 items:
 *                   type: string
 *                 description: >
 *                   IDs of readable assets to attach. The user's own assets are linked; assets from other authors are
 *                   copied into new local assets owned by the user. Repeat the field for several IDs. Values that are not record IDs are
 *                   ignored, and an asset already linked to the target is not linked twice.
 *               trail:
 *                 type: string
 *                 description: >
 *                   Trail id to attach assets to. Together with `waypoint` or `summit_log` it must be that record's
 *                   trail, and the assets are not linked to the trail itself.
 *               waypoint:
 *                 type: string
 *                 description: Waypoint id to attach assets to
 *               summit_log:
 *                 type: string
 *                 description: Summit log id to attach assets to
 *               metadata:
 *                 type: string
 *                 description: JSON object stored as `metadata` on every asset created from `files`.
 *     responses:
 *       200:
 *         description: The assets attached from assetIds, followed by uploaded files. Copies include sourceAssetId.
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/AttachedAsset'
 *       400:
 *         description: Missing or invalid target relation, or a waypoint or summit log that is not on `trail`
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Insufficient permissions for target trail
 *       404:
 *         description: Target or linked asset not found
 *       413:
 *         description: Request body exceeds the fixed 1 GiB asset batch limit (request_body_too_large). Split larger selections into smaller requests.
 *       502:
 *         description: A selected photo could not be downloaded. Completed attachments remain in detail.
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AssetWriteError'
 *       504:
 *         description: Photo copy timed out. Completed attachments remain in detail.
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AssetWriteError'
 *       500:
 *         description: Internal Server Error; detail preserves completed attachments and uploaded file counts.
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AssetWriteError'
 */
export async function PUT(event: RequestEvent) {
    const created: AttachedAsset[] = [];
    const uploadedAssets: Asset[] = [];
    try {
        if (!event.locals.user) {
            return Response.json({ message: "Unauthorized" }, { status: 401 });
        }
        if (!event.locals.user.actor) {
            return Response.json({ message: "Actor not found" }, { status: 401 });
        }

        const data = await event.request.formData();
        const files = data.getAll("files").filter(isUploadedFile);
        const assetIds = [...new Set(data
            .getAll("assetIds")
            .filter((id): id is string => typeof id === "string" && RecordIdValueSchema.safeParse(id).success))];
        const target = await validateTarget(event, data);
        const metadata = metadataValue(data.get("metadata"));

        for (const assetId of assetIds) {
            // Read and download as the caller, so a revoked share cannot be
            // used to copy another author's private photo.
            const source = await event.locals.pb.collection(Collection.assets).getOne<Asset>(assetId);
            const copied = source.author !== event.locals.user.actor;
            const copy = copied ? await copyAsset(event, source) : { asset: source, created: false };
            const record = copy.asset;
            try {
                await createAssetLinks(event, record.id, target);
            } catch (error) {
                if (copy.created) {
                    await cleanupUnlinkedAsset(event, record.id);
                }
                throw error;
            }
            created.push(copied ? { ...record, sourceAssetId: source.id } : record);
        }

        for (const file of files) {
            const asset = new FormData();
            asset.append("type", "photo");
            asset.append("storage_mode", "copy");
            asset.append("remote_status", "available");
            asset.append("author", event.locals.user.actor);
            asset.append("file", file);
            if (metadata) asset.append("metadata", JSON.stringify(metadata));

            const record = await event.locals.pb.collection(Collection.assets).create<Asset>(asset, { requestKey: null });
            try {
                await createAssetLinks(event, record.id, target);
            } catch (error) {
                await cleanupUnlinkedAsset(event, record.id);
                throw error;
            }
            created.push(record);
            uploadedAssets.push(record);
        }

        return Response.json(created);
    } catch (e: any) {
        const response = handleError(e);
        if (!created.length) return response;
        const body = await response.json();
        const headers = new Headers(response.headers);
        headers.delete("content-length");
        return Response.json({
            ...body,
            detail: { ...body.detail, attachedAssets: created, uploadedAssets, completedFileCount: uploadedAssets.length },
        }, { status: response.status, headers });
    }
}

// A saved copy may already have been reused while its first link was pending.
// Cleanup must recheck current links atomically before removing the file.
async function cleanupUnlinkedAsset(event: RequestEvent, assetId: string) {
    try {
        await event.locals.pb.send(`/assets/${assetId}?orphan=1`, {
            method: "DELETE", fetch: event.fetch, requestKey: null,
        });
    } catch {
        // Keep the original write error when cleanup needs a later retry.
    }
}

async function copyAsset(event: RequestEvent, source: Asset): Promise<{ asset: Asset; created: boolean }> {
    try {
        // Hidden provenance is resolved only by the backend after checking
        // source visibility and restricting the result to the caller's copy.
        const asset = await event.locals.pb.send<Asset>(`/assets/${source.id}/owned-copy`, {
            method: "GET", fetch: event.fetch, requestKey: null,
        });
        return { asset, created: false };
    } catch (error) {
        if (!(error instanceof ClientResponseError) || error.status !== 404) throw error;
    }
    return sharedAssetCopy(`${event.locals.pb.baseURL}:${event.locals.user!.actor}:${source.id}`, event.request.signal,
        (signal) => createCopiedAsset(event, source, signal));
}

async function createCopiedAsset(event: RequestEvent, source: Asset, signal: AbortSignal): Promise<Asset> {
    let response: Response;
    try {
        response = await event.fetch(event.locals.pb.buildURL(`assets/${source.id}/file`), {
            headers: { Authorization: `Bearer ${event.locals.pb.authStore.token}` }, signal,
        });
    } catch (error) {
        if (signal.aborted) throw error;
        throw new ClientResponseError({ status: 502, response: { code: "photo_download_failed", message: "Photo download unavailable; retry the request" } });
    }
    if (!response.ok) {
        const body = await response.json().catch(() => ({ message: "Photo download unavailable; retry the request" }));
        throw new ClientResponseError({ status: response.status, response: body });
    }
    const blob = await response.blob();
    if (!blob.size) throw new ClientResponseError({ status: 502, response: { code: "photo_download_failed", message: "Photo file is empty" } });

    const filename = assetCopyFilename(source, response.headers.get("Content-Type") ?? undefined);
    const file = new File([blob], filename, {
        type: response.headers.get("Content-Type") || "image/jpeg",
    });
    const data = new FormData();
    data.append("type", "photo");
    data.append("storage_mode", "copy");
    data.append("remote_status", "available");
    data.append("author", event.locals.user!.actor);
    data.append("file", file);
    for (const field of ["lat", "lon", "taken_at"] as const) {
        if (source[field] !== undefined && source[field] !== "") {
            data.append(field, String(source[field]));
        }
    }
    // Keep photo metadata, without retaining the other author's remote
    // references or legacy record IDs in this independent local copy.
    const metadata = Object.fromEntries(["capture_time", "generated"]
        .filter((key) => source.metadata?.[key] !== undefined)
        .map((key) => [key, source.metadata![key]]));
    metadata.copied_from = source.id;
    metadata.original_filename = filename;
    if (Object.keys(metadata).length) {
        data.append("metadata", JSON.stringify(metadata));
    }
    return event.locals.pb.collection(Collection.assets).create<Asset>(data, { requestKey: null, signal });
}

async function createAssetLinks(
    event: RequestEvent,
    assetId: string,
    target: AssetLinkTarget,
) {
    const links = [
        { collection: Collection.trail_assets, field: "trail", id: target.trail },
        { collection: Collection.waypoint_assets, field: "waypoint", id: target.waypoint },
        { collection: Collection.summit_log_assets, field: "summit_log", id: target.summit_log },
    ];

    for (const link of links) {
        if (!link.id) {
            continue;
        }
        const existing = await event.locals.pb.collection(link.collection).getFullList({
            filter: event.locals.pb.filter(`asset = {:asset} && ${link.field} = {:target}`, {
                asset: assetId,
                target: link.id,
            }),
        });
        if (existing.length) {
            continue;
        }
        await event.locals.pb.collection(link.collection).create(
            {
                asset: assetId,
                [link.field]: link.id,
            },
            { requestKey: null },
        );
    }
}

async function validateTarget(event: RequestEvent, data: FormData) {
    const user = event.locals.user!;
    const requestedTarget = {
        trail: recordIdValue(data.get("trail")),
        waypoint: recordIdValue(data.get("waypoint")),
        summit_log: recordIdValue(data.get("summit_log")),
    };
    let trailContext = requestedTarget.trail;
    let summitLogAuthor = false;

    if (!requestedTarget.trail && !requestedTarget.waypoint && !requestedTarget.summit_log) {
        throw new ClientResponseError({
            status: 400,
            response: { message: "Asset target is required" },
        });
    }

    if (requestedTarget.waypoint) {
        const waypoint = await event.locals.pb.collection(Collection.waypoints).getOne(requestedTarget.waypoint);
        if (!trailContext) {
            trailContext = waypoint.trail;
        } else if (waypoint.trail !== trailContext) {
            throw new ClientResponseError({
                status: 400,
                response: { message: "Waypoint does not belong to trail" },
            });
        }
    }

    if (requestedTarget.summit_log) {
        const summitLog = await event.locals.pb.collection(Collection.summit_logs).getOne(requestedTarget.summit_log);
        if (!trailContext) {
            trailContext = summitLog.trail;
        } else if (summitLog.trail !== trailContext) {
            throw new ClientResponseError({
                status: 400,
                response: { message: "Summit log does not belong to trail" },
            });
        }
        summitLogAuthor = summitLog.author === user.actor;
    }

    const needsTrailEdit = Boolean(requestedTarget.waypoint || !requestedTarget.summit_log);
    let authorized = summitLogAuthor && !needsTrailEdit;
    if (trailContext && !authorized) {
        const trail = await event.locals.pb.collection(Collection.trails).getOne(trailContext);
        const trailAuthor = trail.author === user.actor;
        const canEditTrail = trailAuthor || (needsTrailEdit && await hasTrailEditShare(event, trailContext, user.actor));
        const canEditEntry = !requestedTarget.summit_log || summitLogAuthor || trailAuthor;
        authorized = canEditEntry && (!needsTrailEdit || canEditTrail);
    }

    if (!authorized) {
        throw new ClientResponseError({
            status: 403,
            response: { message: "Insufficient permissions for trail" },
        });
    }

    return {
        trail: requestedTarget.waypoint || requestedTarget.summit_log ? undefined : requestedTarget.trail,
        waypoint: requestedTarget.waypoint,
        summit_log: requestedTarget.summit_log,
    };
}

async function hasTrailEditShare(event: RequestEvent, trailId: string, actorId: string): Promise<boolean> {
    const shares = await event.locals.pb.collection(Collection.trail_share).getFullList({
        filter: event.locals.pb.filter("trail = {:trail} && actor = {:actor} && permission = {:permission}", {
            trail: trailId,
            actor: actorId,
            permission: "edit",
        }),
    });
    return shares.length > 0;
}

function stringValue(value: FormDataEntryValue | null): string | undefined {
    return typeof value === "string" && value.length ? value : undefined;
}

function recordIdValue(value: FormDataEntryValue | null): string | undefined {
    const raw = stringValue(value);
    return raw ? RecordIdValueSchema.parse(raw) : undefined;
}

function isUploadedFile(value: FormDataEntryValue): value is File {
    return (
        typeof value === "object" &&
        value !== null &&
        "arrayBuffer" in value &&
        typeof value.arrayBuffer === "function" &&
        "name" in value
    );
}

function metadataValue(value: FormDataEntryValue | null): Record<string, unknown> | undefined {
    if (typeof value !== "string" || !value.length) {
        return undefined;
    }
    let parsed: unknown;
    try {
        parsed = JSON.parse(value);
    } catch {
        return undefined;
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
        return undefined;
    }
    return parsed as Record<string, unknown>;
}
