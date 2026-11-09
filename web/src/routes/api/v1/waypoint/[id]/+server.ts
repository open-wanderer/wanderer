import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { committedAssetResponseError } from "#lib/server/committed_asset_response.ts";
import { WaypointUpdateSchema } from "#lib/models/api/waypoint_schema.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { Collection, handleError, remove, show, update } from "#lib/util/api_util.ts";
import { type RequestEvent } from "@sveltejs/kit";
import { completeChildAssetResponse } from "#lib/server/trail_asset_pagination.ts";

/**
 * @swagger
 * /api/v1/waypoint/{id}:
 *   get:
 *     summary: Get waypoint
 *     tags:
 *       - Waypoints
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
 *         description: Waypoint details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Waypoint'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<Waypoint>(event, Collection.waypoints)
        await completeChildAssetResponse(event, r, "waypoint");
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/waypoint/{id}:
 *   post:
 *     summary: Update waypoint
 *     tags:
 *       - Waypoints
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
 *             $ref: '#/components/schemas/WaypointUpdateInput'
 *     responses:
 *       200:
 *         description: Waypoint updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Waypoint'
 *       400:
 *         description: Invalid input or `waypoint_photos_form_field_removed` when photo modifier keys are present. A plain `photos` key is ignored. Save photos through PUT /api/v1/assets.
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
    let committedRecord: Waypoint | undefined;
    try {
        const r = await update<Waypoint>(event, WaypointUpdateSchema, Collection.waypoints)
        committedRecord = r;
        await completeChildAssetResponse(event, r, "waypoint");
        return json(r);
    } catch (e: any) {
        return committedAssetResponseError(e, committedRecord)
    }
}

/**
 * @swagger
 * /api/v1/waypoint/{id}:
 *   delete:
 *     summary: Delete waypoint
 *     tags:
 *       - Waypoints
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Waypoint deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.waypoints)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
