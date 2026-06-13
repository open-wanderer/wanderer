import { describe, it, expect, vi, beforeEach } from "vitest";
import { NavigateRequestSchema } from "$lib/models/api/valhalla_navigate_schema";
import { encodePolyline } from "$lib/util/polyline_util";

// Mock getValhallaBaseUrl so the endpoint doesn't need a real VALHALLA_URL env var in tests
vi.mock("$lib/server/valhalla", () => ({
  getValhallaBaseUrl: () => "http://valhalla.test",
}));

import { POST } from "./+server";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/** Build a minimal fake RequestEvent for the navigate endpoint. */
function makeEvent(opts: {
  user?: unknown;
  body?: unknown;
  fetchResponse?: { ok: boolean; status: number; json?: () => Promise<unknown>; text?: () => Promise<string> };
}) {
  const { user = { id: "user123" }, body = {}, fetchResponse } = opts;

  const defaultFetch = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ trip: { legs: [] } }),
    text: async () => "",
  });

  const fetchMock = fetchResponse
    ? vi.fn().mockResolvedValue({
        ok: fetchResponse.ok,
        status: fetchResponse.status,
        json: fetchResponse.json ?? (async () => ({})),
        text: fetchResponse.text ?? (async () => ""),
      })
    : defaultFetch;

  return {
    locals: { user },
    request: {
      json: async () => body,
    },
    fetch: fetchMock,
  } as unknown as import("@sveltejs/kit").RequestEvent;
}

// ---------------------------------------------------------------------------
// Encode a set of coordinates as a precision-6 polyline.
// encodePolyline expects [lat, lng] pairs (first value is latitude, which is
// what decodePolyline accumulates first and then outputs as [lng, lat]).
// So to get decode output [lng=8.1, lat=47.5] we must encode [lat=47.5, lng=8.1].
// The endpoint flips decode's [lng, lat] → [lat, lng] for the response shape.
//
// Known coordinates used across tests (all distinct so lat != lon):
//   Point A: lat=47.5, lon=8.1  → encode as [lat=47.5, lon=8.1]
//   Point B: lat=47.6, lon=8.2  → encode as [lat=47.6, lon=8.2]
//   Point C: lat=47.7, lon=8.3  → encode as [lat=47.7, lon=8.3]
// ---------------------------------------------------------------------------
const coordsLatLng: number[][] = [
  [47.5, 8.1],
  [47.6, 8.2],
  [47.7, 8.3],
];
const encodedShape = encodePolyline(coordsLatLng, 6);

// Two-leg test coordinates
const leg1CoordsLatLng: number[][] = [
  [47.5, 8.1],
  [47.6, 8.2],
  [47.7, 8.3],
];
const leg2CoordsLatLng: number[][] = [
  [47.8, 8.4],
  [47.9, 8.5],
];
const encodedLeg1 = encodePolyline(leg1CoordsLatLng, 6);
const encodedLeg2 = encodePolyline(leg2CoordsLatLng, 6);

// ---------------------------------------------------------------------------
// Shared test trip data
// ---------------------------------------------------------------------------
const happyPathTrip = {
  legs: [
    {
      shape: encodedShape,
      maneuvers: [
        {
          instruction: "Head north on Trail Path",
          length: 1.5,
          begin_shape_index: 0,
          bearing_after: 15,
          bearing_before: 0,
        },
        {
          instruction: "Turn right onto Forest Road",
          length: 0.8,
          begin_shape_index: 1,
          bearing_after: 90,
          bearing_before: 15,
        },
      ],
    },
  ],
};

