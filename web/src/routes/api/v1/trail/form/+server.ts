import { hasRemovedPhotoFields } from "#lib/server/removed_photo_form.ts";
import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { committedAssetResponseError } from "#lib/server/committed_asset_response.ts";
import type { Trail } from '#lib/models/trail.ts';
import { enrichTrailResponse, withTrailAssetExpands } from '#lib/server/trail_response_util.ts';
import { Collection, handleError, uploadCreate } from '#lib/util/api_util.ts';
import { type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail/form:
 *   put:
 *     summary: Create trail with file upload
 *     description: >
 *       Photo fields and their modifiers, including @jsonPayload, return 400 `trail_photos_form_field_removed` before any write. Use PUT /api/v1/assets to attach photos after saving.
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/TrailCreateInput'
 *     responses:
 *       201:
 *         description: Trail created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Save or response refresh failed; detail.recordCommitted and detail.savedRecord identify an already persisted record.
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AssetWriteError'
 *       default:
 *         description: Any error status may carry detail.recordCommitted and detail.savedRecord when the write succeeded before response refresh failed. Use the saved ID on retry to avoid creating a duplicate.
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AssetWriteError'
 */
export async function PUT(event: RequestEvent) {
    if (!event.locals.user) return Response.json({ message: "Unauthorized" }, { status: 401 });
    let committedRecord: Trail | undefined;
    try {
        const data = await event.request.clone().formData();
        if (hasRemovedPhotoFields(data)) { return Response.json({ message: "trail_photos_form_field_removed" }, { status: 400 }); }

        let r = await uploadCreate<Trail>(event, Collection.trails)
        committedRecord = r;
        r = await event.locals.pb.collection(Collection.trails).getOne<Trail>(r.id!, {
            expand: withTrailAssetExpands({
                expand: event.url.searchParams.get("expand") ?? undefined,
            }).expand,
        });
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return committedAssetResponseError(e, committedRecord)
    }
}

function enrichRecord(r: Trail) {
    enrichTrailResponse(r);
}
