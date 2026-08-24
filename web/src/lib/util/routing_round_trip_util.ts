import type {
    RoutingCandidate,
    RoutingEngine,
    RoutingSegmentProvenance,
} from "$lib/models/routing";

export const ROUND_TRIP_COMPASS_DIRECTIONS = [
    { bearing: 0, labelKey: "routing-round-trip-direction-north" },
    { bearing: 45, labelKey: "routing-round-trip-direction-north-east" },
    { bearing: 90, labelKey: "routing-round-trip-direction-east" },
    { bearing: 135, labelKey: "routing-round-trip-direction-south-east" },
    { bearing: 180, labelKey: "routing-round-trip-direction-south" },
    { bearing: 225, labelKey: "routing-round-trip-direction-south-west" },
    { bearing: 270, labelKey: "routing-round-trip-direction-west" },
    { bearing: 315, labelKey: "routing-round-trip-direction-north-west" },
] as const;

export function routingEngineSupportsRoundTrip(engine: RoutingEngine) {
    const routing = engine.metadata?.routing;
    return (
        engine.enabled &&
        typeof routing === "object" &&
        routing !== null &&
        (routing as Record<string, unknown>).supportsRoundTrip === true
    );
}

export function selectRoundTripRoutingEngine(
    engines: RoutingEngine[],
    preferredPluginId?: string,
    preferredInstanceId?: string,
) {
    const preferred = engines.find(
        (engine) =>
            engine.pluginId === preferredPluginId &&
            (!preferredInstanceId || engine.instanceId === preferredInstanceId) &&
            routingEngineSupportsRoundTrip(engine),
    );
    return preferred ?? engines.find(routingEngineSupportsRoundTrip);
}

export function roundTripProfileForEngine<T extends {
    key?: string;
    kind?: string;
    nativeConfig?: Record<string, unknown>;
}>(profile: T, providerChanged: boolean): T {
    if (!providerChanged) return profile;
    return {
        ...profile,
        key: "",
        kind: undefined,
        nativeConfig: {},
    };
}

export function roundTripPreferencesForEngine<T extends Record<string, unknown> | undefined>(
    preferences: T,
    providerChanged: boolean,
): T | Record<string, never> {
    return providerChanged ? {} : preferences;
}

export function isPersistedClosedLoop(
    provenance: (RoutingSegmentProvenance | null)[],
    segmentCount: number,
) {
    return (
        segmentCount > 0 &&
        provenance.length === segmentCount &&
        provenance.every((entry) => entry?.routeTopology === "closed_loop")
    );
}

export function shouldShowRoundTripControls(input: {
    capabilityAvailable: boolean;
    anchorCount: number;
    hasRoute: boolean;
    provenance: (RoutingSegmentProvenance | null)[];
    segmentCount: number;
}) {
    if (!input.capabilityAvailable || input.anchorCount < 1) return false;
    if (input.anchorCount === 1 && !input.hasRoute) return true;
    if (!isPersistedClosedLoop(input.provenance, input.segmentCount)) return false;
    return input.provenance.some(
        (entry) => entry?.source === "round_trip" || Boolean(entry?.roundTripRequestId),
    );
}

export function roundTripLoopProvenanceForSegments(
    previous: RoutingSegmentProvenance,
    segments: (RoutingSegmentProvenance | null | undefined)[],
    actualDistance: number,
): RoutingSegmentProvenance[] {
    const count = segments.length;
    return segments.map((segment, index) => ({
        ...segment,
        routeTopology: "closed_loop",
        roundTripRequestId: previous.roundTripRequestId,
        roundTripTargetMeters: previous.roundTripTargetMeters,
        roundTripActualMeters: actualDistance,
        roundTripDirection: previous.roundTripDirection,
        roundTripSeed: previous.roundTripSeed,
        syntheticFromAnchor: index !== 0,
        syntheticToAnchor: (index + 1) % count !== 0,
    }));
}

export function candidateForClosedLoop(
    candidate: RoutingCandidate,
    previous: RoutingSegmentProvenance,
): RoutingCandidate {
    const count = candidate.segments.length;
    if (count === 0) return candidate;
    const provenance = roundTripLoopProvenanceForSegments(
        previous,
        candidate.segments.map((segment) => segment.provenance),
        candidate.summary.distance,
    );
    return {
        ...candidate,
        compositionMode: "round_trip",
        segments: candidate.segments.map((segment, index) => ({
            ...segment,
            toAnchor: index === count - 1 ? 0 : segment.toAnchor,
            provenance: provenance[index],
        })),
    };
}

export type ClosedLoopEditAction =
    | "ordinary"
    | "clear"
    | "invalid"
    | "manual"
    | "routed";

export function closedLoopEditAction(input: {
    closedLoop: boolean;
    anchorCount: number;
    hasTopology: boolean;
    autoRouting: boolean;
}): ClosedLoopEditAction {
    if (input.anchorCount < 2) return "clear";
    if (!input.closedLoop) return "ordinary";
    if (!input.hasTopology) return "invalid";
    return input.autoRouting ? "routed" : "manual";
}

export function anchorsForClosedLoopEdit<T>(
    anchors: T[],
    action: ClosedLoopEditAction,
): T[] {
    return action === "routed" && anchors.length > 0
        ? [...anchors, anchors[0]]
        : anchors;
}