// ---------------------------------------------------------------------------
// Test 1 & 2: Happy path + coordinate flip
// ---------------------------------------------------------------------------
describe("POST /api/v1/valhalla/navigate", () => {
  it("Test 1: returns 200 with maneuver list and shape for valid authenticated request", async () => {
    const event = makeEvent({
      body: { shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
      fetchResponse: {
        ok: true,
        status: 200,
        json: async () => ({ trip: happyPathTrip }),
      },
    });

    const response = await POST(event);
    const body = await response.json();

    expect(response.status).toBe(200);

    // maneuvers array has correct length and shape
    expect(body.maneuvers).toHaveLength(2);

    const [m1, m2] = body.maneuvers;

    // Each maneuver has the required fields
    expect(typeof m1.instruction).toBe("string");
    expect(m1.instruction.length).toBeGreaterThan(0);
    expect(typeof m1.length).toBe("number");
    expect(Number.isInteger(m1.begin_shape_index)).toBe(true);
    expect(typeof m1.bearing).toBe("number");

    expect(typeof m2.instruction).toBe("string");
    expect(m2.instruction.length).toBeGreaterThan(0);
    expect(typeof m2.length).toBe("number");
    expect(Number.isInteger(m2.begin_shape_index)).toBe(true);
    expect(typeof m2.bearing).toBe("number");

    // shape is an array of [lat, lon] pairs
    expect(Array.isArray(body.shape)).toBe(true);
    expect(body.shape.length).toBeGreaterThanOrEqual(3);

    // First shape point: decode gives [lng=8.1, lat=47.5], after flip → [lat=47.5, lon=8.1]
    expect(body.shape[0][0]).toBeCloseTo(47.5, 4);
    expect(body.shape[0][1]).toBeCloseTo(8.1, 4);
  });

  it("Test 2: coordinate flip — shape[0] is [lat, lon] not [lon, lat]", async () => {
    // lat=47.5 and lon=8.1 are distinct so order is observable
    const event = makeEvent({
      body: { shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
      fetchResponse: {
        ok: true,
        status: 200,
        json: async () => ({ trip: happyPathTrip }),
      },
    });

    const response = await POST(event);
    const body = await response.json();

    // After flip: first element must be latitude (~47.5), second longitude (~8.1)
    // If wrongly NOT flipped: first would be ~8.1 (longitude), second ~47.5 (latitude)
    expect(body.shape[0][0]).toBeCloseTo(47.5, 4); // lat
    expect(body.shape[0][1]).toBeCloseTo(8.1, 4);  // lon
    // Confirm it's not the un-flipped order (lat != lon so we can distinguish)
    expect(body.shape[0][0]).not.toBeCloseTo(8.1, 1);
  });

  // ---------------------------------------------------------------------------
  // Test 3: Schema validation
  // ---------------------------------------------------------------------------
  it("Test 3: NavigateRequestSchema validation", () => {
    // Rejects empty shape array
    expect(() => NavigateRequestSchema.parse({ shape: [] })).toThrow();

    // Rejects single-point array (min 2)
    expect(() =>
      NavigateRequestSchema.parse({ shape: [{ lat: 47.5, lon: 8.1 }] })
    ).toThrow();

    // Rejects lat=200 / lon=200 (out of bounds)
    expect(() =>
      NavigateRequestSchema.parse({
        shape: [
          { lat: 200, lon: 200 },
          { lat: 47.6, lon: 8.2 },
        ],
      })
    ).toThrow();

    // Defaults costing to "pedestrian" when omitted
    const result = NavigateRequestSchema.parse({
      shape: [
        { lat: 47.5, lon: 8.1 },
        { lat: 47.7, lon: 8.3 },
      ],
    });
    expect(result.costing).toBe("pedestrian");

    // Accepts costing: "bicycle"
    const resultBicycle = NavigateRequestSchema.parse({
      shape: [
        { lat: 47.5, lon: 8.1 },
        { lat: 47.7, lon: 8.3 },
      ],
      costing: "bicycle",
    });
    expect(resultBicycle.costing).toBe("bicycle");
  });

  // ---------------------------------------------------------------------------
  // Test 4: Auth gate (D-09)
  // ---------------------------------------------------------------------------
  it("Test 4: returns 401 and does not call Valhalla when user is not authenticated", async () => {
    const fetchMock = vi.fn();
    const event = {
      locals: { user: null },
      request: {
        json: async () => ({
          shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }],
        }),
      },
      fetch: fetchMock,
    } as unknown as import("@sveltejs/kit").RequestEvent;

    let threw = false;
    let thrownStatus: number | undefined;
    try {
      const response = await POST(event);
      thrownStatus = response.status;
    } catch (e: any) {
      threw = true;
      // SvelteKit error() throws an HttpError — check its status
      thrownStatus = e?.status ?? e?.statusCode;
    }

    // Either threw a 401 HttpError or returned 401 response
    expect(thrownStatus).toBe(401);
    // Valhalla was never called
    expect(fetchMock).not.toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Test 5: Invalid input → 400 (Success Criterion 3)
  // ---------------------------------------------------------------------------
  it("Test 5: invalid input returns 400 with message: invalid_params — never a 500", async () => {
    // Empty shape fails Zod min(2) → handleError → 400
    const event = makeEvent({
      body: { shape: [] },
    });

    const response = await POST(event);
    const body = await response.json();

    expect(response.status).toBe(400);
    expect(body.message).toBe("invalid_params");
  });

  // ---------------------------------------------------------------------------
  // Test 6: Valhalla upstream failure → 502 (Pitfall 5)
  // ---------------------------------------------------------------------------
  it("Test 6: upstream Valhalla failure returns 502 with message: valhalla_error — never a 500", async () => {
    const event = makeEvent({
      body: { shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
      fetchResponse: {
        ok: false,
        status: 400,
        text: async () => "no path found between points",
      },
    });

    const response = await POST(event);
    const body = await response.json();

    expect(response.status).toBe(502);
    expect(body.message).toBe("valhalla_error");
  });

  // ---------------------------------------------------------------------------
  // Test 7: Multi-leg shape offset (Pitfall 1 / D-03)
  // ---------------------------------------------------------------------------
  it("Test 7: multi-leg shape offset — second leg maneuver begin_shape_index is offset by leg-1 point count", async () => {
    // Leg 1 has 3 points (encodedLeg1), leg 2 has 2 points (encodedLeg2)
    const multiLegTrip = {
      legs: [
        {
          shape: encodedLeg1,
          maneuvers: [
            {
              instruction: "Start heading north",
              length: 1.2,
              begin_shape_index: 0, // leg-local index 0
              bearing_after: 10,
              bearing_before: 0,
            },
          ],
        },
        {
          shape: encodedLeg2,
          maneuvers: [
            {
              instruction: "Continue on trail",
              length: 0.5,
              begin_shape_index: 1, // leg-local index 1
              bearing_after: 20,
              bearing_before: 10,
            },
          ],
        },
      ],
    };

    const event = makeEvent({
      body: { shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }, { lat: 47.9, lon: 8.5 }] },
      fetchResponse: {
        ok: true,
        status: 200,
        json: async () => ({ trip: multiLegTrip }),
      },
    });

    const response = await POST(event);
    const body = await response.json();

    expect(response.status).toBe(200);

    // Total shape = leg1 (3 points) + leg2 (2 points) = 5
    expect(body.shape).toHaveLength(3 + 2);

    // Leg-1 maneuver: begin_shape_index = 0 + 0 = 0
    expect(body.maneuvers[0].begin_shape_index).toBe(0);

    // Leg-2 maneuver: local index 1, offset by leg1's 3 points → 3 + 1 = 4
    expect(body.maneuvers[1].begin_shape_index).toBe(3 + 1);
  });

  // ---------------------------------------------------------------------------
  // Test 8: Missing maneuvers array (guard against crash)
  // ---------------------------------------------------------------------------
  it("Test 8: leg with no maneuvers field does not crash — returns 200 with empty maneuver list", async () => {
    const tripWithNoManeuvers = {
      legs: [
        {
          shape: encodedShape,
          // no maneuvers field — tests the `?? []` guard
        },
      ],
    };

    const event = makeEvent({
      body: { shape: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
      fetchResponse: {
        ok: true,
        status: 200,
        json: async () => ({ trip: tripWithNoManeuvers }),
      },
    });

    const response = await POST(event);
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(body.maneuvers).toHaveLength(0);
    expect(Array.isArray(body.shape)).toBe(true);
  });
});
