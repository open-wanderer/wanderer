import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { committedAssetResponseError } from "#lib/server/committed_asset_response.ts";
import { SummitLogUpdateSchema } from "#lib/models/api/summit_log_schema.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { Collection, handleError, remove, show, update } from "#lib/util/api_util.ts";
import { error, type RequestEvent } from "@sveltejs/kit";
import { completeChildAssetResponse } from "#lib/server/trail_asset_pagination.ts";


/**
 * @swagger
 * /api/v1/summit-log/{id}:
 *   get:
 *     summary: Get summit log
 *     tags:
 *       - Summit Logs
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Summit log details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<SummitLog>(event, Collection.summit_logs)
        await completeChildAssetResponse(event, r, "summit_log");
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/summit-log/{id}:
 *   post:
 *     summary: Update summit log
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
 *         application/json:
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
 *         description: Invalid input or `summit_log_photos_form_field_removed` when photo modifier keys are present. A plain `photos` key is ignored. Save photos through PUT /api/v1/assets.
 *       404:
 *         description: Not Found
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
export async function POST(event: RequestEvent) {
    let committedRecord: SummitLog | undefined;
    try {
        const r = await update<SummitLog>(event, SummitLogUpdateSchema, Collection.summit_logs)
        committedRecord = r;
        await completeChildAssetResponse(event, r, "summit_log");
        return json(r);
    } catch (e: any) {
        return committedAssetResponseError(e, committedRecord)
    }
}

/**
 * @swagger
 * /api/v1/summit-log/{id}:
 *   delete:
 *     summary: Delete summit log
 *     tags:
 *       - Summit Logs
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Summit log deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.summit_logs)
        return Response.json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
