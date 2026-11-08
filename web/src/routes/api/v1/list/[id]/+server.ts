import { completeTrailCardPreviews, withoutListPhotoExpands } from "#lib/server/trail_card_previews.ts";
import { assetResponseJson as json } from "#lib/server/asset_response_json.ts";
import { ListUpdateSchema } from "#lib/models/api/list_schema.ts";
import type { List } from "#lib/models/list.ts";
import { completeTrailAssetExpands } from "#lib/server/trail_asset_pagination.ts";
import { enrichListTrailResponse, withListTrailAssetExpandParams } from "#lib/server/trail_response_util.ts";
import { Collection, handleError, remove, update } from "#lib/util/api_util.ts";
import { type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/list/{id}:
 *   get:
 *     summary: Get list
 *     description: Retrieves a list by ID
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: query
 *         name: photoPreview
 *         description: Set true for trail cards with up to three photos including the selected thumbnail, instead of full photo link expands.
 *         schema:
 *           type: boolean
 *           default: false
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
 *         description: List
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 *   post:
 *     summary: Update list
 *     description: Updates a list by ID
 *     tags:
 *       - Lists
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
 *             type: object
 *     responses:
 *       200:
 *         description: List
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 *   delete:
 *     summary: Delete list
 *     description: Deletes a list by ID
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Success
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const { url, params } = event;

    try {
        const preview = url.searchParams.get("photoPreview") === "true";
        const requested = url.searchParams.get("expand")?.split(",").some((path) => path.trim() === "trails" || path.trim().startsWith("trails."));
        const searchParams = requested ? (preview ? withoutListPhotoExpands(url.searchParams) : withListTrailAssetExpandParams(url.searchParams)) : url.searchParams;
        let list: List = await event.locals.pb.send(`/remote/list/${params.id}?` + searchParams, {
            method: "GET",
            fetch: event.fetch,
        })

        if (requested && preview) await completeTrailCardPreviews(event, list.expand?.trails ?? []);
        else if (requested) {
            await completeTrailAssetExpands(event, list.expand?.trails ?? []);
            enrichRecord(list, url.searchParams.get("share") ?? undefined);
        }
        return json(list)
    } catch (e: any) {
        return handleError(e);
    }
}

function enrichRecord(list: List, share?: string) {
    enrichListTrailResponse(list, { share });
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<List>(event, ListUpdateSchema, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
