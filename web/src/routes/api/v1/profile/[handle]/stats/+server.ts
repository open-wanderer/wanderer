import { RecordListOptionsSchema } from '$lib/models/api/base_schema';
import type { SummitLog } from '$lib/models/summit_log';
import { enrichSummitLogAssetPhotos, withRequiredExpand } from '$lib/server/profile_asset_util';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { Collection, handleError } from '$lib/util/api_util';
import { isURL } from '$lib/util/file_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError } from 'pocketbase';

/**
 * @swagger
 * /api/v1/profile/{handle}/stats:
 *   get:
 *     summary: Get user summit statistics
 *     description: Retrieves summit log statistics for a user, with federation support
 *     tags:
 *       - Profiles
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *     responses:
 *       200:
 *         description: SummitLog statistics
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }
    
    try {
        const { actor } = await getActorResponseForHandle(event, handle);

        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = RecordListOptionsSchema.parse(searchParams);

        if(safeSearchParams.filter?.length) {
            safeSearchParams.filter = safeSearchParams.filter + `&&author='${actor.id}'`
        }else {
            safeSearchParams.filter = `author='${actor.id}'`
        }

        let summitLogs: SummitLog[];
        if (actor.is_local) {
            const listOptions = withRequiredExpand(safeSearchParams, ["summit_log_assets_via_summit_log.asset"]);
            summitLogs = await event.locals.pb.collection(Collection.summit_logs)
                .getFullList<SummitLog>(listOptions.page, { ...listOptions })
            enrichSummitLogAssetPhotos(summitLogs);
        } else {
            const origin = new URL(actor.iri).origin
            const summitLogURL = `${origin}/api/v1/profile/${actor.preferred_username}/stats?` + event.url.searchParams
            const response = await event.fetch(summitLogURL, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            summitLogs = await response.json()

            enrichSummitLogAssetPhotos(summitLogs, origin);
            summitLogs.forEach(i => {
                if (i.gpx) {
                    i.gpx = normalizeRemoteFileURL(i.gpx, origin, "summit_logs", i.id ?? "")
                }

            })
        }


        return json(summitLogs)
    } catch (e) {
        return handleError(e)
    }
}

function normalizeRemoteFileURL(file: string, origin: string, collection: string, recordId: string): string {
    if (isURL(file)) {
        return file;
    }
    if (file.startsWith("/")) {
        return new URL(file, origin).toString();
    }
    return new URL(`/api/v1/files/${collection}/${recordId}/${file}`, origin).toString();
}
