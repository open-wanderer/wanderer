import { withTrailPreferenceMeiliFilter } from "$lib/server/category_preference_filter";
import type { TrailBoundingBox } from "$lib/models/trail";
import type { Actor } from "$lib/models/activitypub/actor";
import { getActorResponseForHandle } from "$lib/util/activitypub_server_util";
import { error, json, type RequestEvent } from "@sveltejs/kit";

const worldViewBoundingBox: TrailBoundingBox = {
    min_lat: 0,
    max_lat: 0,
    min_lon: 0,
    max_lon: 0,
    has_trails: false,
};

function isFiniteInRange(value: unknown, min: number, max: number): value is number {
    return typeof value === "number" && Number.isFinite(value) && value >= min && value <= max;
}

function isValidBoundingBoxCoordinates(body: any): boolean {
    return (
        isFiniteInRange(body?.min_lat, -90, 90) &&
        isFiniteInRange(body?.max_lat, -90, 90) &&
        isFiniteInRange(body?.min_lon, -180, 180) &&
        isFiniteInRange(body?.max_lon, -180, 180) &&
        body.min_lat <= body.max_lat &&
        body.min_lon <= body.max_lon
    );
}

async function boundingBoxForFederatedActor(event: RequestEvent, actor: Actor): Promise<TrailBoundingBox> {
    const origin = new URL(actor.iri).origin;
    try {
        const url = `${origin}/api/v1/trail/bounding-box?handle=${encodeURIComponent(actor.preferred_username)}`;
        const response = await event.fetch(url, { signal: AbortSignal.timeout(4000) });

        if (!response.ok) {
            console.warn(`profile trail bounding-box: federated proxy to ${origin} responded with ${response.status}`);
            return worldViewBoundingBox;
        }

        const body = await response.json();
        if (!body?.has_trails || !isValidBoundingBoxCoordinates(body)) {
            console.warn(`profile trail bounding-box: federated proxy to ${origin} returned an unusable payload`);
            return worldViewBoundingBox;
        }

        return {
            min_lat: body.min_lat,
            max_lat: body.max_lat,
            min_lon: body.min_lon,
            max_lon: body.max_lon,
            has_trails: true,
        };
    } catch (e) {
        console.warn(`profile trail bounding-box: federated proxy to ${origin} failed`, e);
        return worldViewBoundingBox;
    }
}

/**
 * @swagger
 * /api/v1/trail/bounding-box:
 *   get:
 *     summary: Get trail bounding box
 *     description: Retrieves geographic bounding box (lat/lon bounds) for user's trails, or, when
 *       `handle` is supplied, for a single profile's trails (proxied to the origin instance when
 *       that profile is federated).
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: query
 *         name: handle
 *         required: false
 *         schema:
 *           type: string
 *         description: Optional profile handle to scope the bounding box to a single author's trails.
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
 *                 has_trails:
 *                   type: boolean
 *                   description: False when there are no trails to bound, or when a federated
 *                     lookup could not be completed — in either case the other four fields are 0.
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    if (!event.locals.pb.authStore.record) {
        return json(worldViewBoundingBox);
    }

    const handle = event.url.searchParams.get("handle");

    try {
        let baseFilter: string | string[] | undefined = undefined;

        if (handle) {
            try {
                const { actor } = await getActorResponseForHandle(event, handle);

                if (!actor.is_local) {
                    return json(await boundingBoxForFederatedActor(event, actor));
                }

                baseFilter = [`author = ${actor.id}`];
            } catch (e) {
                console.warn("profile trail bounding-box: failed to resolve actor for handle", e);
                return json(worldViewBoundingBox);
            }
        }

        const filter = await withTrailPreferenceMeiliFilter(event, baseFilter);
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
            return json(worldViewBoundingBox);
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
