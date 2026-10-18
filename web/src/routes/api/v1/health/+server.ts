import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/health:
 *   get:
 *     summary: Health / reachability probe
 *     description: >
 *       Lightweight, unauthenticated endpoint clients use to determine whether
 *       the backend is reachable and its database is up (e.g. to choose online
 *       vs offline map rendering before starting a GPS recording). Pings
 *       PocketBase's own health check: returns 200 when both the SvelteKit
 *       proxy and PocketBase respond, 503 when PocketBase / the database is
 *       unreachable, and (client-side) a connection error or timeout means the
 *       proxy itself is offline. Clients should treat only a 200 as "online".
 *     tags:
 *       - Maps
 *     responses:
 *       200:
 *         description: Backend and database are reachable
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                   example: "ok"
 *       503:
 *         description: PocketBase / database is unreachable
 */
export async function GET(event: RequestEvent) {
    try {
        // requestKey: null disables the SDK's auto-cancellation so a rapid
        // second probe never aborts this one. This deliberately hits
        // PocketBase so a running proxy in front of a dead database still
        // reports as offline.
        await event.locals.pb.health.check({ requestKey: null });
        return json({ status: 'ok' });
    } catch {
        return json({ status: 'unavailable' }, { status: 503 });
    }
}
