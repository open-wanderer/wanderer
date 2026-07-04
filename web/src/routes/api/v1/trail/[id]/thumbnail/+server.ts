import { RecordIdSchema, RecordIdValueSchema } from "$lib/models/api/base_schema";
import type { AssetLink } from "$lib/models/asset";
import { Collection, handleError } from "$lib/util/api_util";
import { ClientResponseError } from "pocketbase";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const TrailThumbnailSchema = z.object({
    asset: RecordIdValueSchema.nullable().optional(),
});

export async function PATCH(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
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

        return json({ acknowledged: true });
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
