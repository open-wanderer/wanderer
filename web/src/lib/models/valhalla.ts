import * as M from "maplibre-gl";


interface ValhallaCostingOptions {
    shortest?: boolean
}

export interface ValhallaPedestrianCostingOptions extends ValhallaCostingOptions {
    use_ferry?: number
    use_living_streets?: number
    use_tracks?: number
    service_penalty?: number
    service_factor?: number
    use_hills?: number
    walking_speed?: number
    walkway_factor?: number
    sidewalk_factor?: number
    alley_factor?: number
    driveway_factor?: number
    step_penalty?: number
    max_hiking_difficulty?: number
    use_lit?: number
    transit_start_end_max_distance?: number
    transit_transfer_max_distance?: number
}

export interface ValhallaBicycleCostingOptions extends ValhallaCostingOptions {
    maneuver_penalty?: number
    country_crossing_penalty?: number
    country_crossing_cost?: number
    use_ferry?: number
    use_living_streets?: number
    service_penalty?: number
    service_factor?: number
    bicycle_type?: "Road" | "Hybrid" | "City" | "Cross" | "Mountain"
    cycling_speed?: number
    use_roads?: number
    use_hills?: number
    avoid_bad_surfaces?: number
    gate_penalty?: number
    gate_cost?: number
}

export interface ValhallaAutoCostingOptions extends ValhallaCostingOptions {
    maneuver_penalty?: number
    country_crossing_penalty?: number
    country_crossing_cost?: number
    width?: number
    height?: number
    use_highways?: number
    use_tolls?: number
    use_ferry?: number
    ferry_cost?: number
    use_living_streets?: number
    use_tracks?: number
    private_access_penalty?: number
    ignore_closures?: boolean
    ignore_restrictions?: boolean
    ignore_access?: boolean
    closure_factor?: number
    service_penalty?: number
    service_factor?: number
    exclude_unpaved?: number
    exclude_cash_only_tolls?: boolean
    top_speed?: number
    fixed_speed?: number
    toll_booth_penalty?: number
    toll_booth_cost?: number
    gate_penalty?: number
    gate_cost?: number
    include_hov2?: boolean
    include_hov3?: boolean
    include_hot?: boolean
    disable_hierarchy_pruning?: boolean
}

export interface RoutingOptions {
    autoRouting: boolean
    modeOfTransport: "pedestrian" | "bicycle" | "auto"
    pedestrianOptions?: ValhallaPedestrianCostingOptions
    bicycleOptions?: ValhallaBicycleCostingOptions
    autoOptions?: ValhallaAutoCostingOptions
}

interface ValhallaRouteResponse {
    trip: Trip
}

export interface Trip {
    locations: Location[]
    legs: Leg[]
    summary: Summary
    status_message: string
    status: number
    units: string
    language: string
}

export interface Location {
    type: string
    lat: number
    lon: number
    original_index: number
}

export interface ValhallaManeuver {
    instruction: string
    length: number
    begin_shape_index: number
    bearing_before?: number
    bearing_after?: number
}

export interface Leg {
    summary: Summary
    shape: string
    maneuvers?: ValhallaManeuver[]
}

export interface Summary {
    has_time_restrictions: boolean
    has_toll: boolean
    has_highway: boolean
    has_ferry: boolean
    min_lat: number
    min_lon: number
    max_lat: number
    max_lon: number
    time: number
    length: number
    cost: number
}

interface ValhallaHeightResponse {
    height: number[];
}

interface ValhallaAnchor {
    id: string,
    lat: number,
    lon: number,
    marker?: M.Marker
}

export { type ValhallaAnchor, type ValhallaHeightResponse, type ValhallaRouteResponse };
