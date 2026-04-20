import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { proxyJsonResponse } from '$lib/server/http';
import { json, type RequestEvent } from "@sveltejs/kit";

type RouteRequestBody = Record<string, unknown> & {
    include_elevation_profile?: boolean;
};

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
    const baseUrl = getValhallaBaseUrl();
    const data: RouteRequestBody = await event.request.json();
    if (!baseUrl) {
        return json({ message: "VALHALLA_URL not set" }, { status: 400 })
    }

    try {
        const response = await event.fetch(baseUrl + '/route', {
            method: "POST",
            body: JSON.stringify(data)
        });
        return await proxyJsonResponse(response);
    } catch (e: any) {
        return json({ message: "Valhalla request failed" }, { status: 502 })
    }
}
