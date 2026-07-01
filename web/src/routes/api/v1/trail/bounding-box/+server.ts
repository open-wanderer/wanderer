import { withTrailPreferenceMeiliFilter } from "$lib/server/category_preference_filter";
import type { TrailBoundingBox } from "$lib/models/trail";
import { error, json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail/bounding-box:
 *   get:
 *     summary: Get trail bounding box
 *     description: Retrieves geographic bounding box (lat/lon bounds) for user's trails
 *     tags:
 *       - Trails
 *     responses:
 *       200:
 *         description: Bounding box coordinates
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 max_lat:
 *                   type: number
 *                 min_lat:
 *                   type: number
 *                 max_lon:
 *                   type: number
 *                 min_lon:
 *                   type: number
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    if (!event.locals.pb.authStore.record) {
        return json({
            max_lat: 0,
            min_lat: 0,
            max_lon: 0,
            min_lon: 0,
            has_trails: false,
        });
    }
    try {
        const filter = await withTrailPreferenceMeiliFilter(event, undefined);
        const attributesToRetrieve = ["min_lat", "max_lat", "min_lon", "max_lon"];
        const r = await event.locals.ms.multiSearch({
            queries: [
                {
                    indexUid: "trails",
                    q: "",
                    filter,
                    attributesToRetrieve,
                    sort: ["min_lat:asc"],
                    limit: 1,
                },
                {
                    indexUid: "trails",
                    q: "",
                    filter,
                    attributesToRetrieve,
                    sort: ["max_lat:desc"],
                    limit: 1,
                },
                {
                    indexUid: "trails",
                    q: "",
                    filter,
                    attributesToRetrieve,
                    sort: ["min_lon:asc"],
                    limit: 1,
                },
                {
                    indexUid: "trails",
                    q: "",
                    filter,
                    attributesToRetrieve,
                    sort: ["max_lon:desc"],
                    limit: 1,
                },
            ],
        });

        const [minLatResult, maxLatResult, minLonResult, maxLonResult] = r.results;
        const hasTrails =
            minLatResult.hits.length > 0 &&
            maxLatResult.hits.length > 0 &&
            minLonResult.hits.length > 0 &&
            maxLonResult.hits.length > 0;

        if (!hasTrails) {
            return json({
                min_lat: 0,
                max_lat: 0,
                min_lon: 0,
                max_lon: 0,
                has_trails: false,
            });
        }

        const boundingBox: TrailBoundingBox = {
            min_lat: minLatResult.hits[0].min_lat,
            max_lat: maxLatResult.hits[0].max_lat,
            min_lon: minLonResult.hits[0].min_lon,
            max_lon: maxLonResult.hits[0].max_lon,
            has_trails: true,
        };

        return json(boundingBox)
    } catch (e: any) {
        console.error(e);
        throw error(e.httpStatus || 500, e.message ?? "Unable to get trail bounding box");
    }
}
