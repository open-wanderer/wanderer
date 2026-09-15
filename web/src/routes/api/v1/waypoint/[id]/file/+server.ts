import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/waypoint/{id}/file:
 *   post:
 *     summary: Upload waypoint file
 *     deprecated: true
 *     description: >
 *       This endpoint no longer accepts photo uploads and returns 400 `waypoint_photos_form_field_removed`.
 *       Upload and attach photos through `PUT /api/v1/assets` with the `waypoint` target field.
 *     tags:
 *       - Waypoints
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       400:
 *         description: Photo uploads through this endpoint have been removed
 *       401:
 *         description: Unauthorized
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return json({ message: "Unauthorized" }, { status: 401 });
    }
    return json({ message: "waypoint_photos_form_field_removed" }, { status: 400 });
}
