import { SettingsCreateSchema } from '$lib/models/api/settings_schema';
import type { Settings } from "$lib/models/settings";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";


/**
 * @swagger
 * /api/v1/settings/{id}:
 *   get:
 *     summary: Get settings
 *     tags:
 *       - Settings
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
 *         description: Settings details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Settings'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<Settings>(event, Collection.settings)
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/settings/{id}:
 *   post:
 *     summary: Update settings
 *     tags:
 *       - Settings
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
 *             $ref: '#/components/schemas/SettingsInput'
 *     responses:
 *       200:
 *         description: Settings updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Settings'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Settings>(event, SettingsCreateSchema, Collection.settings)
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/settings/{id}:
 *   delete:
 *     summary: Delete settings
 *     tags:
 *       - Settings
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Settings deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.settings)
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}