import { describe, expect, it } from "vitest";
import type { RoutingEngine } from "$lib/models/routing";
import {
    ROUND_TRIP_COMPASS_DIRECTIONS,
    anchorsForClosedLoopEdit,
    candidateForClosedLoop,
    closedLoopEditAction,
    routingEngineSupportsRoundTrip,
    isPersistedClosedLoop,
    roundTripPreferencesForEngine,
    roundTripProfileForEngine,
    roundTripLoopProvenanceForSegments,
    selectRoundTripRoutingEngine,
    shouldShowRoundTripControls,
} from "./routing_round_trip_util";

describe("round-trip compass directions", () => {
    it("maps the eight compass points clockwise to bearings", () => {
        expect(
            ROUND_TRIP_COMPASS_DIRECTIONS.map((direction) => direction.bearing),
        ).toEqual([0, 45, 90, 135, 180, 225, 270, 315]);
        expect(
            ROUND_TRIP_COMPASS_DIRECTIONS.map((direction) => direction.labelKey),
        ).toEqual([
            "routing-round-trip-direction-north",
            "routing-round-trip-direction-north-east",
            "routing-round-trip-direction-east",
            "routing-round-trip-direction-south-east",
            "routing-round-trip-direction-south",
            "routing-round-trip-direction-south-west",
            "routing-round-trip-direction-west",
            "routing-round-trip-direction-north-west",
        ]);
    });
});

function engine(pluginId: string, supportsRoundTrip: boolean, enabled = true): RoutingEngine {
    return {
        pluginId,
        instanceId: `${pluginId}-instance`,
        name: pluginId,
        enabled,
        metadata: { routing: { supportsRoundTrip } },
    };
}

describe("round-trip engine discovery", () => {
    it("is unavailable when no enabled engine declares support", () => {
        const engines = [engine("a", false), engine("b", true, false)];
        expect(engines.some(routingEngineSupportsRoundTrip)).toBe(false);
        expect(selectRoundTripRoutingEngine(engines, "a")).toBeUndefined();
    });

    it("prefers the selected capable instance and otherwise falls back", () => {
        const engines = [engine("fallback", true), engine("selected", true)];
        expect(selectRoundTripRoutingEngine(engines, "selected")?.pluginId).toBe("selected");
        expect(selectRoundTripRoutingEngine(engines, "missing")?.pluginId).toBe("fallback");
    });
});

describe("round-trip provider profile", () => {
    it("removes provider-specific profile data when falling back to another engine", () => {
        expect(
            roundTripProfileForEngine(
                { key: "pedestrian", kind: "builtin", nativeConfig: { shortest: true } },
                true,
            ),
        ).toEqual({ key: "", kind: undefined, nativeConfig: {} });
        expect(
            roundTripPreferencesForEngine(
                { speedPreference: 4.5, maxHikingDifficulty: 5 },
                true,
            ),
        ).toEqual({});
    });

    it("keeps the profile for the selected engine", () => {
        const profile = { key: "trekking", kind: "builtin", nativeConfig: { custom: true } };
        expect(roundTripProfileForEngine(profile, false)).toBe(profile);
        const preferences = { hillPreference: 0.5 };
        expect(roundTripPreferencesForEngine(preferences, false)).toBe(preferences);
    });
});

