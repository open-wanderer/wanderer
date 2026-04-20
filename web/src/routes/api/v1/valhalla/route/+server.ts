import { env } from '$env/dynamic/public';
import { error, json, type NumericRange, type RequestEvent } from "@sveltejs/kit";


/**
 * @swagger
 * /api/v1/valhalla/route:
 *   post:
 *     summary: Get route data
 *     description: Queries Valhalla service for routing data
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
 *         description: Route data from Valhalla
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
        return json({ message: "PUBLIC_VALHALLA_URL not set" }, { status: 400 })
    }
    try {
        const r = await event.fetch(env.PUBLIC_VALHALLA_URL + '/route', { method: "POST", body: JSON.stringify(data) });
        const response = await r.json();
        if (!r.ok) {
            return json({ message: response }, { status: r.status })

        }
        return json(response);
    } catch (e: any) {
        return json({ message: e }, { status: 500 })
    }
}