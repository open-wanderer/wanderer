import type { TrailFilter } from "$lib/models/trail";

const TRAIL_SORT_OPTIONS = new Set([
    "name",
    "distance",
    "duration",
    "difficulty",
    "elevation_gain",
    "elevation_loss",
    "like_count",
    "created",
    "date",
]);

function limitToRange(value: number, min: number, max: number): number {
    return Math.min(Math.max(value, min), max);
}

function getNumber(value: unknown, fallback: number): number;
function getNumber(value: unknown, fallback?: number): number | undefined;
function getNumber(value: unknown, fallback?: number): number | undefined {
    return typeof value === "number" ? value : fallback;
}

function getNumberInRange(
    value: unknown,
    fallback: number,
    min: number,
    max: number,
): number {
    return limitToRange(getNumber(value, fallback), min, max);
}

function asRecord(value: unknown): Record<string, unknown> {
    return value !== null && typeof value === "object"
        ? (value as Record<string, unknown>)
        : {};
}

function getString(value: unknown, fallback: string): string;
function getString(value: unknown, fallback?: string): string | undefined;
function getString(value: unknown, fallback?: string): string | undefined {
    return typeof value === "string" ? value : fallback;
}

function getStringArray(value: unknown, fallback: string[]): string[] {
    return Array.isArray(value)
        ? value.filter((item): item is string => typeof item === "string")
        : fallback;
}

function toDifficulty(value: unknown): 0 | 1 | 2 | undefined {
    if (value === 0 || value === 1 || value === 2) {
        return value;
    }
    if (value === "easy") {
        return 0;
    }
    if (value === "moderate") {
        return 1;
    }
    if (value === "difficult") {
        return 2;
    }
    return undefined;
}

function parseDifficulty(
    value: unknown,
    fallback: (0 | 1 | 2)[],
): (0 | 1 | 2)[] {
    if (!Array.isArray(value)) {
        return fallback;
    }

    const parsed = value
        .map(toDifficulty)
        .filter((d): d is 0 | 1 | 2 => d !== undefined);

    return parsed.length ? parsed : fallback;
}

function getBoolean(value: unknown, fallback?: boolean): boolean | undefined {
    return typeof value === "boolean" ? value : fallback;
}

export function sanitizeTrailFilter(
    candidate: unknown,
    defaultFilter: TrailFilter,
): TrailFilter {
    const source = asRecord(candidate);
    const near = asRecord(source.near);

    const restored: TrailFilter = {
        ...defaultFilter,
        q: getString(source.q, defaultFilter.q),
        category: getStringArray(source.category, defaultFilter.category),
        tags: getStringArray(source.tags, defaultFilter.tags),
        difficulty: parseDifficulty(source.difficulty, defaultFilter.difficulty),
        author: getString(source.author, defaultFilter.author),
        public: getBoolean(source.public, defaultFilter.public),
        shared: getBoolean(source.shared, defaultFilter.shared),
        private: getBoolean(source.private, defaultFilter.private),
        near: {
            lat: getNumber(near.lat, defaultFilter.near.lat),
            lon: getNumber(near.lon, defaultFilter.near.lon),
            radius: Math.max(1, getNumber(near.radius, defaultFilter.near.radius)),
        },
        distanceLimit: defaultFilter.distanceLimit,
        elevationGainLimit: defaultFilter.elevationGainLimit,
        elevationLossLimit: defaultFilter.elevationLossLimit,
        distanceMin: getNumberInRange(
            source.distanceMin,
            defaultFilter.distanceMin,
            0,
            defaultFilter.distanceLimit,
        ),
        distanceMax: getNumberInRange(
            source.distanceMax,
            defaultFilter.distanceMax,
            0,
            defaultFilter.distanceLimit,
        ),
        elevationGainMin: getNumberInRange(
            source.elevationGainMin,
            defaultFilter.elevationGainMin,
            0,
            defaultFilter.elevationGainLimit,
        ),
        elevationGainMax: getNumberInRange(
            source.elevationGainMax,
            defaultFilter.elevationGainMax,
            0,
            defaultFilter.elevationGainLimit,
        ),
        elevationLossMin: getNumberInRange(
            source.elevationLossMin,
            defaultFilter.elevationLossMin,
            0,
            defaultFilter.elevationLossLimit,
        ),
        elevationLossMax: getNumberInRange(
            source.elevationLossMax,
            defaultFilter.elevationLossMax,
            0,
            defaultFilter.elevationLossLimit,
        ),
        startDate: getString(source.startDate, defaultFilter.startDate),
        endDate: getString(source.endDate, defaultFilter.endDate),
        completed: getBoolean(source.completed, defaultFilter.completed),
        liked: getBoolean(source.liked, defaultFilter.liked),
        sort:
            typeof source.sort === "string" && TRAIL_SORT_OPTIONS.has(source.sort)
                ? (source.sort as TrailFilter["sort"])
                : defaultFilter.sort,
        sortOrder: source.sortOrder === "-" ? "-" : "+",
    };

    if (restored.distanceMin > restored.distanceMax) {
        restored.distanceMin = restored.distanceMax;
    }
    if (restored.elevationGainMin > restored.elevationGainMax) {
        restored.elevationGainMin = restored.elevationGainMax;
    }
    if (restored.elevationLossMin > restored.elevationLossMax) {
        restored.elevationLossMin = restored.elevationLossMax;
    }

    return restored;
}
