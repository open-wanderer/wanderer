import type { Asset } from "$lib/models/asset";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }
        if (!event.locals.user.actor) {
            return json({ message: "Actor not found" }, { status: 401 });
        }

        const assets = await event.locals.pb.collection(Collection.assets).getFullList<Asset>({
            filter: event.locals.pb.filter("author = {:author} && type = 'photo'", {
                author: event.locals.user.actor,
            }),
            sort: "created",
            requestKey: null,
        });

        return json(assets.filter(assetNeedsMetadataBackfill));
    } catch (e: any) {
        return handleError(e);
    }
}

function assetNeedsMetadataBackfill(asset: Asset): boolean {
    if (!asset.file) {
        return false;
    }
    if (hasExifBackfillMarker(asset)) {
        return false;
    }
    return !assetHasCoordinates(asset) || !asset.taken_at || isLegacyMigratedAsset(asset);
}

function isLegacyMigratedAsset(asset: Asset): boolean {
    return typeof asset.metadata?.source_collection === "string";
}

function assetHasCoordinates(asset: Asset): boolean {
    return (
        typeof asset.lat === "number" &&
        typeof asset.lon === "number" &&
        Number.isFinite(asset.lat) &&
        Number.isFinite(asset.lon) &&
        (asset.lat !== 0 || asset.lon !== 0)
    );
}

function hasExifBackfillMarker(asset: Asset): boolean {
    const marker = asset.metadata?.exif_backfill;
    return Boolean(
        marker &&
        typeof marker === "object" &&
        !Array.isArray(marker) &&
        typeof (marker as { checked_at?: unknown }).checked_at === "string",
    );
}
