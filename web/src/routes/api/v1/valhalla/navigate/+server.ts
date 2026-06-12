import { getValhallaBaseUrl } from '$lib/server/valhalla';
import { handleError } from '$lib/util/api_util';
import { decodePolyline } from '$lib/util/polyline_util';
import { NavigateRequestSchema } from '$lib/models/api/valhalla_navigate_schema';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import type { NavigateManeuver } from '$lib/models/api/valhalla_navigate_schema';

/**
 * @swagger
 * /api/v1/valhalla/navigate:
 *   post:
 *     summary: Get turn-by-turn navigation maneuvers
 *     description: Accepts a trail waypoint array and returns a structured maneuver list with decoded shape points for Flutter navigation
 *     tags:
 *       - Valhalla
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - waypoints
 *             properties:
 *               waypoints:
 *                 type: array
 *                 minItems: 2
 *                 items:
 *                   type: object
 *                   required: [lat, lon]
 *                   properties:
 *                     lat:
 *                       type: number
 *                       minimum: -90
 *                       maximum: 90
 *                     lon:
 *                       type: number
 *                       minimum: -180
 *                       maximum: 180
 *               costing:
 *                 type: string
 *                 enum: [pedestrian, bicycle]
 *                 default: pedestrian
 *     responses:
 *       200:
 *         description: Maneuver list and decoded shape points
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 maneuvers:
 *                   type: array
 *                   items:
 *                     type: object
 *                     properties:
 *                       instruction:
 *                         type: string
 *                       length:
 *                         type: number
 *                       begin_shape_index:
 *                         type: integer
 *                       bearing:
 *                         type: number
 *                 shape:
 *                   type: array
 *                   items:
 *                     type: array
 *                     items:
 *                       type: number
 *       400:
 *         description: Invalid request body or missing VALHALLA_URL
 *       401:
 *         description: Unauthorized
 *       502:
 *         description: Valhalla upstream error
 */
export async function POST(event: RequestEvent) {
  if (!event.locals.user) {
    return error(401, "Unauthorized");
  }

  try {
    const baseUrl = getValhallaBaseUrl();
    if (!baseUrl) {
      return json({ message: "VALHALLA_URL not set" }, { status: 400 });
    }

    const body = NavigateRequestSchema.parse(await event.request.json());

    const valhallaReq = {
      locations: body.waypoints,
      costing: body.costing,
      directions_type: "instructions",
    };

    const res = await event.fetch(baseUrl + "/route", {
      method: "POST",
      body: JSON.stringify(valhallaReq),
    });

    if (!res.ok) {
      const detail = await res.text();
      return json({ message: "valhalla_error", detail }, { status: 502 });
    }

    const trip = (await res.json()).trip;

    const shape: [number, number][] = [];
    const maneuvers: NavigateManeuver[] = [];

    for (const leg of trip.legs) {
      const offset = shape.length;

      for (const [lng, lat] of decodePolyline(leg.shape)) {
        shape.push([lat, lng]);
      }

      for (const m of leg.maneuvers ?? []) {
        maneuvers.push({
          instruction: m.instruction,
          length: m.length,
          begin_shape_index: offset + m.begin_shape_index,
          bearing: m.bearing_after ?? m.bearing_before ?? 0,
        });
      }
    }

    return json({ maneuvers, shape });
  } catch (e: any) {
    return handleError(e);
  }
}
