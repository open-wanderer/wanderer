import {
    PluginInstanceCreateSchema,
} from "$lib/models/api/plugin_instance_schema";
import type { PluginInstance } from "$lib/models/plugin_instance";
import { Collection, create, handleError, list } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-instance:
 *   get:
 *     summary: List plugin instances
 *     description: Retrieves the authenticated user's configured plugin instances.
 *     tags:
 *       - Plugins
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
 *     responses:
 *       200:
 *         description: ListResult<PluginInstance>
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<PluginInstance>(event, Collection.plugin_instances);
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/plugin-instance:
 *   put:
 *     summary: Create plugin instance
 *     description: Creates a plugin instance for the authenticated user.
 *     tags:
 *       - Plugins
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - user
 *               - plugin_id
 *             properties:
 *               user:
 *                 type: string
 *                 description: User record ID
 *               plugin_id:
 *                 type: string
 *               enabled:
 *                 type: boolean
 *               auth:
 *                 type: object
 *                 additionalProperties:
 *                   type: string
 *               config:
 *                 type: object
 *                 additionalProperties: true
 *               state:
 *                 type: object
 *                 additionalProperties: true
 *               status:
 *                 type: string
 *                 enum: [configured, needs_auth, needs_reauth, syncing, rate_limited, unavailable, unsupported_protocol, error, disabled]
 *     responses:
 *       200:
 *         description: Plugin instance created
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<PluginInstance>(
            event,
            PluginInstanceCreateSchema,
            Collection.plugin_instances,
        );
        return json(r);
    } catch (e) {
        return handleError(e);
    }
}
