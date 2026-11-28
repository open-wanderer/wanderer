import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { committedAssetResponseError } from "#lib/server/committed_asset_response.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { Collection, handleError, uploadUpdate } from "#lib/util/api_util.ts";
import { applyGpxToForm, summitLogGpxFields } from "#lib/util/gpx_util.ts";
import { RecordIdSchema } from "#lib/models/api/base_schema.ts";
import { type RequestEvent } from "@sveltejs/kit";
import { completeChildAssetResponse } from "#lib/server/trail_asset_pagination.ts";
import { elevationFetch, withParseSlot } from "#lib/server/parse_slot.ts";

/**
 * @swagger
 * /api/v1/summit-log/form/{id}:
 *   post:
 *     summary: Update summit log with file upload
 *     description: >
 *       Updates a summit log with file upload (photos/GPX) and date normalization. The record is addressed by the `id` path parameter; an `id` in the body is optional and must match it (400 `id_mismatch` otherwise).
 *       A `gpx` part replaces the track: the file is converted to GPX (FIT, KML, KMZ and TCX are accepted) and `distance`, `duration`, `elevation_gain` and `elevation_loss` are derived from it unless the request sets them explicitly.
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
 *         description: Bad Request (invalid id, body `id` differs from the path, or the `gpx` file cannot be parsed)
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Not Found
 *       413:
 *         description: The `gpx` file is too large to parse
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
 *       503:
 *         description: Too many files are waiting to be parsed
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return json({ message: "Unauthorized" }, { status: 401 });
    }
    let committedRecord: SummitLog | undefined;
    try {
        RecordIdSchema.parse(event.params);
        const data = await event.request.formData();
        if (data.has("gpx")) {
            await withParseSlot(() => applyGpxToForm(data, summitLogGpxFields, true, elevationFetch(event.fetch)));
        }
        const r = await uploadUpdate<SummitLog>(event, Collection.summit_logs, data)
        committedRecord = r;
        await completeChildAssetResponse(event, r, "summit_log");
        r.date = r.date?.substring(0, 10) ?? "";
        return json(r);
    } catch (e) {
        return committedAssetResponseError(e, committedRecord)
    }
}
