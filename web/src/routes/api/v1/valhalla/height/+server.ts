import { env } from '$env/dynamic/public';
import { error, json, type NumericRange, type RequestEvent } from "@sveltejs/kit";


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
    const data = await event.request.json()
    if (!env.PUBLIC_VALHALLA_URL) {
        return error(400, "PUBLIC_VALHALLA_URL not set")
    }
    try {
        const r = await event.fetch(env.PUBLIC_VALHALLA_URL + '/height', { method: "POST", body: JSON.stringify(data) });        
        const response = await r.json();
        if (!r.ok) {
            throw error(r.status as NumericRange<400,500>, response);
        }
        return json(response);
    } catch (e: any) {
        throw error(e.status || 500, e)
    }
}