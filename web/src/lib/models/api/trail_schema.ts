import { z, ZodType } from "zod";
import type { RoutingSegmentProvenance } from "../routing";
import type { Trail } from "../trail";

const RoutingLoopTopologyShape = {
    routeTopology: z.literal("closed_loop"),
    roundTripRequestId: z.string().optional(),
    roundTripTargetMeters: z.number().optional(),
    roundTripActualMeters: z.number().optional(),
    roundTripDirection: z.number().int().min(0).max(359).optional(),
    roundTripSeed: z.string().optional(),
    syntheticFromAnchor: z.boolean().optional(),
    syntheticToAnchor: z.boolean().optional(),
};

const RoutingEngineProvenanceSchema = z.object({
    source: z.string().optional(),
    routeTopology: z.literal("closed_loop").optional(),
    roundTripRequestId: z.string().optional(),
    roundTripTargetMeters: z.number().optional(),
    roundTripActualMeters: z.number().optional(),
    roundTripDirection: z.number().int().min(0).max(359).optional(),
    roundTripSeed: z.string().optional(),
    syntheticFromAnchor: z.boolean().optional(),
    syntheticToAnchor: z.boolean().optional(),
    category: z.string().optional(),
    subcategory: z.string().optional(),
    routingMode: z.enum(["segment", "via"]).optional(),
    preferences: z.record(z.unknown()).optional(),
    requestedPreferences: z.record(z.unknown()).optional(),
    pluginId: z.string().min(1),
    instanceId: z.string().optional(),
    provider: z.string().optional(),
    profileId: z.string().optional(),
    profileKey: z.string().optional(),
    profileKind: z.string().optional(),
    nativeConfig: z.record(z.unknown()).optional(),
    requestedNativeConfig: z.record(z.unknown()).optional(),
    profileRevision: z.string().optional(),
}).strict().refine(
    (value) =>
        value.routingMode !== undefined ||
        (value.source === "round_trip" && value.routeTopology === "closed_loop"),
    { message: "routingMode is required for ordinary engine provenance" },
);

const RoutingTopologyOnlyProvenanceSchema = z.object(
    RoutingLoopTopologyShape,
).strict();

const RoutingSegmentProvenanceSchema: ZodType<RoutingSegmentProvenance> = z.union([
    RoutingEngineProvenanceSchema,
    RoutingTopologyOnlyProvenanceSchema,
]);

const TrailCreateSchema = z.object({
    id: z.string().length(15).optional(),
    name: z.string().min(1, "required"),
    description: z.string().optional(),
    location: z.string().optional(),
    date: z.string().optional().refine((val) => !val || !isNaN(Date.parse(val)), "invalid-date"),
    public: z.boolean(),
    completed: z.boolean(),
    difficulty: z.enum(["easy", "moderate", "difficult"]).optional(),
    lat: z.number().min(-90).max(90).optional(),
    lon: z.number().min(-180).max(180).optional(),
    distance: z.number({ coerce: true }).nonnegative().optional(),
    elevation_gain: z.number({ coerce: true }).nonnegative().optional(),
    elevation_loss: z.number({ coerce: true }).nonnegative().optional(),
    duration: z.number({ coerce: true }).nonnegative().optional(),
    thumbnail: z.number().int().nonnegative().optional(),
    like_count: z.number().int().min(0).optional().default(0),
    category: z.string().length(15).optional().or(z.literal('')),
    subcategory: z.string().length(15).optional().or(z.literal('')),
    tags: z.array(z.string()).default([]),
    gpx: z.string().optional(),
    routing_provenance: z.array(RoutingSegmentProvenanceSchema.nullable()).optional(),
    author: z.string().length(15),

}) satisfies ZodType<Partial<Trail>>

const TrailUpdateSchema = z.object({
    name: z.string(),
    description: z.string().optional(),
    location: z.string().optional(),
    date: z.string().optional().refine((val) => !val || !isNaN(Date.parse(val)), "invalid-date"),
    public: z.boolean().optional(),
    completed: z.boolean().optional(),
    difficulty: z.enum(["easy", "moderate", "difficult"]).optional(),
    lat: z.number().min(-90).max(90).optional(),
    lon: z.number().min(-180).max(180).optional(),
    distance: z.number({ coerce: true }).nonnegative().optional(),
    elevation_gain: z.number({ coerce: true }).nonnegative().optional(),
    elevation_loss: z.number({ coerce: true }).nonnegative().optional(),
    duration: z.number({ coerce: true }).nonnegative().optional(),
    thumbnail: z.number().int().nonnegative().optional(),
    like_count: z.number().int().min(0).optional(),
    category: z.string().optional(),
    subcategory: z.string().optional(),
    tags: z.array(z.string()).optional(),
    gpx: z.string().optional(),
    routing_provenance: z.array(RoutingSegmentProvenanceSchema.nullable()).optional(),
}) satisfies ZodType<Partial<Trail>>

const TrailRecommendSchema = z.object({
    size: z.number({ coerce: true }).optional()
})

export { TrailCreateSchema, TrailUpdateSchema, TrailRecommendSchema };
