import type { Asset } from "$lib/models/asset";
import { RecordIdSchema, RecordIdValueSchema } from "$lib/models/api/base_schema";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const AssetMetadataPatchSchema = z.object({
    lat: z.number().min(-90).max(90).optional(),
    lon: z.number().min(-180).max(180).optional(),
    takenAt: z.string().optional(),
    replaceCoordinates: z.boolean().optional(),
    markChecked: z.boolean().optional(),
}).refine((data) => (data.lat === undefined) === (data.lon === undefined), {
    message: "lat and lon must be provided together",
});

/**
 * @swagger
 * /api/v1/assets/{id}:
 *   delete:
 *     summary: Delete asset
 *     tags:
 *       - Assets
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Asset deleted
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 acknowledged:
 *                   type: boolean
 *       400:
 *         description: Invalid asset id
 *       404:
 *         description: Asset not found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const safeParams = RecordIdSchema.parse(event.params);

        const targets = z.object({
            trail: RecordIdValueSchema.optional(),
            waypoint: RecordIdValueSchema.optional(),
            summit_log: RecordIdValueSchema.optional(),
        }).parse(Object.fromEntries(event.url.searchParams));

        const linkTargets = assetLinkTargets(targets);

        const searchParams = new URLSearchParams();
        for (const target of linkTargets) {
            searchParams.set(target.field, target.id);
        }
        const query = searchParams.toString();
        const response = await event.locals.pb.send(`/assets/${safeParams.id}${query ? `?${query}` : ""}`, {
            method: "DELETE",
            fetch: event.fetch,
        });
        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}

export async function PATCH(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const safeParams = RecordIdSchema.parse(event.params);
        const data = AssetMetadataPatchSchema.parse(await event.request.json());

        const asset = await event.locals.pb.collection(Collection.assets).getOne<Asset>(safeParams.id);
        const update: Record<string, unknown> = {};

        if (
            data.lat !== undefined &&
            data.lon !== undefined &&
            (
                !assetHasCoordinates(asset) ||
                (data.replaceCoordinates && isLegacyMigratedAsset(asset))
            )
        ) {
            update.lat = data.lat;
            update.lon = data.lon;
        }

        if (data.takenAt && !asset.taken_at) {
            const takenAt = new Date(data.takenAt);
            if (!Number.isNaN(takenAt.getTime())) {
                update.taken_at = takenAt.toISOString();
            }
        }

        if (data.markChecked) {
            update.metadata = metadataWithExifBackfillMarker(asset.metadata);
        }

        if (Object.keys(update).length === 0) {
            return json(asset);
        }

        const updated = await event.locals.pb.collection(Collection.assets).update(
            safeParams.id,
            update,
            { requestKey: null },
        );
        return json(updated);
    } catch (e: any) {
        return handleError(e);
    }
}

function assetLinkTargets(targets: { trail?: string; waypoint?: string; summit_log?: string }) {
    const trailIsDirectTarget = targets.trail && !targets.waypoint && !targets.summit_log;
    return [
        { field: "trail", id: trailIsDirectTarget ? targets.trail : undefined },
        { field: "waypoint", id: targets.waypoint },
        { field: "summit_log", id: targets.summit_log },
    ].filter((target): target is { field: string; id: string } => Boolean(target.id));
}

function assetHasCoordinates(asset: Pick<Asset, "lat" | "lon">): boolean {
    return (
        typeof asset.lat === "number" &&
        typeof asset.lon === "number" &&
        Number.isFinite(asset.lat) &&
        Number.isFinite(asset.lon) &&
        (asset.lat !== 0 || asset.lon !== 0)
    );
}

function isLegacyMigratedAsset(asset: Pick<Asset, "metadata">): boolean {
    return typeof asset.metadata?.source_collection === "string";
}

function metadataWithExifBackfillMarker(metadata: Asset["metadata"]): Record<string, unknown> {
    const base = metadata && typeof metadata === "object" && !Array.isArray(metadata)
        ? metadata
        : {};
    return {
        ...base,
        exif_backfill: {
            checked_at: new Date().toISOString(),
        },
    };
}
