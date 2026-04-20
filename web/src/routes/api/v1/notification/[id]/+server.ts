import { NotificationUpdateSchema } from "$lib/models/api/notification_schema";
import type { Notification } from "$lib/models/notification";
import { Collection, handleError, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";


/**
 * @swagger
 * /api/v1/notification/{id}:
 *   post:
 *     summary: Update notification
 *     description: Updates a notification by ID (typically to mark as read)
 *     tags:
 *       - Notifications
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
 *         description: Notification
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Notification>(event, NotificationUpdateSchema, Collection.notifications)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}