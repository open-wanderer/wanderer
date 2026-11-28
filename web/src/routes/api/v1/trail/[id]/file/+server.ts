import { hasRemovedPhotoFields } from "#lib/server/removed_photo_form.ts";
import { assertFileField, handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { POST as formPOST } from "../../form/[id]/+server";

const fileFields = ["gpx"] as const;

/**
 * @swagger
 * /api/v1/trail/{id}/file:
 *   post:
 *     summary: Upload trail file
 *     deprecated: true
 *     description: >
 *       Deprecated alias of `POST /api/v1/trail/form/{id}`, which accepts the same multipart body and is the endpoint to use.
 *       Kept for compatibility; behaves like the form endpoint, except that a body without a `gpx` part
 *       (PocketBase's `+`/`-` modifiers are accepted) is rejected with 400 `missing_file`. Photo fields return 400 `trail_photos_form_field_removed` before any write.
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
 *         description: Bad Request (no `gpx` part in the body, invalid id, or body `id` differs from the path)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) return Response.json({ message: "Unauthorized" }, { status: 401 });
    // The form handler reads the body itself, so validate a clone.
    try {
        const data = await event.request.clone().formData();
        if (hasRemovedPhotoFields(data)) return Response.json({ message: "trail_photos_form_field_removed" }, { status: 400 });
        assertFileField(data, fileFields);
    } catch (e) {
        return handleError(e);
    }
    return formPOST(event);
}