describe("persisted round-trip state", () => {
    const provenance = {
        source: "round_trip",
        routeTopology: "closed_loop" as const,
        pluginId: "brouter",
    };

    it("requires round-trip provenance on every persisted segment", () => {
        expect(isPersistedClosedLoop([provenance, provenance], 2)).toBe(true);
        expect(isPersistedClosedLoop([provenance, null], 2)).toBe(false);
        expect(isPersistedClosedLoop([provenance], 2)).toBe(false);
    });

    it("does not classify geometry-only closed tracks", () => {
        expect(isPersistedClosedLoop([], 3)).toBe(false);
        expect(
            isPersistedClosedLoop(
                [{ routingMode: "segment", pluginId: "brouter" }],
                1,
            ),
        ).toBe(false);
    });

    it("shows generation only for a bare start anchor or an existing generated round trip", () => {
        const bareStart = {
            capabilityAvailable: true,
            anchorCount: 1,
            hasRoute: false,
            provenance: [],
            segmentCount: 0,
        };
        expect(shouldShowRoundTripControls(bareStart)).toBe(true);
        expect(
            shouldShowRoundTripControls({
                ...bareStart,
                capabilityAvailable: false,
            }),
        ).toBe(false);
        expect(
            shouldShowRoundTripControls({
                ...bareStart,
                anchorCount: 2,
            }),
        ).toBe(false);
        expect(
            shouldShowRoundTripControls({
                ...bareStart,
                hasRoute: true,
            }),
        ).toBe(false);

        const generatedLoop = {
            capabilityAvailable: true,
            anchorCount: 3,
            hasRoute: true,
            provenance: [provenance, provenance, provenance],
            segmentCount: 3,
        };
        expect(shouldShowRoundTripControls(generatedLoop)).toBe(true);
        expect(
            shouldShowRoundTripControls({
                ...generatedLoop,
                provenance: [
                    { routeTopology: "closed_loop", roundTripRequestId: "rt_edited" },
                    { routeTopology: "closed_loop", roundTripRequestId: "rt_edited" },
                    { routeTopology: "closed_loop", roundTripRequestId: "rt_edited" },
                ],
            }),
        ).toBe(true);
        expect(
            shouldShowRoundTripControls({
                ...generatedLoop,
                provenance: [
                    { routeTopology: "closed_loop" },
                    { routeTopology: "closed_loop" },
                    { routeTopology: "closed_loop" },
                ],
            }),
        ).toBe(false);
    });

    it("preserves round-trip identity while refreshing segment routing data", () => {
        const previous = {
            ...provenance,
            roundTripRequestId: "rt_original",
            roundTripTargetMeters: 20_000,
            roundTripActualMeters: 21_000,
            roundTripDirection: 270,
            roundTripSeed: "seed",
        };
        const result = roundTripLoopProvenanceForSegments(
            previous,
            [
                { routingMode: "segment", pluginId: "new-engine" },
                { routingMode: "segment", pluginId: "new-engine" },
                { routingMode: "segment", pluginId: "new-engine" },
            ],
            19_800,
        );
        expect(result).toHaveLength(3);
        expect(result[0]).toMatchObject({
            pluginId: "new-engine",
            roundTripRequestId: "rt_original",
            roundTripActualMeters: 19_800,
            syntheticFromAnchor: false,
            syntheticToAnchor: true,
        });
        expect(result[2]).toMatchObject({
            routeTopology: "closed_loop",
            routingMode: "segment",
            roundTripDirection: 270,
            syntheticFromAnchor: true,
            syntheticToAnchor: false,
        });
    });

    it("keeps topology but drops fabricated engine provenance for manual or reversed loops", () => {
        const values = roundTripLoopProvenanceForSegments(
            {
                ...provenance,
                provider: "BRouter",
                profileKey: "trekking",
                profileRevision: "revision",
                preferences: { hillPreference: 0.5 },
                nativeConfig: { profile: "custom" },
                roundTripRequestId: "rt_manual",
            },
            [undefined, undefined, undefined],
            10_000,
        );
        expect(values.map((entry) => [
            entry.syntheticFromAnchor,
            entry.syntheticToAnchor,
        ])).toEqual([
            [false, true],
            [true, true],
            [true, false],
        ]);
        for (const entry of values) {
            expect(entry).toMatchObject({
                routeTopology: "closed_loop",
                roundTripRequestId: "rt_manual",
                roundTripActualMeters: 10_000,
            });
            expect(entry.source).toBeUndefined();
            expect(entry.routingMode).toBeUndefined();
            expect(entry.pluginId).toBeUndefined();
            expect(entry.provider).toBeUndefined();
            expect(entry.profileKey).toBeUndefined();
            expect(entry.profileRevision).toBeUndefined();
            expect(entry.preferences).toBeUndefined();
            expect(entry.nativeConfig).toBeUndefined();
        }
    });

    it("copies a recalculated candidate and keeps its actual segment provenance", () => {
        const candidate = {
            id: "variant",
            summary: { distance: 9_900, duration: 3_000 },
            segments: [
                {
                    fromAnchor: 0,
                    toAnchor: 1,
                    geometry: {
                        format: "encoded_polyline" as const,
                        precision: 6,
                        coordinates: "geometry-a",
                    },
                    distance: 5_000,
                    duration: 1_500,
                    provenance: {
                        routingMode: "segment" as const,
                        pluginId: "new-engine",
                        profileKey: "new-profile",
                    },
                },
                {
                    fromAnchor: 1,
                    toAnchor: 2,
                    geometry: {
                        format: "encoded_polyline" as const,
                        precision: 6,
                        coordinates: "geometry-b",
                    },
                    distance: 4_900,
                    duration: 1_500,
                    provenance: {
                        routingMode: "segment" as const,
                        pluginId: "new-engine",
                        profileKey: "new-profile",
                    },
                },
            ],
        };
        const before = structuredClone(candidate);
        const result = candidateForClosedLoop(candidate, {
            ...provenance,
            roundTripRequestId: "rt_copy",
        });

        expect(candidate).toEqual(before);
        expect(result).not.toBe(candidate);
        expect(result.segments[0]).not.toBe(candidate.segments[0]);
        expect(result.segments[0].provenance).toMatchObject({
            routeTopology: "closed_loop",
            routingMode: "segment",
            pluginId: "new-engine",
            profileKey: "new-profile",
            roundTripRequestId: "rt_copy",
        });
        expect(result.segments[1].toAnchor).toBe(0);
    });

    it("selects every closed-loop edit transition explicitly", () => {
        expect(closedLoopEditAction({
            closedLoop: false,
            anchorCount: 3,
            hasTopology: false,
            autoRouting: false,
        })).toBe("ordinary");
        expect(closedLoopEditAction({
            closedLoop: true,
            anchorCount: 1,
            hasTopology: true,
            autoRouting: false,
        })).toBe("clear");
        expect(closedLoopEditAction({
            closedLoop: true,
            anchorCount: 3,
            hasTopology: false,
            autoRouting: true,
        })).toBe("invalid");
        expect(closedLoopEditAction({
            closedLoop: true,
            anchorCount: 3,
            hasTopology: true,
            autoRouting: false,
        })).toBe("manual");
        expect(closedLoopEditAction({
            closedLoop: true,
            anchorCount: 3,
            hasTopology: true,
            autoRouting: true,
        })).toBe("routed");
    });

    it("does not duplicate the start anchor while recovering invalid topology", () => {
        const anchors = [{ id: "start" }, { id: "middle" }, { id: "end" }];

        expect(anchorsForClosedLoopEdit(anchors, "invalid")).toBe(anchors);
        expect(anchorsForClosedLoopEdit(anchors, "ordinary")).toBe(anchors);
        expect(anchorsForClosedLoopEdit(anchors, "routed")).toEqual([
            anchors[0],
            anchors[1],
            anchors[2],
            anchors[0],
        ]);
    });
});
