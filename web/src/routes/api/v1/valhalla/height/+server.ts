import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { proxyJsonResponse } from '$lib/server/http';
import { json, type RequestEvent } from "@sveltejs/kit";


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
    const baseUrl = getValhallaBaseUrl();
    const data = await event.request.json()
    if (!baseUrl) {
        return json({ message: "VALHALLA_URL not set" }, { status: 400 })
    }
    try {
        const response = await event.fetch(baseUrl + '/height', { method: "POST", body: JSON.stringify(data) });
        return await proxyJsonResponse(response);
    } catch (e: any) {
        return json({ message: "Valhalla request failed" }, { status: 502 })
    }
}
