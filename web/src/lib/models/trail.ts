import cryptoRandomString from "crypto-random-string";
import type { Actor } from "./activitypub/actor";
import type { Category } from "./category";
import type { Comment } from "./comment";
import type GPX from "./gpx/gpx";
import type { Subcategory } from "./subcategory";
import { SummitLog } from "./summit_log";
import type { Tag } from "./tag";
import type { TrailLike } from "./trail_like";
import type { TrailShare } from "./trail_share";
import { Waypoint } from "./waypoint";

const defaultTrailDuplicateOptions: TrailDuplicateOptions = {
    waypoints: true,
    summitLogs: false,
    trailPhotos: false,
    waypointPhotos: false,
    summitLogPhotos: false,
};

class Trail {
    id?: string;
    name: string;
    location?: string;
    date?: string;
    public: boolean;
    completed: boolean;
    distance?: number;
    elevation_gain?: number;
    elevation_loss?: number;
    duration?: number;
    difficulty?: "easy" | "moderate" | "difficult"
    lat?: number;
    lon?: number;
    thumbnail?: number;
    photos: string[];
    gpx?: string;
    created?: string;
    updated?: string;
    category?: string;
    subcategory?: string;
    tags: string[];
    polyline?: string;
    domain?: string;
    iri?: string;
    like_count: number;
    bounding_box_diagonal?: number;
    expand?: {
        tags?: Tag[]
        category?: Category;
        subcategory?: Subcategory;
        waypoints_via_trail?: Waypoint[]
        summit_logs_via_trail?: SummitLog[]
        author?: Actor
        comments_via_trail?: Comment[]
        gpx_data?: string
        gpx?: GPX
        trail_share_via_trail?: TrailShare[]
        trail_like_via_trail?: TrailLike[]

    }
    description?: string;
    author: string;

    constructor(name: string,
        params?: {
            id?: string,
            location?: string,
            date?: string,
            public?: boolean,
            completed?: boolean,
            distance?: number,
            elevation_gain?: number,
            elevation_loss?: number,
            duration?: number,
            difficulty?: "easy" | "moderate" | "difficult",
            lat?: number,
            lon?: number,
            thumbnail?: number,
            photos?: string[],
            gpx?: string,
            gpx_data?: string,
            category?: Category,
            subcategory?: Subcategory,
            waypoints?: Waypoint[],
            summit_logs?: SummitLog[],
            comments?: Comment[],
            shares?: TrailShare[],
            tags?: Tag[],
            description?: string,
            created?: string,
            bounding_box_diagonal?: number
        }

    ) {
        this.id = params?.id;
        this.name = name;
        this.location = params?.location;
        this.date = params?.date ?? new Date().toISOString().split('T')[0];
        this.public = params?.public ?? false
        this.completed = params?.completed ?? false,
        this.distance = params?.distance ?? 0;
        this.elevation_gain = params?.elevation_gain ?? 0;
        this.elevation_loss = params?.elevation_loss ?? 0;
        this.duration = params?.duration ?? 0;
        this.difficulty = params?.difficulty ?? "easy";
        this.lat = params?.lat;
        this.lon = params?.lon;
        this.thumbnail = params?.thumbnail ?? 0;
        this.photos = params?.photos ?? [];
        this.tags = [];
        this.gpx = params?.gpx;
        this.category = params?.category?.id;
        this.subcategory = params?.subcategory?.id;
        this.bounding_box_diagonal = params?.bounding_box_diagonal ?? 0;
        this.like_count = 0
        this.expand = {
            category: params?.category,
            subcategory: params?.subcategory,
            waypoints_via_trail: params?.waypoints ?? [],
            summit_logs_via_trail: params?.summit_logs ?? [],
            comments_via_trail: params?.comments ?? [],
            trail_share_via_trail: params?.shares ?? [],
            gpx_data: params?.gpx_data,
            tags: params?.tags
        }
        this.description = params?.description ?? "";
        this.created = params?.created;
        this.author = "000000000000000"
    }

    static from(orig: Trail, actorId?: string, options: Partial<TrailDuplicateOptions> = {}): Trail {
        const duplicateOptions = normalizeTrailDuplicateOptions(options);

        return new Trail(orig.name, {
            date: orig.date,
            description: orig.description,
            difficulty: orig.difficulty,
            completed: orig.completed,
            distance: orig.distance,
            duration: orig.duration,
            elevation_gain: orig.elevation_gain,
            elevation_loss: orig.elevation_loss,
            thumbnail: orig.thumbnail,
            lat: orig.lat,
            lon: orig.lon,
            location: orig.location,
            public: orig.public,
            tags: orig.expand?.tags,
            category: orig.expand?.category,
            subcategory: orig.expand?.subcategory,
            gpx_data: orig.expand?.gpx_data,
            waypoints: duplicateOptions.waypoints
                ? orig.expand?.waypoints_via_trail?.map((waypoint) =>
                    cloneWaypoint(waypoint, actorId),
                )
                : [],
            summit_logs: duplicateOptions.summitLogs
                ? orig.expand?.summit_logs_via_trail?.map((log) =>
                    cloneSummitLog(log, actorId),
                )
                : [],
            bounding_box_diagonal: orig.bounding_box_diagonal,
        })
    }
}

