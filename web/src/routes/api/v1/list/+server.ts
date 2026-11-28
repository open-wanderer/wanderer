import { completeTrailCardPreviews, withoutListPhotoExpands } from "#lib/server/trail_card_previews.ts";
import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { ListCreateSchema } from '#lib/models/api/list_schema.ts';
import { completeTrailAssetExpands } from "#lib/server/trail_asset_pagination.ts";
import { enrichListTrailResponse, withListTrailAssetExpandParams } from "#lib/server/trail_response_util.ts";
import type { List } from '#lib/models/list.ts';
import { Collection, create, handleError, list } from '#lib/util/api_util.ts';
import { type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/list:
 *   get:
 *     summary: List all lists
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: query
 *         name: photoPreview
 *         description: Set true for trail cards with up to three photos including the selected thumbnail, instead of full photo link expands.
 *         schema:
 *           type: boolean
 *           default: false
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
 *         description: List of lists
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
        const preview = event.url.searchParams.get("photoPreview") === "true";
        const requested = event.url.searchParams.get("expand")?.split(",").some((path) => path.trim() === "trails" || path.trim().startsWith("trails."));
        const queryEvent = { ...event, url: new URL(event.url) };
        if (requested) queryEvent.url.search = (preview ? withoutListPhotoExpands(event.url.searchParams) : withListTrailAssetExpandParams(event.url.searchParams)).toString();
        const r = await list<List>(queryEvent, Collection.lists);
        if (requested) {
            if (preview) await completeTrailCardPreviews(event, r.items.flatMap(record => record.expand?.trails ?? []));
            else {
                await completeTrailAssetExpands(event, r.items.flatMap((record) => record.expand?.trails ?? []));
                for (const record of r.items) enrichListTrailResponse(record, { share: event.url.searchParams.get("share") ?? undefined });
            }
        }
        return json(r)
    } catch (e) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/list:
 *   put:
 *     summary: Create list
 *     tags:
 *       - Lists
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/ListInput'
 *     responses:
 *       201:
 *         description: List created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/List'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<List>(event, ListCreateSchema, Collection.lists)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}