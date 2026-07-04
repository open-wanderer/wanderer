import type { Asset, AssetLink } from "$lib/models/asset";
import { RecordIdValueSchema } from "$lib/models/api/base_schema";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

const OrphanedAssetDeleteSchema = z.object({
    assetIds: z.array(RecordIdValueSchema).max(500),
});

export async function GET(event: RequestEvent) {
    try {
        const actor = event.locals.user?.actor;
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }
        if (!actor) {
            return json({ message: "Actor not found" }, { status: 401 });
        }

        const assets = await event.locals.pb.collection(Collection.assets).getFullList<Asset>({
            filter: event.locals.pb.filter("author = {:author}", { author: actor }),
            sort: "-created",
            requestKey: null,
        });
        const linkedAssetIds = await linkedAssetIdsForAuthor(event, actor);

        return json(assets.filter((asset) => !linkedAssetIds.has(asset.id)));
    } catch (e: any) {
        return handleError(e);
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const actor = event.locals.user?.actor;
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }
        if (!actor) {
            return json({ message: "Actor not found" }, { status: 401 });
        }

        const data = OrphanedAssetDeleteSchema.parse(await event.request.json());
        const deleted: string[] = [];
        const skipped: string[] = [];
        const failed: { id: string; error: string }[] = [];

        for (const assetId of Array.from(new Set(data.assetIds))) {
            try {
                const asset = await event.locals.pb.collection(Collection.assets).getOne<Asset>(assetId, {
                    requestKey: null,
                });
                if (asset.author !== actor || await assetHasLinks(event, assetId)) {
                    skipped.push(assetId);
                    continue;
                }

                await event.locals.pb.collection(Collection.assets).delete(assetId, { requestKey: null });
                deleted.push(assetId);
            } catch (e: any) {
                failed.push({
                    id: assetId,
                    error: e instanceof Error ? e.message : "delete_failed",
                });
            }
        }

        return json({ deleted, skipped, failed });
    } catch (e: any) {
        return handleError(e);
    }
}

async function linkedAssetIdsForAuthor(event: RequestEvent, actor: string): Promise<Set<string>> {
    const linked = new Set<string>();
    for (const collection of assetLinkCollections()) {
        const links = await event.locals.pb.collection(collection).getFullList<AssetLink>({
            filter: event.locals.pb.filter("asset.author = {:author}", { author: actor }),
            requestKey: null,
        });
        for (const link of links) {
            linked.add(link.asset);
        }
    }
    return linked;
}

async function assetHasLinks(event: RequestEvent, assetId: string): Promise<boolean> {
    for (const collection of assetLinkCollections()) {
        const links = await event.locals.pb.collection(collection).getFullList({
            filter: event.locals.pb.filter("asset = {:asset}", { asset: assetId }),
            requestKey: null,
        });
        if (links.length > 0) {
            return true;
        }
    }
    return false;
}

function assetLinkCollections() {
    return [
        Collection.trail_assets,
        Collection.waypoint_assets,
        Collection.summit_log_assets,
    ];
}
