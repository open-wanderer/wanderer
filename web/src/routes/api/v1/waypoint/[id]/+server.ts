import { WaypointUpdateSchema } from "$lib/models/api/waypoint_schema";
import type { Waypoint } from "$lib/models/waypoint";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

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
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Waypoint>(event, WaypointUpdateSchema, Collection.waypoints)
        return json(r);
    } catch (e: any) {
        return handleError(e)
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
