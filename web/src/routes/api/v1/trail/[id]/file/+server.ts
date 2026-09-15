import { assertFileField, handleError } from "$lib/util/api_util";
import type { RequestEvent } from "@sveltejs/kit";
import { POST as formPOST } from "../../form/[id]/+server";

const fileFields = ["gpx", "photos"] as const;

/**
 * @swagger
 * /api/v1/trail/{id}/file:
 *   post:
 *     summary: Upload trail file
 *     deprecated: true
 *     description: >
 *       Deprecated alias of `POST /api/v1/trail/form/{id}`, which accepts the same multipart body and is the endpoint to use.
 *       Kept for compatibility; behaves like the form endpoint, except that a body without a `gpx` or `photos` part
 *       (PocketBase's `+`/`-` modifiers are accepted) is rejected with 400 `missing_file` instead of being applied as a no-op.
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/TrailUpdateInput'
 *     responses:
 *       200:
 *         description: Trail updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request (no `gpx` or `photos` part in the body, invalid id, or body `id` differs from the path)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    // The form handler reads the body itself, so validate a clone.
    try {
        assertFileField(await event.request.clone().formData(), fileFields);
    } catch (e) {
        return handleError(e);
    }
    return formPOST(event);
}
