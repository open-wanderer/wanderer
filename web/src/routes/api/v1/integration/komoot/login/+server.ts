import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/integration/komoot/login:
 *   get:
 *     summary: Get Komoot login endpoint
 *     description: Proxies to backend to get Komoot login configuration
 *     tags:
 *       - Integrations
 *     responses:
 *       200:
 *         description: Komoot login endpoint
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await event.locals.pb.send("/integration/komoot/login", {
            method: "GET",
        });
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}