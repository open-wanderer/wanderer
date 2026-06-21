import {
    PluginInstanceUpdateSchema,
} from "$lib/models/api/plugin_instance_schema";
import type { PluginInstance } from "$lib/models/plugin_instance";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/plugin-instance/{id}:
 *   get:
 *     summary: Get plugin instance
 *     description: Retrieves a plugin instance by ID.
 *     tags:
 *       - Plugins
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           description: Plugin instance record ID
 *     responses:
 *       200:
 *         description: Plugin instance details
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<PluginInstance>(event, Collection.plugin_instances);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/plugin-instance/{id}:
 *   post:
 *     summary: Update plugin instance
 *     description: Updates plugin instance settings, auth, state, or status.
 *     tags:
 *       - Plugins
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           description: Plugin instance record ID
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             properties:
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
 *         description: Plugin instance updated
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<PluginInstance>(
            event,
            PluginInstanceUpdateSchema,
            Collection.plugin_instances,
        );
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/plugin-instance/{id}:
 *   delete:
 *     summary: Delete plugin instance
 *     description: Deletes a configured plugin instance.
 *     tags:
 *       - Plugins
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *           description: Plugin instance record ID
 *     responses:
 *       200:
 *         description: Success
 *       401:
 *         description: Unauthorized
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.plugin_instances);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
