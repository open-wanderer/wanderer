import { describe, expect, it } from "vitest";
import { TrailUpdateSchema } from "./trail_schema";

describe("trail routing provenance schema", () => {
    it("persists closed-loop topology independently from engine provenance", () => {
        const parsed = TrailUpdateSchema.parse({
            name: "Loop",
            routing_provenance: [
                {
                    routeTopology: "closed_loop",
                    roundTripRequestId: "rt_test",
                    roundTripTargetMeters: 20_000,
                    roundTripActualMeters: 19_800,
                    roundTripDirection: 270,
                    roundTripSeed: "seed",
                    syntheticFromAnchor: false,
                    syntheticToAnchor: true,
                },
            ],
        });

        expect(parsed.routing_provenance?.[0]).toEqual({
            routeTopology: "closed_loop",
            roundTripRequestId: "rt_test",
            roundTripTargetMeters: 20_000,
            roundTripActualMeters: 19_800,
            roundTripDirection: 270,
            roundTripSeed: "seed",
            syntheticFromAnchor: false,
            syntheticToAnchor: true,
        });
        expect(parsed.routing_provenance?.[0]?.pluginId).toBeUndefined();
        expect(parsed.routing_provenance?.[0]?.routingMode).toBeUndefined();
    });

    it("retains valid provider provenance and rejects invalid round-trip bearings", () => {
        const parsed = TrailUpdateSchema.parse({
            name: "Loop",
            routing_provenance: [
                {
                    source: "round_trip",
                    routeTopology: "closed_loop",
                    pluginId: "brouter",
                    provider: "BRouter",
                    profileKey: "trekking",
                },
            ],
        });
        expect(parsed.routing_provenance?.[0]).toMatchObject({
            source: "round_trip",
            routeTopology: "closed_loop",
            pluginId: "brouter",
        });

        expect(() =>
            TrailUpdateSchema.parse({
                name: "Invalid loop",
                routing_provenance: [
                    { routeTopology: "closed_loop", roundTripDirection: 360 },
                ],
            }),
        ).toThrow();
        expect(() =>
            TrailUpdateSchema.parse({
                name: "Missing engine",
                routing_provenance: [
                    { routingMode: "segment", category: "hiking" },
                ],
            }),
        ).toThrow();
        expect(() =>
            TrailUpdateSchema.parse({
                name: "Fabricated engine",
                routing_provenance: [
                    { routeTopology: "closed_loop", provider: "BRouter" },
                ],
            }),
        ).toThrow();
    });
});
