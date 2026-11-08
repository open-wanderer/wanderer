import { RecordIdSchema, RecordIdValueSchema } from "#lib/models/api/base_schema.ts";
import type { AssetLink } from "#lib/models/asset.ts";
import { Collection, handleError } from "#lib/util/api_util.ts";
import { ClientResponseError } from "pocketbase";
import type { RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const TrailThumbnailSchema = z.object({
    asset: RecordIdValueSchema.nullable().optional(),
});

/**
 * @swagger
 * /api/v1/trail/{id}/thumbnail:
 *   patch:
 *     summary: Set trail thumbnail
 *     description: >
 *       Marks the trail's link to the photo `asset` as thumbnail (`trail_assets.is_thumbnail`) and clears the mark on
 *       its other photos. Without `asset`, or with `null`, all marks are cleared. Trail responses report the marked
 *       photo's index in `photos` as `thumbnail`, or else the first photo's. A generated route preview counts only
 *       when the trail has no other photo, even when it is marked.
 *       Requires the trail's author or an edit share.
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
 *               asset:
 *                 type: string
 *                 nullable: true
 *                 description: ID of a photo asset linked to the trail
 *     responses:
 *       200:
 *         description: Thumbnail updated
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 acknowledged:
 *                   type: boolean
 *       400:
 *         description: Invalid id or body, or the asset is not linked to the trail
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Insufficient permissions for trail
 *       404:
 *         description: Trail not found
 *       500:
 *         description: Internal Server Error
 */
export async function PATCH(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return Response.json({ message: "Unauthorized" }, { status: 401 });
        }

        const safeParams = RecordIdSchema.parse(event.params);
        const data = TrailThumbnailSchema.parse(await event.request.json());
        await assertTrailWritable(event, safeParams.id);

        let selectedLink: AssetLink | undefined;
        if (data.asset) {
            selectedLink = await linkedTrailAsset(event, safeParams.id, data.asset);
        }

        const links = await event.locals.pb.collection(Collection.trail_assets).getFullList<AssetLink>({
            filter: event.locals.pb.filter("trail = {:trail}", { trail: safeParams.id }),
            requestKey: null,
        });

        for (const link of links) {
            const shouldBeThumbnail = Boolean(selectedLink && link.id === selectedLink.id);
            if (Boolean(link.is_thumbnail) === shouldBeThumbnail) {
                continue;
            }
            await event.locals.pb.collection(Collection.trail_assets).update(link.id!, {
                is_thumbnail: shouldBeThumbnail,
            }, { requestKey: null });
        }

        return Response.json({ acknowledged: true });
    } catch (e) {
        return handleError(e);
    }
}

async function assertTrailWritable(event: RequestEvent, trailID: string) {
    const actorID = event.locals.user?.actor;
    if (!actorID) {
        throw new ClientResponseError({
            status: 401,
            response: { message: "Unauthorized" },
        });
    }

    const trail = await event.locals.pb.collection(Collection.trails).getOne(trailID);
    if (trail.author === actorID) {
        return;
    }

    const shares = await event.locals.pb.collection(Collection.trail_share).getFullList({
        filter: event.locals.pb.filter("trail = {:trail} && actor = {:actor} && permission = {:permission}", {
            trail: trailID,
            actor: actorID,
            permission: "edit",
        }),
        requestKey: null,
    });
    if (shares.length > 0) {
        return;
    }

    throw new ClientResponseError({
        status: 403,
        response: { message: "Insufficient permissions for trail" },
    });
}

async function linkedTrailAsset(event: RequestEvent, trailID: string, assetID: string): Promise<AssetLink> {
    const links = await event.locals.pb.collection(Collection.trail_assets).getFullList<AssetLink>({
        filter: event.locals.pb.filter("trail = {:trail} && asset = {:asset}", {
            trail: trailID,
            asset: assetID,
        }),
        requestKey: null,
    });
    if (!links.length) {
        throw new ClientResponseError({
            status: 400,
            response: { message: "Asset is not linked to trail" },
        });
    }
    return links[0];
}
