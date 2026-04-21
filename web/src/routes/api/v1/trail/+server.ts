import { TrailCreateSchema } from '$lib/models/api/trail_schema';
import type { Trail } from '$lib/models/trail';
import { Collection, create, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/trail:
 *   get:
 *     summary: List trails
 *     description: Retrieves a paginated list of trails with optional filtering and sorting
 *     tags:
 *       - Trails
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
 *         description: ListResult<Trail>
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<Trail>(event, Collection.trails);

        for (const t of r.items) {
            if (!t.author || !event.locals.pb.authStore.record) {
                continue;
            }
            if (!t.expand) {
                t.expand = {} as any
            }

            t.expand?.waypoints_via_trail?.sort((a, b) => (a.distance_from_start ?? 0) - (b.distance_from_start ?? 0))
        }
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/trail:
 *   put:
 *     summary: Create trail
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/TrailCreateInput'
 *     responses:
 *       201:
 *         description: Trail created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {        
        const r = await create<Trail>(event, TrailCreateSchema, Collection.trails)
        enrichRecord(r);
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}

function enrichRecord(r: Trail) {
    r.date = r.date?.substring(0, 10) ?? "";
    for (const log of r.expand?.summit_logs_via_trail ?? []) {
        log.date = log.date.substring(0, 10);
    }
}