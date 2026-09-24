import GPX from "$lib/models/gpx/gpx";
import type { Trail, TrailSearchResult } from "$lib/models/trail";
import { searchLocationReverse } from "$lib/stores/search_store";
import { trails_create } from "$lib/stores/trail_store";
import { handleError } from "$lib/util/api_util";
import { fromFile, gpx2trail } from "$lib/util/gpx_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import type { Meilisearch } from "meilisearch";
import { ClientResponseError } from "pocketbase";

/**
 * @swagger
 * /api/v1/trail/upload:
 *   put:
 *     summary: Upload and parse GPX file as trail
 *     description: Uploads a GPX file, parses it to extract trail data, performs duplicate detection, and indexes in search
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *               name:
 *                 type: string
 *               ignoreDuplicates:
 *                 type: boolean
 *     responses:
 *       201:
 *         description: Trail created from GPX
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request - Invalid or empty GPX file
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    if (!event.locals.user) {
        return json({ message: "Unauthorized" }, { status: 401 });
    }

    try {
        const data = await event.request.formData();

        const { gpxData, gpxFile } = await fromFile(data.get("file") as Blob)

        if (!gpxData.length) {
            throw new ClientResponseError({ status: 400, response: { message: "Empty file" } })
        }
        let parseResult: { trail: Trail, gpx: GPX };
        try {
            parseResult = await gpx2trail(gpxData, data.get("name") as string | undefined, true, event.fetch);
        } catch (e: any) {
            console.error(e)
            throw new ClientResponseError({ status: 400, response: { message: "Invalid file" } })
        }
        let trail = parseResult.trail;

        const ignoreDuplicates = data.get("ignoreDuplicates") === "true"
        if (!ignoreDuplicates) {
            let duplicate: TrailSearchResult | null = null;
            try {
                duplicate = await findDuplicate(event.locals.ms, trail)
            } catch (e: any) {
                throw new ClientResponseError({ status: 500, response: { message: "Error checking for duplicates" } })
            }
            if (duplicate !== null) {
                throw new ClientResponseError({ status: 400, response: { message: `Duplicate trail`, id: duplicate.id, name: duplicate.name, domain: `${duplicate.author_name}${duplicate.domain ? '@' + duplicate.domain : ''}` }, })
            }
        }

        if (trail.lat && trail.lon) {
            try {
                const location = await searchLocationReverse(
                    trail.lat,
                    trail.lon,
                    {},
                    event.fetch,
                )
                trail.location ??= location;
            } catch (e: any) {
                console.warn("Reverse geocoding failed during upload", e);
            }
        }

        trail.public = event.locals.settings.privacy?.trails == "public"

        // const log = new SummitLog(trail.date as string, {
        //     distance: trail.distance,
        //     elevation_gain: trail.elevation_gain,
        //     elevation_loss: trail.elevation_loss,
        //     duration: trail.duration ? trail.duration * 60 : undefined,
        // })
        // log.expand!.gpx_data = gpxData;
        // const fileName = (data.get("name") as string | null)?.length ? data.get("name") as string : "file"
        // log._gpx = new File([gpxFile], fileName);

        // trail.expand!.summit_logs?.push(log);


        try {
            trail = await trails_create(trail, [], gpxFile, event.fetch, event.locals.user);
        } catch (e: any) {
            console.error(e)
            return handleError(e)
        }
        return json(trail);

    } catch (e: any) {
        return handleError(e)
    }
}

async function findDuplicate(ms: Meilisearch, t1: Trail) {
    const distance = t1.distance ?? 0;
    const elevationGain = t1.elevation_gain ?? 0;
    const elevationLoss = t1.elevation_loss ?? 0;
    // Filter before limiting so any matching, tenant-visible trail suffices.
    // The 100 m radius follows Meilisearch's inclusive geo boundary; the
    // distance and elevation differences remain strictly less than 50 m.
    const response = await ms.index("trails").search<TrailSearchResult>("", {
        filter: [
            `_geoRadius(${t1.lat ?? 0}, ${t1.lon ?? 0}, 100)`,
            `distance > ${distance - 50} AND distance < ${distance + 50}`,
            `elevation_gain > ${elevationGain - 50} AND elevation_gain < ${elevationGain + 50}`,
            `elevation_loss > ${elevationLoss - 50} AND elevation_loss < ${elevationLoss + 50}`,
        ],
        attributesToRetrieve: ["id", "name", "author_name", "domain"],
        limit: 1,
    });
    return response.hits[0] ?? null;
}
