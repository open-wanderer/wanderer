import { env } from "$env/dynamic/public";
import type { Actor } from "$lib/models/activitypub/actor";
import { defaultTrailSearchAttributes, type TrailSearchResult } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import type { Hits, MultiSearchParams, MultiSearchResponse, MultiSearchResult, SearchParams, SearchResponse } from "meilisearch";
import type { ListResult } from "pocketbase";
import { version } from "$app/environment";

export type LocationSearchResult = {
    name: string;
    description: string;
    lat: number;
    lon: number;
    category: string;
    type: string;
}

export type ListSearchResult = {
    id: string;
    author: string;
    author_name: string;
    author_avatar: string;
    avatar?: string;
    created: number;
    description: string;
    name: string;
    elevation_gain: number;
    elevation_loss: number;
    distance: number,
    duration: number,
    domain?: string,
    public: boolean;
    trails: number
    shares?: string[];
    iri?: string;
}

type NominatimResponse = {
    type: string
    licence: string
    features: Feature[]
}

type Feature = {
    type: string
    properties: Properties
    bbox: number[]
    geometry: Geometry
}

type Address = {
    amenity: string
    road: string
    neighbourhood: string
    suburb: string
    city_district?: string
    city?: string
    town?: string
    hamlet?: string
    village?: string;
    state: string
    "ISO3166-2-lvl4": string
    postcode: string
    country: string
    country_code: string
}
type Properties = {
    place_id: number
    osm_type: string
    osm_id: number
    place_rank: number
    category: string
    type: string
    importance: number
    addresstype: string
    name: string
    display_name: string
    address: Address
}

type Geometry = {
    type: string
    coordinates: number[]
}


export async function searchTrails(q: string, options: SearchParams): Promise<Hits<TrailSearchResult>> {
    const r = await fetch("/api/v1/search/trails", {
        method: "POST",
        body: JSON.stringify({
            q,
            attributesToRetrieve: defaultTrailSearchAttributes,
            options
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: SearchResponse<TrailSearchResult> = await r.json();

    return response.hits
}

export async function searchLocations(q: string, limit?: number): Promise<Hits<LocationSearchResult>> {
    const nominatimURL = env.PUBLIC_NOMINATIM_URL ?? "https://nominatim.openstreetmap.org"
    const r = await fetch(`${nominatimURL}/search?q=${q}&format=geojson&addressdetails=1${limit ? '&limit=' + limit : ''}`, {
        method: "GET",
        headers: new Headers({
            "User-Agent": "wanderer/" + version
        })
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    const response: NominatimResponse = await r.json();
    return response.features.map(f => ({
        category: f.properties.category,
        type: f.properties.type == "administrative" ? f.properties.addresstype : f.properties.type,
        description: getLocationDescription(f.properties.address),
        name: f.properties.name.length ? f.properties.name : f.properties.display_name,
        lat: f.geometry.coordinates[1],
        lon: f.geometry.coordinates[0],
    }))
}

export async function searchLocationReverse(lat: number, lon: number) {
    const nominatimURL = env.PUBLIC_NOMINATIM_URL ?? "https://nominatim.openstreetmap.org"
    const r = await fetch(`${nominatimURL}/reverse?lat=${lat}&lon=${lon}&format=geojson&addressdetails=1`, {
        method: "GET",
        headers: new Headers({
            "User-Agent": "wanderer/" + version
        })
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    const response: NominatimResponse = await r.json();

    if (response.features?.at(0)?.properties.address) {
        return getLocationDescription(response.features[0].properties.address)
    }
    return ""
}

function getLocationDescription(address: Address) {
    let description = ""

    if (address.country) {
        description += address.country;
    }
    if (address.state) {
        description = `${address.state}, ` + description
    }
    if (address.city) {
        description = `${address.city}, ` + description
    } else if (address.town) {
        description = `${address.town}, ` + description
    } else if (address.hamlet) {
        description = `${address.hamlet}, ` + description
    } else if (address.village) {
        description = `${address.village}, ` + description
    }
    return description;
}

export async function searchMulti(options: MultiSearchParams): Promise<MultiSearchResult<any>[]> {

    const locationQuery = options.queries.find(q => q.indexUid === "locations");
    const locationQueryIndex = locationQuery ? options.queries.indexOf(locationQuery) : -1
    if (locationQueryIndex >= 0) {
        options.queries.splice(locationQueryIndex, 1)
    }
    const r = await fetch("/api/v1/search/multi", {
        method: "POST",
        body: JSON.stringify(options),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: MultiSearchResponse<any> = await r.json();


    if (locationQuery && locationQuery.q !== undefined && locationQuery.q !== null) {
        const locationsResults = await searchLocations(locationQuery.q, locationQuery.limit)
        response.results.splice(locationQueryIndex,
            0,
            { hits: locationsResults, indexUid: "locations", query: locationQuery.q, processingTimeMs: 0 }
        )
    }


    return response.results
}

export async function searchActors(q: string, includeSelf: boolean = true): Promise<Actor[]> {
    try {
        const r = await fetch(`/api/v1/search/actor?q=${q}&includeSelf=${includeSelf}`,)

        if (!r.ok) {
            return []
        }
        const response: ListResult<Actor> = await r.json()

        return response.items
    } catch (e) {
        console.log(e);

        return []
    }
}
