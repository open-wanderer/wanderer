import { RecordListOptionsSchema } from '$lib/models/api/base_schema';
import type { StatisticActivity } from '$lib/models/statistic_activity';
import type { Subcategory } from '$lib/models/subcategory';
import type { SummitLog } from '$lib/models/summit_log';
import type { Trail } from '$lib/models/trail';
import {
    buildCompletedTrailFilter,
    buildSummitLogStatisticsFilter,
    mergeStatisticActivities,
    ProfileStatisticsFilterSchema,
    summitLogToStatisticActivity,
    type ProfileStatisticsFilter,
} from '$lib/server/profile_statistics';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { Collection, handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { ClientResponseError } from 'pocketbase';

/**
 * @swagger
 * /api/v1/profile/{handle}/stats:
 *   get:
 *     summary: Get user activity statistics
 *     description: Retrieves summit logs and completed trails without summit logs for a user, with federation support
 *     tags:
 *       - Profiles
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: startDate
 *         schema:
 *           type: string
 *           format: date
 *       - in: query
 *         name: endDate
 *         schema:
 *           type: string
 *           format: date
 *       - in: query
 *         name: category
 *         schema:
 *           type: string
 *         description: Comma-separated category IDs
 *       - in: query
 *         name: subcategory
 *         schema:
 *           type: string
 *         description: Comma-separated subcategory filter values
 *     responses:
 *       200:
 *         description: Activity statistics
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/StatisticActivity'
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
        if (!actor.id) {
            return error(404, { message: "Actor not found" });
        }

        const searchParams = Object.fromEntries(event.url.searchParams);
        const safeSearchParams = RecordListOptionsSchema.parse(searchParams);
        const statisticsFilter: ProfileStatisticsFilter =
            ProfileStatisticsFilterSchema.parse({
                startDate: event.url.searchParams.get('startDate') || undefined,
                endDate: event.url.searchParams.get('endDate') || undefined,
                category: (event.url.searchParams.get('category') ?? '')
                    .split(',')
                    .filter(Boolean),
                subcategory: (event.url.searchParams.get('subcategory') ?? '')
                    .split(',')
                    .filter(Boolean),
            });

        let activities: StatisticActivity[];
        if (actor.is_local) {
            const availableSubcategories = await event.locals.pb
                .collection(Collection.subcategories)
                .getFullList<Subcategory>();
            const explicitSummitLogFilter = buildSummitLogStatisticsFilter(
                statisticsFilter,
                availableSubcategories,
            );
            safeSearchParams.filter = [
                `author='${actor.id}'`,
                explicitSummitLogFilter,
            ]
                .filter(Boolean)
                .join('&&');
            const summitLogs = await event.locals.pb.collection(Collection.summit_logs)
                .getFullList<SummitLog>({ ...safeSearchParams });

            const completedTrails = await event.locals.pb
                .collection(Collection.trails)
                .getFullList<Trail>({
                    filter: buildCompletedTrailFilter(
                        actor.id,
                        statisticsFilter,
                        availableSubcategories,
                    ),
                    expand: 'category,subcategory,subcategory.category,author,summit_logs_via_trail',
                });

            activities = mergeStatisticActivities(summitLogs, completedTrails);
        } else {
            const remoteSearchParams = new URLSearchParams(
                event.url.searchParams,
            );
            remoteSearchParams.delete('filter');
            const origin = new URL(actor.iri).origin
            const summitLogURL = `${origin}/api/v1/profile/${actor.preferred_username}/stats?` + remoteSearchParams
            const response = await event.fetch(summitLogURL, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                throw new ClientResponseError({ status: response.status, response: errorResponse });
            }
            const remoteActivities: Array<StatisticActivity | SummitLog> = await response.json()
            activities = remoteActivities.map((activity) =>
                'source' in activity
                    ? activity
                    : summitLogToStatisticActivity(activity),
            );

            activities.forEach(i => {
                const collection = i.collectionId ||
                    (i.source === 'completed_trail' ? Collection.trails : Collection.summit_logs);
                i.collectionId = collection;
                i.collectionName = collection;
                i.photos = (i.photos ?? []).map(p =>
                    `${origin}/api/v1/files/${collection}/${i.id}/${p}`
                )
                if (i.gpx) {
                    i.gpx = `${origin}/api/v1/files/${collection}/${i.id}/${i.gpx}`
                }

            })
        }

        return json(activities)
    } catch (e) {
        return handleError(e)
    }
}
