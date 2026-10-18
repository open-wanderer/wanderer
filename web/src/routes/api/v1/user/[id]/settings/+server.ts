import { RecordIdSchema } from '$lib/models/api/base_schema';
import type { Settings } from '$lib/models/settings';
import { Collection, handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/user/{id}/settings:
 *   get:
 *     summary: Get settings for a user
 *     tags:
 *       - Settings
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         description: User ID
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         required: false
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: User settings
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Settings'
 *       404:
 *         description: Settings not found for user
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const safeParams = RecordIdSchema.parse(event.params);
        const expand = event.url.searchParams.get('expand') ?? undefined;

        const r = await event.locals.pb
            .collection(Collection.settings)
            .getFirstListItem<Settings>(`user = "${safeParams.id}"`, { expand });

        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
