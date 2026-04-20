import { WaypointCreateSchema } from '$lib/models/api/waypoint_schema';
import type { Waypoint } from '$lib/models/waypoint';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/waypoint:
 *   get:
 *     summary: List waypoints
 *     tags:
 *       - Waypoints
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List of waypoints
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<Waypoint>(event, Collection.waypoints);
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/waypoint:
 *   put:
 *     summary: Create waypoint
 *     tags:
 *       - Waypoints
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/WaypointInput'
 *     responses:
 *       201:
 *         description: Waypoint created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Waypoint'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<Waypoint>(event, WaypointCreateSchema, Collection.waypoints)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}