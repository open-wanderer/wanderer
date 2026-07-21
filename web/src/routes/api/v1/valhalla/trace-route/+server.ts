import { handleError } from '$lib/util/api_util';
import { decodePolyline } from '$lib/util/polyline_util';
import { TraceRouteRequestSchema } from '$lib/models/api/valhalla_trace_route_schema';
import { error, json, type RequestEvent } from '@sveltejs/kit';
import { getValhallaTraceRouteUrl } from '$lib/server/valhalla';

/**
 * @swagger
 * /api/v1/valhalla/trace-route:
 *   post:
 *     summary: Snap a recorded shape to the road network
 *     description: Accepts a recorded (breadcrumb) shape array and returns a decoded, road-snapped shape via Valhalla's trace_route map-matching action. No maneuvers are returned — this is track cleanup, not turn-by-turn guidance.
 *     tags:
 *       - Valhalla
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - shape
 *             properties:
 *               shape:
 *                 type: array
 *                 minItems: 2
 *                 maxItems: 500
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
 *         description: Decoded, road-snapped shape points
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 shape:
 *                   type: array
 *                   items:
 *                     type: object
 *                     properties:
 *                       lat:
 *                         type: number
 *                       lon:
 *                         type: number
 *       400:
 *         description: Invalid request body or missing VALHALLA_TRACE_ROUTE_URL
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
    const traceRouteUrl = getValhallaTraceRouteUrl();
    if (!traceRouteUrl) {
      return json({ message: "VALHALLA_TRACE_ROUTE_URL not set" }, { status: 400 });
    }

    const body = TraceRouteRequestSchema.parse(await event.request.json());

    const valhallaReq = {
      shape: body.shape,
      costing: body.costing,
      shape_match: "map_snap",
    };

    const res = await event.fetch(traceRouteUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(valhallaReq),
    });

    if (!res.ok) {
      const detail = await res.text();
      return json({ message: "valhalla_error", detail }, { status: 502 });
    }

    const data = await res.json();
    const trip = data?.trip;
    if (!trip?.legs) {
      return json({ message: "valhalla_error", detail: "unexpected response shape" }, { status: 502 });
    }

    const shape: { lat: number; lon: number }[] = [];

    for (const leg of trip.legs) {
      for (const [lng, lat] of decodePolyline(leg.shape)) {
        shape.push({ lat, lon: lng });
      }
    }

    return json({ shape });
  } catch (e: any) {
    return handleError(e);
  }
}
