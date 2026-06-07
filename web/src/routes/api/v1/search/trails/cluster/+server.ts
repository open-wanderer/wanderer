import { error, json, type RequestEvent } from "@sveltejs/kit";
import Supercluster from "supercluster";
import { MAP_MAX_POLYLINES } from "$lib/config/map";

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
        let lonFilter = `max_lon >= ${southWest.lng} AND min_lon <= ${northEast.lng}`;
        if (southWest.lng > northEast.lng) {
            lonFilter = `(max_lon >= ${southWest.lng} OR min_lon <= ${northEast.lng})`;
        }
    
        const geoFilter = `max_lat >= ${southWest.lat} AND min_lat <= ${northEast.lat} AND ${lonFilter}`;
        
        const summaryQuery = {
            indexUid: "trails",
            q,
            filter: [geoFilter, filterText].filter(f => f && f !== ""),
            attributesToRetrieve: ["id", "_geo", "bounding_box_diagonal"],
            limit: 10000, 
        };

        const r = await event.locals.ms.multiSearch({
            queries: [summaryQuery]
        });

        const hits = r.results[0].hits;
        
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
            totalHits: r.results[0].estimatedTotalHits ?? r.results[0].totalHits
        });
    } catch (e: any) {
        console.error("Clustering error:", e);
        throw error(e.httpStatus || 500, e.message ?? "Unable to cluster trails");
    }
}
