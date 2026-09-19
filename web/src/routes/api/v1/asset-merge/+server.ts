import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/asset-merge:
 *   post:
 *     summary: Merge duplicate assets
 *     description: Reassigns all links from source assets to a target asset and deletes the source assets.
 *     tags:
 *       - Asset Merge
 *     responses:
 *       200:
 *         description: Merge acknowledged
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 */
export async function POST(event: RequestEvent) {
    try {
        const body = await event.request.json();
        const response = await event.locals.pb.send("/asset-merge", {
            method: "POST",
            body,
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
