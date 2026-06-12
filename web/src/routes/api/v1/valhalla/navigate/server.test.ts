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
      body: { waypoints: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
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
      body: { waypoints: [{ lat: 47.5, lon: 8.1 }, { lat: 47.7, lon: 8.3 }] },
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
    // Rejects empty waypoint array
    expect(() => NavigateRequestSchema.parse({ waypoints: [] })).toThrow();

    // Rejects single-waypoint array (min 2)
    expect(() =>
      NavigateRequestSchema.parse({ waypoints: [{ lat: 47.5, lon: 8.1 }] })
    ).toThrow();

    // Rejects lat=200 / lon=200 (out of bounds)
    expect(() =>
      NavigateRequestSchema.parse({
        waypoints: [
          { lat: 200, lon: 200 },
          { lat: 47.6, lon: 8.2 },
        ],
      })
    ).toThrow();

    // Defaults costing to "pedestrian" when omitted
    const result = NavigateRequestSchema.parse({
      waypoints: [
        { lat: 47.5, lon: 8.1 },
        { lat: 47.7, lon: 8.3 },
      ],
    });
    expect(result.costing).toBe("pedestrian");

    // Accepts costing: "bicycle"
    const resultBicycle = NavigateRequestSchema.parse({
      waypoints: [
        { lat: 47.5, lon: 8.1 },
        { lat: 47.7, lon: 8.3 },
      ],
      costing: "bicycle",
    });
    expect(resultBicycle.costing).toBe("bicycle");
  });
});
