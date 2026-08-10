import { proxyJsonResponse } from '$lib/server/http';
import { getValhallaUrl } from '$lib/server/valhalla';
import { error, json, type RequestEvent } from "@sveltejs/kit";


/**
 * @swagger
 * /api/v1/valhalla/height:
 *   post:
 *     summary: Get elevation data
 *     description: Queries Valhalla service for elevation data at coordinates
 *     tags:
 *       - Valhalla
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *     responses:
 *       200:
 *         description: Elevation data from Valhalla
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return error(401, "Unauthorized");
    }

    const heightUrl = getValhallaUrl() + '/height';
    const data = await event.request.json()
    if (!heightUrl) {
        return json({ message: "VALHALLA_URL not set" }, { status: 400 })
    }
    try {
        const response = await event.fetch(heightUrl, { method: "POST", body: JSON.stringify(data) });
        return await proxyJsonResponse(response);
    } catch (e: any) {
        return json({ message: "Valhalla request failed" }, { status: 502 })
    }
}
