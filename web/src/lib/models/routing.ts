import * as M from "maplibre-gl";
import type Waypoint from "$lib/models/gpx/waypoint";


export interface RoutingOptions {
    autoRouting: boolean
    modeOfTransport: "pedestrian" | "bicycle" | "auto"
    category?: string
    subcategory?: string
    routingPluginId?: string
    routingInstanceId?: string
    routingElevationPluginId?: string
    routingElevationInstanceId?: string
    routingMode?: "segment" | "via"
    routingModeExplicit?: boolean
    engineMode?: "single" | "parallel"
    desiredVariants?: number
    nativeProfileKey?: string
    profileRevisions?: Record<string, string>
    preferences?: Record<string, unknown>
    nativeConfig?: Record<string, unknown>
}

export interface RoutingRouteResult {
    waypoints: Waypoint[]
    snappedAnchors?: RoutingPoint[]
    provenance?: RoutingSegmentProvenance
}

export interface RoutingRouteResponse {
    candidates: RoutingCandidate[]
    engineErrors?: RoutingEngineError[]
    warnings?: string[]
}

export interface RoutingEngineError {
    code: string
    message: string
    pluginId?: string
    instanceId?: string
    provider?: string
}

export interface RoutingCandidate {
    id: string
    profileKey?: string
    geometry?: RoutingGeometry
    elevation?: RoutingCandidateElevation
    summary: RoutingSummary
    segments: RoutingSegment[]
    snappedAnchors?: RoutingPoint[]
    suggestedAnchors?: RoutingPoint[]
    roundTrip?: RoutingRoundTripMetadata
    warnings?: string[]
    provider?: string
    pluginId?: string
    instanceId?: string
    compositionMode?: "segment_single_engine" | "segment_composed" | "via_route" | string
}

export interface RoutingRoundTripMetadata {
    requestId?: string
    targetDistance: number
    actualDistance: number
    direction?: number
    seed?: string
    attempts: number
    tolerance: number
}

export interface RoutingCandidateElevation {
    heights?: number[]
    status?: "included" | "complete" | "partial" | "empty" | "failed" | string
    source?: string
}

export interface RoutingPoint {
    lat: number
    lon: number
}

export interface RoutingGeometry {
    format: "encoded_polyline"
    precision: number
    coordinates: string
}

export interface RoutingSummary {
    distance: number
    duration: number
    elevationGain?: number
    elevationLoss?: number
}

export interface RoutingSegment {
    fromAnchor: number
    toAnchor: number
    geometry: RoutingGeometry
    distance: number
    duration: number
    provenance?: RoutingSegmentProvenance
}

export interface RoutingSegmentProvenance {
    source?: "round_trip" | string
    routeTopology?: "closed_loop"
    roundTripRequestId?: string
    roundTripTargetMeters?: number
    roundTripActualMeters?: number
    roundTripDirection?: number
    roundTripSeed?: string
    syntheticFromAnchor?: boolean
    syntheticToAnchor?: boolean
    category?: string
    subcategory?: string
    routingMode?: "segment" | "via"
    preferences?: Record<string, unknown>
    requestedPreferences?: Record<string, unknown>
    pluginId?: string
    instanceId?: string
    provider?: string
    profileId?: string
    profileKey?: string
    profileKind?: string
    nativeConfig?: Record<string, unknown>
    requestedNativeConfig?: Record<string, unknown>
    profileRevision?: string
}

export interface RoutingSettings {
    primaryRoutePluginId?: string
    elevationPluginId?: string
    maneuverPluginId?: string
    defaultVariantCount?: number
    defaultRoutingMode?: "segment" | "via"
    defaultPreferences?: Record<string, unknown>
    exposedFeatures?: Record<string, boolean>
}

export type RoutingManeuverType =
    | "start"
    | "destination"
    | "continue"
    | "turn_left"
    | "turn_right"
    | "turn_slight_left"
    | "turn_slight_right"
    | "turn_sharp_left"
    | "turn_sharp_right"
    | "keep_left"
    | "keep_right"
    | "uturn_left"
    | "uturn_right"
    | "uturn"
    | "roundabout_enter"
    | "roundabout_exit"
    | "exit_left"
    | "exit_right"
    | "ramp_straight"
    | "ramp_left"
    | "ramp_right"
    | "merge"
    | "merge_left"
    | "merge_right"
    | "ferry"
    | "unknown"

export interface RoutingManeuverGeometry {
    format: "encoded_polyline"
    precision: 6
    coordinates: string
}

export interface RoutingInternalManeuver {
    type: RoutingManeuverType
    providerInstruction?: string
    distanceMeters: number
    durationSeconds?: number
    beginShapeIndex: number
    endShapeIndex: number
    bearingBefore?: number
    bearingAfter?: number
    roundaboutExit?: number
    streetNames?: string[]
    warnings?: string[]
}

export interface RoutingManeuver extends Omit<RoutingInternalManeuver, "providerInstruction"> {
    instruction: string
}

export interface RoutingInternalManeuverResponse {
    geometry: RoutingManeuverGeometry
    maneuvers: RoutingInternalManeuver[]
    warnings?: string[]
}

export interface RoutingManeuverResponse {
    language: string
    geometry: RoutingManeuverGeometry
    maneuvers: RoutingManeuver[]
    warnings?: string[]
}

export interface RoutingProfileMapping {
    id?: string
    scope: "builtin" | "admin" | "user"
    category: string
    subcategory?: string
    pluginId: string
    instanceId?: string
    nativeProfileKey?: string
    profileId?: string
    preferences?: Record<string, unknown>
    nativeConfig?: Record<string, unknown>
}

export interface RoutingProfile {
    id?: string
    scope: "builtin" | "admin" | "user"
    pluginId: string
    key: string
    name: string
    kind: "builtin" | "custom_file" | "generated" | "native_config"
    mode: "foot" | "bike" | "motor" | "mixed" | "other"
    source: "discovery" | "builtin" | "admin" | "user"
    contentBase64?: string
    contentType?: string
    metadata?: Record<string, unknown>
    nativeConfig?: Record<string, unknown>
    enabled: boolean
}

export interface RoutingEngine {
    pluginId: string
    instanceId: string
    name: string
    enabled: boolean
    roles?: string[]
    modes?: string[]
    metadata?: Record<string, unknown>
}

export interface RoutingControl {
    key: string
    label?: string
    labels?: Record<string, string>
    unit?: string
    type: string
    ui?: string
    valueType?: string
    min?: number
    max?: number
    step?: number
    default?: unknown
    current?: unknown
    support?: string
    comparable?: boolean
    target?: string
    path?: string[]
    options?: { value: string; label: string; labels?: Record<string, string> }[]
}

export interface RoutingEffectiveControls {
    category: string
    subcategory?: string
    mode?: string
    profileRevisions?: Record<string, string>
    profileUploadRequired?: Record<string, boolean>
    profilePreparationSupported?: Record<string, boolean>
    controls: RoutingControl[]
    nativeControlGroups?: RoutingNativeControls["groups"]
    hiddenControls?: { key: string; reason: string }[]
    warnings?: string[]
}

export interface RoutingNativeControls {
    pluginId: string
    instanceId?: string
    profileId?: string
    groups: { key: string; label: string; labels?: Record<string, string>; controls: RoutingControl[] }[]
    warnings?: string[]
}

interface RoutingElevationResponse {
    heights: number[];
    status?: "included" | "partial" | "empty";
}

interface RoutingAnchor {
    id: string,
    lat: number,
    lon: number,
    marker?: M.Marker
}

export { type RoutingAnchor, type RoutingElevationResponse };
