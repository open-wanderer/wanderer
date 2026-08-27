import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/asset-merge/suggest:
 *   post:
 *     summary: Suggest duplicate asset groups
 *     description: Returns groups of photo assets that can be consolidated into one target asset.
 *     tags:
 *       - Asset Merge
 *     responses:
 *       200:
 *         description: Suggested duplicate asset groups
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 */
export async function POST(event: RequestEvent) {
    try {
        const response = await event.locals.pb.send("/asset-merge/suggest", {
            method: "POST",
            body: {},
            fetch: event.fetch,
        });

        return json(response);
    } catch (e: any) {
        return handleError(e);
    }
}
