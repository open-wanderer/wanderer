import { TagCreateSchema } from '$lib/models/api/tag_schema';
import type { Tag } from '$lib/models/tag';
import { handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/tag/lookup:
 *   post:
 *     summary: Find the first tag with an exact name
 *     description: Returns at most one existing tag, choosing the smallest ID when names are duplicated. The name is case sensitive and must be normalized. This operation does not create or merge tags.
 *     tags:
 *       - Tags
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TagInput'
 *     responses:
 *       200:
 *         description: Exact match, or an empty items array
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               required: [items]
 *               properties:
 *                 items:
 *                   type: array
 *                   maxItems: 1
 *                   items:
 *                     $ref: '#/components/schemas/Tag'
 *       400:
 *         description: Invalid tag name
 *       403:
 *         description: Tag lookup is not permitted
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const data = TagCreateSchema.pick({ name: true }).parse(await event.request.json());
        const result = await event.locals.pb.send<{ items: Tag[] }>('/tags/lookup', {
            method: 'POST',
            body: data,
            fetch: event.fetch,
            requestKey: null,
        });
        return json(result);
    } catch (e) {
        return handleError(e);
    }
}
