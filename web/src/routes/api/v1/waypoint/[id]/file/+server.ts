import type { RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/waypoint/{id}/file:
 *   post:
 *     summary: Reject removed waypoint photo uploads
 *     deprecated: true
 *     description: Use PUT /api/v1/assets with a waypoint target. This endpoint always returns 400 waypoint_photos_form_field_removed without writing a record.
 *     tags: [Waypoints]
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       400:
 *         description: waypoint_photos_form_field_removed
 */
export async function POST(event: RequestEvent) {
 if (!event.locals.user) return Response.json({message:"Unauthorized"}, {status:401});
 return Response.json({message:"waypoint_photos_form_field_removed"}, {status:400});
}
