import { SettingsCreateSchema } from '$lib/models/api/settings_schema';
import type { Settings } from '$lib/models/settings';
import { Collection, create } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/settings:
 *   put:
 *     summary: Create settings
 *     tags:
 *       - Settings
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/SettingsInput'
 *     responses:
 *       201:
 *         description: Settings created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Settings'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<Settings>(event, SettingsCreateSchema, Collection.settings)

        return json(r);
    } catch (e: any) {
        throw error(e.status, e)
    }
}
