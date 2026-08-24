import { describe, expect, it } from "vitest";
import type { RoutingCandidate } from "$lib/models/routing";
import { routingCandidateEngineLabel } from "./routing_variant_util";

function candidate(
    provider: string | undefined,
    pluginId: string | undefined,
    segments: [string | undefined, string | undefined][],
): RoutingCandidate {
    return {
        id: "candidate",
        provider,
        pluginId,
        summary: { distance: 1_000, duration: 600 },
        segments: segments.map(([segmentProvider, segmentPluginId], index) => ({
            fromAnchor: index,
            toAnchor: index + 1,
            geometry: { format: "encoded_polyline", precision: 6, coordinates: "" },
            distance: 500,
            duration: 300,
            provenance: { provider: segmentProvider, pluginId: segmentPluginId },
        })),
    };
}

describe("routing candidate engine label", () => {
    it("deduplicates provider and plugin aliases for the same engine", () => {
        expect(
            routingCandidateEngineLabel(
                candidate("Valhalla", "valhalla", [
                    [undefined, "valhalla"],
                    ["Valhalla", undefined],
                ]),
            ),
        ).toBe("Valhalla");
    });

    it("keeps distinct engines in first-seen order", () => {
        expect(
            routingCandidateEngineLabel(
                candidate(undefined, undefined, [
                    ["Valhalla", "valhalla"],
                    ["BRouter", "brouter"],
                ]),
            ),
        ).toBe("Valhalla · BRouter");
    });
});
