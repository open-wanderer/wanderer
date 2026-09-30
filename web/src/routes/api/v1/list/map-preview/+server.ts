import { MAP_MAX_POLYLINES } from "$lib/config/map";
import {
    applyPolylineBudget,
    meiliIdInFilter,
    meiliIriInFilter,
    unionTrailBounds,
    type PreviewTrailGeometry,
} from "$lib/util/list_map_preview_util";
import { error, json, type RequestEvent } from "@sveltejs/kit";

type ListHit = {
    id: string;
    trail_ids?: string[];
    iri?: string;
};

type TrailHit = {
    id: string;
    iri?: string;
    polyline?: string;
    min_lat?: number;
    max_lat?: number;
    min_lon?: number;
    max_lon?: number;
    _geo?: { lat?: number; lng?: number };
};

function trailFromHit(hit: TrailHit): PreviewTrailGeometry {
    return {
        id: hit.id,
        iri: hit.iri,
        polyline: hit.polyline || undefined,
        lat: hit._geo?.lat,
        lon: hit._geo?.lng,
        min_lat: hit.min_lat,
        max_lat: hit.max_lat,
        min_lon: hit.min_lon,
        max_lon: hit.max_lon,
    };
}

function guessTrailIris(listIri: string | undefined, trailRef: string): string[] {
    const iris = new Set<string>();
    if (trailRef.startsWith("http://") || trailRef.startsWith("https://")) {
        iris.add(trailRef);
        return [...iris];
    }
    if (listIri) {
        try {
            const origin = new URL(listIri).origin;
            iris.add(`${origin}/api/v1/trail/${trailRef}`);
            iris.add(`${origin}/api/v1/trails/${trailRef}`);
        } catch {
            // ignore invalid list IRI
        }
    }
    return [...iris];
}

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
            attributesToRetrieve: ["id", "trail_ids", "iri"],
            hitsPerPage: listIds.length,
        });

        const listTrailIds = new Map<string, string[]>();
        const listIriById = new Map<string, string | undefined>();
        const allTrailRefs = new Set<string>();

        for (const hit of listSearch.hits as ListHit[]) {
            const ids = (hit.trail_ids ?? []).filter(Boolean);
            listTrailIds.set(hit.id, ids);
            listIriById.set(hit.id, hit.iri);
            for (const id of ids) {
                allTrailRefs.add(id);
            }
        }

        // Preserve request order for stable responses.
        for (const id of listIds) {
            if (!listTrailIds.has(id)) {
                listTrailIds.set(id, []);
            }
        }

        const trailById = new Map<string, PreviewTrailGeometry>();
        const trailByIri = new Map<string, PreviewTrailGeometry>();
        const trailRefList = [...allTrailRefs];
        const batchSize = 100;

        const storeHit = (hit: TrailHit) => {
            const trail = trailFromHit(hit);
            trailById.set(hit.id, trail);
            if (hit.iri) {
                trailByIri.set(hit.iri, trail);
            }
        };

        for (let i = 0; i < trailRefList.length; i += batchSize) {
            const batch = trailRefList.slice(i, i + batchSize);
            const result = await event.locals.ms.index("trails").search("", {
                filter: meiliIdInFilter(batch),
                attributesToRetrieve: [
                    "id",
                    "iri",
                    "polyline",
                    "_geo",
                    "min_lat",
                    "max_lat",
                    "min_lon",
                    "max_lon",
                ],
                hitsPerPage: batch.length,
            });

            for (const hit of result.hits as TrailHit[]) {
                storeHit(hit);
            }
        }

        // Federated lists may store remote trail ids; resolve misses via IRI.
        const missingRefs = trailRefList.filter((ref) => !trailById.has(ref));
        if (missingRefs.length > 0) {
            const iris = new Set<string>();
            for (const [listId, refs] of listTrailIds) {
                const listIri = listIriById.get(listId);
                for (const ref of refs) {
                    if (trailById.has(ref)) {
                        continue;
                    }
                    for (const iri of guessTrailIris(listIri, ref)) {
                        iris.add(iri);
                    }
                }
            }

            const iriList = [...iris];
            for (let i = 0; i < iriList.length; i += batchSize) {
                const batch = iriList.slice(i, i + batchSize);
                const result = await event.locals.ms.index("trails").search("", {
                    filter: meiliIriInFilter(batch),
                    attributesToRetrieve: [
                        "id",
                        "iri",
                        "polyline",
                        "_geo",
                        "min_lat",
                        "max_lat",
                        "min_lon",
                        "max_lon",
                    ],
                    hitsPerPage: batch.length,
                });

                for (const hit of result.hits as TrailHit[]) {
                    storeHit(hit);
                }
            }
        }

        const resolveTrail = (ref: string, listId: string) => {
            const byId = trailById.get(ref);
            if (byId) {
                return byId;
            }
            for (const iri of guessTrailIris(listIriById.get(listId), ref)) {
                const byIri = trailByIri.get(iri);
                if (byIri) {
                    return byIri;
                }
            }
            return undefined;
        };

        // Apply a single global polyline budget across unique trails.
        const uniqueTrails = [...trailById.values()];
        const { truncated } = applyPolylineBudget(
            uniqueTrails,
            MAP_MAX_POLYLINES,
        );

        const lists = listIds.map((listId) => {
            const trails = (listTrailIds.get(listId) ?? [])
                .map((trailRef) => resolveTrail(trailRef, listId))
                .filter((trail): trail is PreviewTrailGeometry => !!trail);
            // Dedupe if the same trail resolved twice via id/iri.
            const seen = new Set<string>();
            const deduped = trails.filter((trail) => {
                if (seen.has(trail.id)) {
                    return false;
                }
                seen.add(trail.id);
                return true;
            });
            const bounds = unionTrailBounds(deduped);
            return {
                id: listId,
                trails: deduped,
                ...(bounds ? { bounds } : {}),
            };
        });

        return json({ lists, truncated });
    } catch (e: any) {
        console.error(e);
        throw error(e.httpStatus || 500, e);
    }
}
