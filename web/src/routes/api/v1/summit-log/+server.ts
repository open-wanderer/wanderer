import { SummitLogCreateSchema } from '$lib/models/api/summit_log_schema';
import type { SummitLog } from '$lib/models/summit_log';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/summit-log:
 *   get:
 *     summary: List summit logs
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
 *     responses:
 *       200:
 *         description: List of summit logs
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const summitLogs = await list<SummitLog>(event, Collection.summit_logs);
        removeTimeFromDates(summitLogs.items)
        return json(summitLogs)

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