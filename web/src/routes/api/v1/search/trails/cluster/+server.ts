import { withTrailPreferenceMeiliFilter } from "$lib/server/category_preference_filter";
import { error, json, type RequestEvent } from "@sveltejs/kit";
import Supercluster from "supercluster";
import { MAP_MAX_POLYLINES } from "$lib/config/map";
import { buildTrailSearchBoundsFilter } from "$lib/util/trail_search_util";

function isFiniteNumber(value: unknown): value is number {
    return typeof value === "number" && Number.isFinite(value);
}

function isValidLngLat(value: any): value is { lat: number; lng: number } {
    return isFiniteNumber(value?.lat) && isFiniteNumber(value?.lng);
}

export async function POST(event: RequestEvent) {
    const data = await event.request.json()
    const { southWest, northEast, zoom, filterText, q = "" } = data;

    if (!southWest || !northEast || zoom === undefined) {
        throw error(400, "Missing required parameters: southWest, northEast, zoom");
    }

    if (!isValidLngLat(southWest) || !isValidLngLat(northEast) || !isFiniteNumber(zoom)) {
        throw error(400, "Invalid cluster bounds or zoom");
    }

    try {
        const geoFilter = buildTrailSearchBoundsFilter({ northEast, southWest });
        const baseFilters = [geoFilter, filterText].filter(f => f && f !== "");
        const summaryQuery = {
            indexUid: "trails",
            q,
            filter: await withTrailPreferenceMeiliFilter(
                event,
                baseFilters,
            ),
            attributesToRetrieve: ["id", "_geo", "bounding_box_diagonal"],
            limit: 10000,
        };
        // Existence outside the viewport is reliable even when result counts
        // reach Meilisearch's pagination limit. Apply the same query and access
        // constraints, so unrelated or inaccessible trails cannot activate scope.
        const outsideFilters = [`NOT (${geoFilter})`, filterText].filter(Boolean);
        const outsideQuery = {
            ...summaryQuery,
            filter: await withTrailPreferenceMeiliFilter(event, outsideFilters),
            attributesToRetrieve: ["id"],
            limit: 1,
        };
        const response = await event.locals.ms.multiSearch({
            queries: [summaryQuery, outsideQuery],
        });
        const hits = response.results[0].hits;
        const totalHits = response.results[0].estimatedTotalHits ?? response.results[0].totalHits;
        const hasOutsideBounds = response.results[1].hits.length > 0;

        const clusteringMaxZoom = event.locals.settings?.behavior?.mapClusteringMaxZoom ?? 11;
        const forceClustering = zoom < clusteringMaxZoom;

        // Dynamic Threshold: Sort by diagonal and pick top N for polylines
        const sortedHits = [...hits].sort((a: any, b: any) => (b.bounding_box_diagonal ?? 0) - (a.bounding_box_diagonal ?? 0));
        
        const largeHits = forceClustering ? [] : sortedHits.slice(0, MAP_MAX_POLYLINES);
        const smallHits = forceClustering ? sortedHits : sortedHits.slice(MAP_MAX_POLYLINES);

        const smallFeatures: GeoJSON.Feature<GeoJSON.Point, any>[] = smallHits.map((h: any) => ({
            type: "Feature",
            properties: {
                id: h.id, 
                bounding_box_diagonal: h.bounding_box_diagonal ?? 0
            },
            geometry: {
                type: "Point",
                coordinates: [h._geo.lng, h._geo.lat]
            }
        }));

        const index = new Supercluster({
            radius: 40, // Less aggressive clustering
            maxZoom: 16,
        });

        index.load(smallFeatures);

        const bbox: [number, number, number, number] = southWest.lng > northEast.lng
            ? [-180, southWest.lat, 180, northEast.lat]
            : [southWest.lng, southWest.lat, northEast.lng, northEast.lat];

        const clusters = index.getClusters(
            bbox,
            Math.floor(zoom)
        );

        function abbreviateCount(count: number): string {
            if (count >= 1000) {
                return (count / 1000).toFixed(1) + "k";
            }
            return count.toString();
        }

        const normalizedSmallFeatures = clusters.map((f: any) => {
            if (f.properties.cluster) {
                f.properties.point_count_abbreviated = abbreviateCount(f.properties.point_count);
            } else {
                f.properties.point_count = 1;
                f.properties.point_count_abbreviated = "1";
                f.properties.is_large = false;
            }
            return f;
        });

        // Step 3: Individual markers for large trails (NOT clustered)
        const largeFeatures: GeoJSON.Feature<GeoJSON.Point, any>[] = largeHits.map((h: any) => ({
            type: "Feature",
            properties: {
                id: h.id,
                cluster: false,
                is_large: true,
                point_count: 1,
                point_count_abbreviated: "1",
                bounding_box_diagonal: h.bounding_box_diagonal ?? 0
            },
            geometry: {
                type: "Point",
                coordinates: [h._geo.lng, h._geo.lat] // Back to stable anchor point
            }
        }));

        return json({
            type: "FeatureCollection",
            features: [...normalizedSmallFeatures, ...largeFeatures],
            totalHits,
            hasOutsideBounds,
        });
    } catch (e: any) {
        console.error("Clustering error:", e);
        throw error(e.httpStatus || 500, e.message ?? "Unable to cluster trails");
    }
}