function cloneWaypoint(wp: Waypoint, actorId?: string): Waypoint {
    const cloned = new Waypoint(wp.lat, wp.lon, {
        id: cryptoRandomString({ length: 15 }),
        description: wp.description,
        icon: wp.icon,
        name: wp.name,
    });
    cloned.author = actorId ?? wp.author;
    cloned.distance_from_start = wp.distance_from_start;
    addDuplicatePhotoSource(cloned, wp);
    return cloned;
}

function cloneSummitLog(log: SummitLog, actorId?: string): SummitLog {
    const cloned = new SummitLog(log.date, {
        id: cryptoRandomString({ length: 15 }),
        text: log.text,
        distance: log.distance,
        elevation_gain: log.elevation_gain,
        elevation_loss: log.elevation_loss,
        duration: log.duration,
    });
    cloned.author = actorId ?? log.author;
    cloned.expand = {
        gpx_data: log.expand?.gpx_data,
    };
    addDuplicatePhotoSource(cloned, log);
    return cloned;
}

interface DuplicatePhotoSource {
    id: string;
    collectionId?: string;
    collectionName?: string;
    photos: string[];
}

interface TrailDuplicateOptions {
    waypoints: boolean;
    summitLogs: boolean;
    trailPhotos: boolean;
    waypointPhotos: boolean;
    summitLogPhotos: boolean;
}

function normalizeTrailDuplicateOptions(
    options: Partial<TrailDuplicateOptions> = {},
): TrailDuplicateOptions {
    const normalized = {
        ...defaultTrailDuplicateOptions,
        ...options,
    };

    if (!normalized.waypoints) {
        normalized.waypointPhotos = false;
    }
    if (!normalized.summitLogs) {
        normalized.summitLogPhotos = false;
    }

    return normalized;
}

function hasDuplicatePhotos(options?: TrailDuplicateOptions) {
    return Boolean(
        options?.trailPhotos ||
        options?.waypointPhotos ||
        options?.summitLogPhotos,
    );
}

function addDuplicatePhotoSource<T extends object>(target: T, source: object) {
    const photoSource = duplicatePhotoSource(source);
    if (photoSource) {
        (target as T & { _duplicatePhotoSource?: DuplicatePhotoSource })._duplicatePhotoSource = photoSource;
    }
}

function duplicatePhotoSource(source: object): DuplicatePhotoSource | undefined {
    const record = source as {
        id?: string;
        collectionId?: string;
        collectionName?: string;
        photos?: string[];
    };
    if (!record.id || !record.photos?.length) {
        return undefined;
    }
    return {
        id: record.id,
        collectionId: record.collectionId,
        collectionName: record.collectionName,
        photos: record.photos,
    };
}

interface TrailFilter {
    q: string,
    category: string[],
    subcategory: string[],
    tags: string[],
    difficulty: (0 | 1 | 2)[]
    author?: string;
    public?: boolean;
    shared?: boolean;
    private?: boolean;
    near: {
        lat?: number,
        lon?: number,
        radius: number
    }
    distanceMin: number,
    distanceMax: number,
    distanceLimit: number,
    elevationGainMin: number;
    elevationGainMax: number;
    elevationGainLimit: number;
    elevationLossMin: number;
    elevationLossMax: number;
    elevationLossLimit: number;
    startDate?: string;
    endDate?: string;
    completed?: boolean;
    liked?: boolean;
    sort: "name" | "distance" | "elevation_gain" | "created";
    sortOrder: "+" | "-"
}

interface TrailFilterValues {
    min_distance: number,
    max_distance: number,
    min_elevation_gain: number,
    max_elevation_gain: number,
    min_elevation_loss: number,
    max_elevation_loss: number,
    min_durtation: number,
    max_duration: number,
}

interface TrailBoundingBox {
    max_lat: number,
    min_lat: number,
    max_lon: number,
    min_lon: number,
    has_trails?: boolean,
}


interface TrailSearchResult {
    id: string;
    author: string;
    author_name: string;
    author_avatar: string;
    name: string;
    description: string;
    location: string;
    distance: number;
    elevation_gain: number;
    elevation_loss: number;
    duration: number;
    difficulty: 0 | 1 | 2;
    category: string;
    category_id?: string | null;
    category_icon?: string;
    subcategory_id?: string | null;
    is_federated?: boolean;
    federated_category_name?: string | null;
    federated_subcategory_name?: string | null;
    completed: boolean;
    date: number;
    created: number;
    public: boolean;
    thumbnail: string;
    polyline?: string;
    likes?: string[];
    like_count: number;
    shares?: string[];
    tags?: string[]
    domain?: string;
    iri?: string;
    gpx: string;
    bounding_box_diagonal: number;
    _geo: {
        lat: number,
        lng: number
    };
}

export const defaultTrailSearchAttributes = [
    "id",
    "author",
    "author_name",
    "author_avatar",
    "name",
    "description",
    "location",
    "distance",
    "elevation_gain",
    "elevation_loss",
    "duration",
    "difficulty",
    "category",
    "category_id",
    "category_icon",
    "subcategory_id",
    "is_federated",
    "federated_category_name",
    "federated_subcategory_name",
    "completed",
    "date",
    "created",
    "public",
    "thumbnail",
    "domain",
    "gpx",
    "tags",
    "like_count",
    "shares",
    "iri",
    "bounding_box_diagonal",
    "_geo",]


export {
    Trail,
    defaultTrailDuplicateOptions,
    hasDuplicatePhotos,
    normalizeTrailDuplicateOptions,
};

export type {
    TrailBoundingBox,
    TrailDuplicateOptions,
    TrailFilter,
    TrailFilterValues,
    TrailSearchResult,
};
