import { Collection, handleError, remove } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/api-token/{id}:
 *   delete:
 *     summary: Delete API token
 *     description: Deletes an API token by ID
 *     tags:
 *       - API Tokens
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           description: API token ID (15 chars)
 *     responses:
 *       200:
 *         description: Success
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.api_tokens)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
