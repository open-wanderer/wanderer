import { SummitLogCreateSchema } from '$lib/models/api/summit_log_schema';
import type { SummitLog } from '$lib/models/summit_log';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { type ListResult } from "pocketbase";

/**
 * @swagger
 * /api/v1/summit-log:
 *   get:
 *     summary: List summit logs
 *     description: Retrieves a paginated list of summit logs with deduplication of federated data
 *     tags:
 *       - Summit Logs
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
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *       - in: query
 *         name: handle
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: ListResult<SummitLog> with local/remote items deduplicated
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.url.searchParams.has("handle")) {
            const summitLogs = await list<SummitLog>(event, Collection.summit_logs);
            removeTimeFromDates(summitLogs.items)
            return json(summitLogs)
        } else {
            const { actor, error } = await event.locals.pb.send(`/activitypub/actor?resource=acct:${event.url.searchParams.get("handle")}`, { method: "GET", fetch: event.fetch, });
            event.url.searchParams.delete("handle")
            const localSummitLogs = await list<SummitLog>(event, Collection.summit_logs);
            if (actor.isLocal) {
                removeTimeFromDates(localSummitLogs.items)

                return json(localSummitLogs)
            }

            const deduplicationMap: Record<string, SummitLog> = {}

            localSummitLogs.items.forEach(l => {
                if (l.iri) {
                    const id = l.iri.substring(l.iri.length - 15)
                    deduplicationMap[id] = l
                } else if (l.id) {
                    deduplicationMap[l.id] = l
                }

                l.date = l.date.substring(0, 10);

            })
            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/summit-log`

            const response = await event.fetch(url + '?' + event.url.searchParams, { method: 'GET' })
            if (!response.ok) {
                const errorResponse = await response.json()
                console.error(errorResponse)

            }
            const remoteSummitLogs: ListResult<SummitLog> = await response.json()

            remoteSummitLogs.items = remoteSummitLogs.items.filter(l => {
                const iriId = l.iri?.substring(l.iri.length - 15) ?? ""
                if (deduplicationMap[l.id!] != undefined) {
                    deduplicationMap[l.id!] = {...l, author: deduplicationMap[l.id!].author}
                    return false
                } else if (deduplicationMap[iriId] != undefined) {
                    deduplicationMap[iriId] = {...l, author: deduplicationMap[iriId].author}
                    return false
                }
                return true
            })

            remoteSummitLogs.items.forEach(l => {
                if (l.gpx) {
                    l.gpx = `${origin}/api/v1/files/summit_logs/${l.id}/${l.gpx}`
                }
                l.photos = l.photos.map(p =>
                    `${origin}/api/v1/files/summit_logs/${l.id}/${p}`
                )

                if (l.expand?.author) {
                    l.expand.author.isLocal = false
                }

            })

            const allSummitLogItems = <ListResult<SummitLog>>{
                items: localSummitLogs.items.concat(remoteSummitLogs.items),
                page: localSummitLogs.page,
                perPage: localSummitLogs.perPage,
                totalItems: localSummitLogs.items.length + remoteSummitLogs.items.length,
                totalPages: Math.ceil((localSummitLogs.items.length + remoteSummitLogs.items.length) / localSummitLogs.perPage)
            }

            allSummitLogItems.items = allSummitLogItems.items.sort((a, b) => {
                return new Date(a.created ?? 0).getTime() - new Date(b.created ?? 0).getTime()
            })

            return json(allSummitLogItems)
        }
    } catch (e) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/summit-log:
 *   put:
 *     summary: Create summit log
 *     tags:
 *       - Summit Logs
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/SummitLogInput'
 *     responses:
 *       201:
 *         description: Summit log created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<SummitLog>(event, SummitLogCreateSchema, Collection.summit_logs)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}


function removeTimeFromDates(logs: SummitLog[]) {
    logs.forEach(l => l.date = l.date.substring(0, 10));

}