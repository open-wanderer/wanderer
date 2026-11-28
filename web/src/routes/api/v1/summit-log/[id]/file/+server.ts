import { hasRemovedPhotoFields } from "#lib/server/removed_photo_form.ts";
import { assertFileField, handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { POST as formPOST } from "../../form/[id]/+server";

const fileFields = ["gpx"] as const;

/**
 * @swagger
 * /api/v1/summit-log/{id}/file:
 *   post:
 *     summary: Upload summit log file
 *     deprecated: true
 *     description: >
 *       Deprecated alias of `POST /api/v1/summit-log/form/{id}`, which accepts the same multipart body and is the endpoint to use.
 *       Kept for compatibility; behaves like the form endpoint, except that a body without a `gpx` part
 *       (PocketBase's `+`/`-` modifiers are accepted) is rejected with 400 `missing_file`. Photo fields return 400 `summit_log_photos_form_field_removed` before any write.
 *     tags:
 *       - Summit Logs
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
 *             $ref: '#/components/schemas/SummitLogUpdateInput'
 *     responses:
 *       200:
 *         description: Summit log updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
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
        if (hasRemovedPhotoFields(data)) return Response.json({ message: "summit_log_photos_form_field_removed" }, { status: 400 });
        assertFileField(data, fileFields);
    } catch (e) {
        return handleError(e);
    }
    return formPOST(event);
}
