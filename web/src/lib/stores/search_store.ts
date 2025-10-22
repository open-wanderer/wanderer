import { env } from "$env/dynamic/public";
import type { Actor } from "$lib/models/activitypub/actor";
import { defaultTrailSearchAttributes, type TrailSearchResult } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import type { Hits, MultiSearchParams, MultiSearchResponse, MultiSearchResult, SearchParams, SearchResponse } from "meilisearch";
import type { ListResult } from "pocketbase";
import { version } from "$app/environment";

const NOMINATIM_RATE_LIMIT_MS = 1000;
const NOMINATIM_MAX_RETRIES = 2;
let lastNominatimCall = 0;

const LOCATION_SEARCH_CACHE_TTL_MS = 5 * 60 * 1000;
const REVERSE_LOCATION_CACHE_TTL_MS = 5 * 60 * 1000;

const nominatimURL = env.PUBLIC_NOMINATIM_URL ?? "https://nominatim.openstreetmap.org";
const nominatimUserAgent = "wanderer/" + version;
const nominatimNeedsRateLimiting = nominatimURL.includes("nominatim.openstreetmap.org");

export type LocationSearchResult = {
    name: string;
    description: string;
    lat: number;
    lon: number;
    category: string;
    type: string;
}

type LocationSearchCacheEntry = {
    data: Hits<LocationSearchResult>;
    expires: number;
};

type ReverseLocationCacheEntry = {
    data: NominatimResponse["features"];
    expires: number;
};

const locationSearchCache = new Map<string, LocationSearchCacheEntry>();
const reverseLocationCache = new Map<string, ReverseLocationCacheEntry>();
const locationSearchPending = new Map<string, Promise<Hits<LocationSearchResult>>>();
const reverseLocationPending = new Map<string, Promise<NominatimResponse["features"]>>();

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
    const cacheKey = `${q}::${limit ?? "default"}`;
    const now = Date.now();

    const cached = locationSearchCache.get(cacheKey);
    if (cached && cached.expires > now) {
        return cached.data;
    }

    const inPending = locationSearchPending.get(cacheKey);
    if (inPending) {
        return inPending;
    }

    const requestPromise = (async () => {
        const r = await fetchNominatim(`${nominatimURL}/search?q=${q}&format=geojson&addressdetails=1${limit ? '&limit=' + limit : ''}`);
        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail)
        }
        const response: NominatimResponse = await r.json();
        const results: Hits<LocationSearchResult> = response.features.map(f => ({
            category: f.properties.category,
            type: f.properties.type == "administrative" ? f.properties.addresstype : f.properties.type,
            description: getLocationDescription(f.properties.address),
            name: f.properties.name.length ? f.properties.name : f.properties.display_name,
            lat: f.geometry.coordinates[1],
            lon: f.geometry.coordinates[0],
        }));

        locationSearchCache.set(cacheKey, {
            data: results,
            expires: Date.now() + LOCATION_SEARCH_CACHE_TTL_MS,
        });

        return results;
    })();

    locationSearchPending.set(cacheKey, requestPromise);
    try {
        return await requestPromise;
    } finally {
        locationSearchPending.delete(cacheKey);
    }
}

export enum LocationDetails {
    NONE = 0,
    COUNTRY = 1 << 0,
    STATE = 1 << 1,
    CITY = 1 << 2,
    STREET = 1 << 4,
    NUMBER = 1 << 5,
    ALL = COUNTRY | STATE | CITY | STREET | NUMBER,
}

const waitTimer = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));
async function nominatimRateLimiter() {
    if (!nominatimNeedsRateLimiting) {
        return;
    }

    const elapsedTime_MS = Date.now() - lastNominatimCall;
    const waitTime = NOMINATIM_RATE_LIMIT_MS - elapsedTime_MS;
    if (waitTime > 0) {
        await waitTimer(waitTime);
    }
    
    lastNominatimCall = Date.now();
}

async function fetchNominatim(url: string): Promise<Response> {
    let attempt = 0;

    while (true) {
        await nominatimRateLimiter();

        try {
            return await fetch(url, {
                method: "GET",
                headers: new Headers({
                    "User-Agent": nominatimUserAgent
                })
            });
        } catch (error) {
            if (attempt < NOMINATIM_MAX_RETRIES) {
                attempt++;
                continue;
            }
            throw error;
        }
    }
}

export async function searchLocationReverse(lat: number, lon: number, details: LocationDetails = LocationDetails.COUNTRY | LocationDetails.STATE | LocationDetails.CITY): Promise<string> {
    const cacheKey = `${lat}:${lon}`;
    const now = Date.now();

    const cached = reverseLocationCache.get(cacheKey);
    if (cached && cached.expires > now) {
        return doGetLocationDescription(cached.data, details);
    }

    const inPending = reverseLocationPending.get(cacheKey);
    if (inPending) {
        const features = await inPending;
        return doGetLocationDescription(features, details);
    }

    const requestPromise = (async () => {
        const r = await fetchNominatim(`${nominatimURL}/reverse?lat=${lat}&lon=${lon}&format=geojson&addressdetails=1`);
        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail)
        }
        const response: NominatimResponse = await r.json();

        const features = response.features ?? [];

        reverseLocationCache.set(cacheKey, {
            data: features,
            expires: Date.now() + REVERSE_LOCATION_CACHE_TTL_MS,
        });

        return features;
    })();

    reverseLocationPending.set(cacheKey, requestPromise);

    try {
        const features = await requestPromise;
        return doGetLocationDescription(features, details);
    } finally {
        reverseLocationPending.delete(cacheKey);
    }
}

function doGetLocationDescription(features: Feature[] | undefined, details: LocationDetails = LocationDetails.COUNTRY | LocationDetails.STATE | LocationDetails.CITY): string {
    const address = features?.at(0)?.properties.address;
    return address ? getLocationDescription(address, details) : "";
}

function getLocationDescription(address: Address, details: LocationDetails = LocationDetails.COUNTRY | LocationDetails.STATE | LocationDetails.CITY) {
    let description = ""

    if ((details & LocationDetails.COUNTRY) == LocationDetails.COUNTRY && address.country) {
        description = address.country;
    }

    if ((details & LocationDetails.STATE) == LocationDetails.STATE && address.state) {
        description = `${address.state}, ` + description
    }

    if ((details & LocationDetails.CITY) == LocationDetails.CITY) {
    if (address.city) {
        description = `${address.city}, ` + description
    } else if (address.town) {
        description = `${address.town}, ` + description
    } else if (address.hamlet) {
        description = `${address.hamlet}, ` + description
        } 
        if (address.village) {
        description = `${address.village}, ` + description
    }
    }

    if ((details & LocationDetails.STREET) == LocationDetails.STREET && address.road) {
        description = `${address.road}, ` + description

        if ((details & LocationDetails.NUMBER) == LocationDetails.NUMBER && address.amenity) {
            description = `${address.amenity} ` + description
        }
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
