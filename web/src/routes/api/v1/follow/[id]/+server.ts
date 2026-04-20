import { Collection, handleError, remove } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";


/**
 * @swagger
 * /api/v1/follow/{id}:
 *   delete:
 *     summary: Delete follow
 *     tags:
 *       - Follows
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Follow deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.follows)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

