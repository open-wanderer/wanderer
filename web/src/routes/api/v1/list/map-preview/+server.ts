import { MAP_MAX_POLYLINES } from "$lib/config/map";
import {
    applyPolylineBudget,
    meiliIdInFilter,
    unionTrailBounds,
    type PreviewTrailGeometry,
} from "$lib/util/list_map_preview_util";
import { error, json, type RequestEvent } from "@sveltejs/kit";

type ListHit = {
    id: string;
    trail_ids?: string[];
};

type TrailHit = {
    id: string;
    polyline?: string;
    lat?: number;
    lon?: number;
    min_lat?: number;
    max_lat?: number;
    min_lon?: number;
    max_lon?: number;
};

/**
 * Compose list overview map geometry from the trail index under the viewer's
 * Meilisearch tenant token. List documents only supply trail_ids.
 */
export async function POST(event: RequestEvent) {
    const data = await event.request.json().catch(() => null);
    const listIds = Array.isArray(data?.list_ids)
        ? (data.list_ids as unknown[]).filter(
              (id): id is string => typeof id === "string" && id.length > 0,
          )
        : [];

    if (listIds.length === 0) {
        return json({ lists: [], truncated: false });
    }

    try {
        const listSearch = await event.locals.ms.index("lists").search("", {
            filter: meiliIdInFilter(listIds),
            attributesToRetrieve: ["id", "trail_ids"],
            hitsPerPage: listIds.length,
        });

        const listTrailIds = new Map<string, string[]>();
        const allTrailIds = new Set<string>();

        for (const hit of listSearch.hits as ListHit[]) {
            const ids = (hit.trail_ids ?? []).filter(Boolean);
            listTrailIds.set(hit.id, ids);
            for (const id of ids) {
                allTrailIds.add(id);
            }
        }

        // Preserve request order for stable responses.
        for (const id of listIds) {
            if (!listTrailIds.has(id)) {
                listTrailIds.set(id, []);
            }
        }

        const trailById = new Map<string, PreviewTrailGeometry>();
        const trailIdList = [...allTrailIds];
        const batchSize = 100;

        for (let i = 0; i < trailIdList.length; i += batchSize) {
            const batch = trailIdList.slice(i, i + batchSize);
            const result = await event.locals.ms.index("trails").search("", {
                filter: meiliIdInFilter(batch),
                attributesToRetrieve: [
                    "id",
                    "polyline",
                    "lat",
                    "lon",
                    "min_lat",
                    "max_lat",
                    "min_lon",
                    "max_lon",
                ],
                hitsPerPage: batch.length,
            });

            for (const hit of result.hits as TrailHit[]) {
                trailById.set(hit.id, {
                    id: hit.id,
                    polyline: hit.polyline || undefined,
                    lat: hit.lat,
                    lon: hit.lon,
                    min_lat: hit.min_lat,
                    max_lat: hit.max_lat,
                    min_lon: hit.min_lon,
                    max_lon: hit.max_lon,
                });
            }
        }

        // Apply a single global polyline budget across unique trails.
        const uniqueTrails = [...trailById.values()];
        const { truncated } = applyPolylineBudget(
            uniqueTrails,
            MAP_MAX_POLYLINES,
        );

        const lists = listIds.map((listId) => {
            const trails = (listTrailIds.get(listId) ?? [])
                .map((trailId) => trailById.get(trailId))
                .filter((trail): trail is PreviewTrailGeometry => !!trail);
            const bounds = unionTrailBounds(trails);
            return {
                id: listId,
                trails,
                ...(bounds ? { bounds } : {}),
            };
        });

        return json({ lists, truncated });
    } catch (e: any) {
        console.error(e);
        throw error(e.httpStatus || 500, e);
    }
}
