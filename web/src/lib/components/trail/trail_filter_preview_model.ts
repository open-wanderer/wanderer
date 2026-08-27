export type PreviewRouteShape = "all" | "loop" | "point-to-point";
export type PreviewSlope = "all" | "gentle" | "balanced" | "steep";

export type PreviewCountFacet =
    | "taxonomy"
    | "location"
    | "routeShape"
    | "photos"
    | "transit"
    | "difficulty";

export type PreviewLocation = {
    id: string;
    name: string;
    region: string;
    regionEn?: string;
    countAt25Km: number;
};

export type PreviewTaxonomySelection = {
    categoryId: string;
    subcategoryIds: string[];
};

export type PreviewFilterState = {
    query: string;
    taxonomy: PreviewTaxonomySelection[];
    locationId: string | null;
    locationRadius: number;
    routeShape: PreviewRouteShape;
    photosOnly: boolean;
    publicTransport: boolean;
    distanceMin: number;
    distanceMax: number;
    durationMin: number;
    durationMax: number;
    difficulty: number[];
    elevationMin: number;
    elevationMax: number;
    altitudeMin: number;
    altitudeMax: number;
    slope: PreviewSlope;
    minimumRating: number;
};

export type PreviewTrail = {
    id: number;
    categoryId: string | null;
    subcategoryId: string | null;
    routeShape: Exclude<PreviewRouteShape, "all">;
    hasPhotos: boolean;
    hasTransit: boolean;
    difficulty: 0 | 1 | 2;
    distance: number;
    duration: number;
    elevation: number;
    altitude: number;
    slope: Exclude<PreviewSlope, "all">;
    rating: number;
    searchRank: number;
    locationDistancesKm: Record<string, number>;
};

type PreviewSubcategoryDefinition = {
    id: string;
    categoryId: string;
};

export const PREVIEW_BASE_RESULT_COUNT = 243;
export const PREVIEW_CATEGORY_WEIGHTS = [164, 91, 58, 37, 22, 14, 8, 5];
export const PREVIEW_LOCATIONS: PreviewLocation[] = [
    { id: "bern", name: "Bern", region: "Bern", countAt25Km: 72 },
    {
        id: "interlaken",
        name: "Interlaken",
        region: "Berner Oberland",
        regionEn: "Bernese Oberland",
        countAt25Km: 64,
    },
    {
        id: "davos",
        name: "Davos",
        region: "Graubünden",
        regionEn: "Grisons",
        countAt25Km: 51,
    },
    {
        id: "zermatt",
        name: "Zermatt",
        region: "Wallis",
        regionEn: "Valais",
        countAt25Km: 46,
    },
    {
        id: "luzern",
        name: "Luzern",
        region: "Zentralschweiz",
        regionEn: "Central Switzerland",
        countAt25Km: 43,
    },
    {
        id: "appenzell",
        name: "Appenzell",
        region: "Alpstein",
        countAt25Km: 38,
    },
];

export const PREVIEW_DISTANCE_BINS = [
    2, 7, 18, 37, 64, 91, 100, 94, 79, 61, 46, 34, 24, 17, 11, 8, 5,
    4, 2, 1,
];
export const PREVIEW_DURATION_BINS = [
    3, 12, 31, 59, 88, 100, 92, 74, 55, 39, 27, 18, 12, 8, 5, 3, 2, 1,
];
export const PREVIEW_ELEVATION_BINS = [
    18, 38, 69, 94, 100, 91, 76, 59, 42, 31, 22, 15, 10, 7, 4, 3, 2, 1,
];
export const PREVIEW_ALTITUDE_BINS = [
    8, 18, 34, 52, 75, 96, 100, 87, 69, 53, 39, 27, 18, 12, 8, 5, 3, 2,
];

const DISTANCE_CAP = 50_000;
const DURATION_CAP = 36_000;
const ELEVATION_CAP = 2_000;
const ALTITUDE_CAP = 3_500;
const SUBCATEGORY_WEIGHTS = [57, 42, 31, 23, 16, 11];

export function buildPreviewTrailCorpus(
    categoryIds: string[],
    subcategories: PreviewSubcategoryDefinition[],
): PreviewTrail[] {
    const categoryAllocations = apportion(
        PREVIEW_BASE_RESULT_COUNT,
        categoryIds.map(
            (_, index) => PREVIEW_CATEGORY_WEIGHTS[index] ?? Math.max(1, 6 - index),
        ),
    );
    const categorySlots = categoryIds.flatMap((categoryId, index) =>
        Array(categoryAllocations[index] ?? 0).fill(categoryId),
    );

    const trails: PreviewTrail[] = Array.from(
        { length: PREVIEW_BASE_RESULT_COUNT },
        (_, id) => {
            const categorySlot = permutedIndex(id, 137, 17);
            const routeSlot = permutedIndex(id, 151, 29);
            const difficultySlot = permutedIndex(id, 167, 71);
            const slopeSlot = permutedIndex(id, 173, 47);
            const ratingSlot = permutedIndex(id, 179, 89);

            return {
                id,
                categoryId: categorySlots[categorySlot] ?? null,
                subcategoryId: null,
                routeShape: routeSlot < 126 ? "loop" : "point-to-point",
                hasPhotos: permutedIndex(id, 157, 41) < 198,
                hasTransit: permutedIndex(id, 163, 53) < 87,
                difficulty:
                    difficultySlot < 132
                        ? 0
                        : difficultySlot < 213
                          ? 1
                          : 2,
                distance: sampleHistogramValue(
                    id,
                    PREVIEW_DISTANCE_BINS,
                    DISTANCE_CAP,
                    181,
                    11,
                ),
                duration: sampleHistogramValue(
                    id,
                    PREVIEW_DURATION_BINS,
                    DURATION_CAP,
                    187,
                    31,
                ),
                elevation: sampleHistogramValue(
                    id,
                    PREVIEW_ELEVATION_BINS,
                    ELEVATION_CAP,
                    193,
                    61,
                ),
                altitude: sampleHistogramValue(
                    id,
                    PREVIEW_ALTITUDE_BINS,
                    ALTITUDE_CAP,
                    197,
                    83,
                ),
                slope:
                    slopeSlot < 105
                        ? "gentle"
                        : slopeSlot < 193
                          ? "balanced"
                          : "steep",
                rating:
                    ratingSlot < 24
                        ? 5
                        : ratingSlot < 81
                          ? 4
                          : ratingSlot < 143
                            ? 3
                            : ratingSlot < 192
                              ? 2
                              : 1,
                searchRank: permutedIndex(id, 211, 101) / 242,
                locationDistancesKm: Object.fromEntries(
                    PREVIEW_LOCATIONS.map((location, index) => [
                        location.id,
                        previewLocationDistance(id, location, index),
                    ]),
                ),
            };
        },
    );

    assignSubcategories(trails, categoryIds, subcategories);
    return trails;
}

export function filterPreviewTrails(
    trails: PreviewTrail[],
    state: PreviewFilterState,
    omitFacet?: PreviewCountFacet,
): PreviewTrail[] {
    return trails.filter((trail) => matchesPreviewTrail(trail, state, omitFacet));
}

export function matchesPreviewTrail(
    trail: PreviewTrail,
    state: PreviewFilterState,
    omitFacet?: PreviewCountFacet,
): boolean {
    const queryLength = state.query.trim().length;
    if (
        queryLength > 0 &&
        trail.searchRank > Math.max(0.24, 1 - queryLength * 0.055)
    ) {
        return false;
    }
    if (!inCappedRange(trail.distance, state.distanceMin, state.distanceMax, DISTANCE_CAP)) {
        return false;
    }
    if (!inCappedRange(trail.duration, state.durationMin, state.durationMax, DURATION_CAP)) {
        return false;
    }
    if (!inCappedRange(trail.elevation, state.elevationMin, state.elevationMax, ELEVATION_CAP)) {
        return false;
    }
    if (!inCappedRange(trail.altitude, state.altitudeMin, state.altitudeMax, ALTITUDE_CAP)) {
        return false;
    }
    if (state.slope !== "all" && trail.slope !== state.slope) {
        return false;
    }
    if (state.minimumRating > 0 && trail.rating < state.minimumRating) {
        return false;
    }
    if (omitFacet !== "taxonomy" && !matchesTaxonomy(trail, state.taxonomy)) {
        return false;
    }
    if (
        omitFacet !== "location" &&
        state.locationId &&
        (trail.locationDistancesKm[state.locationId] ?? Number.POSITIVE_INFINITY) >
            state.locationRadius
    ) {
        return false;
    }
    if (
        omitFacet !== "routeShape" &&
        state.routeShape !== "all" &&
        trail.routeShape !== state.routeShape
    ) {
        return false;
    }
    if (omitFacet !== "photos" && state.photosOnly && !trail.hasPhotos) {
        return false;
    }
    if (omitFacet !== "transit" && state.publicTransport && !trail.hasTransit) {
        return false;
    }
    if (
        omitFacet !== "difficulty" &&
        state.difficulty.length > 0 &&
        !state.difficulty.includes(trail.difficulty)
    ) {
        return false;
    }

    return true;
}

export function countByKey(
    trails: PreviewTrail[],
    key: (trail: PreviewTrail) => string | null,
): Map<string, number> {
    const counts = new Map<string, number>();
    for (const trail of trails) {
        const value = key(trail);
        if (!value) {
            continue;
        }
        counts.set(value, (counts.get(value) ?? 0) + 1);
    }
    return counts;
}

function matchesTaxonomy(
    trail: PreviewTrail,
    taxonomy: PreviewTaxonomySelection[],
): boolean {
    if (taxonomy.length === 0) {
        return true;
    }

    const parent = taxonomy.find(
        (selection) => selection.categoryId === trail.categoryId,
    );
    if (!parent) {
        return false;
    }
    if (parent.subcategoryIds.length === 0) {
        return true;
    }
    return Boolean(
        trail.subcategoryId && parent.subcategoryIds.includes(trail.subcategoryId),
    );
}

function inCappedRange(
    value: number,
    minimum: number,
    maximum: number,
    cap: number,
): boolean {
    return value >= minimum && (maximum === cap || value <= maximum);
}

function assignSubcategories(
    trails: PreviewTrail[],
    categoryIds: string[],
    subcategories: PreviewSubcategoryDefinition[],
) {
    categoryIds.forEach((categoryId, categoryIndex) => {
        const children = subcategories
            .filter((subcategory) => subcategory.categoryId === categoryId)
            .sort((a, b) => a.id.localeCompare(b.id));
        if (children.length === 0) {
            return;
        }

        const categoryTrails = trails
            .filter((trail) => trail.categoryId === categoryId)
            .sort(
                (a, b) =>
                    permutedIndex(a.id, 149, categoryIndex * 19 + 7) -
                    permutedIndex(b.id, 149, categoryIndex * 19 + 7),
            );
        const assignedCount = Math.round(categoryTrails.length * 0.86);
        const allocations = apportion(
            assignedCount,
            children.map(
                (_, index) =>
                    SUBCATEGORY_WEIGHTS[index] ?? Math.max(1, 8 - index),
            ),
        );

        let offset = 0;
        children.forEach((subcategory, index) => {
            const end = offset + (allocations[index] ?? 0);
            for (const trail of categoryTrails.slice(offset, end)) {
                trail.subcategoryId = subcategory.id;
            }
            offset = end;
        });
    });
}

function apportion(total: number, weights: number[]): number[] {
    if (weights.length === 0) {
        return [];
    }

    const safeWeights = weights.map((weight) => Math.max(0, weight));
    const weightTotal = safeWeights.reduce((sum, weight) => sum + weight, 0);
    if (weightTotal === 0) {
        return safeWeights.map((_, index) => (index < total ? 1 : 0));
    }

    const exact = safeWeights.map((weight) => (weight / weightTotal) * total);
    const result = exact.map(Math.floor);
    let remainder = total - result.reduce((sum, value) => sum + value, 0);
    const order = exact
        .map((value, index) => ({ index, fraction: value - Math.floor(value) }))
        .sort((a, b) => b.fraction - a.fraction || a.index - b.index);

    for (let index = 0; index < remainder; index += 1) {
        result[order[index % order.length].index] += 1;
    }
    return result;
}

const permutationCache = new Map<string, number[]>();

function permutedIndex(index: number, multiplier: number, offset: number): number {
    const key = `${multiplier}:${offset}`;
    let ranks = permutationCache.get(key);
    if (!ranks) {
        const values = Array.from(
            { length: PREVIEW_BASE_RESULT_COUNT },
            (_, value) => value,
        );
        const random = seededRandom(Math.imul(multiplier, 0x9e3779b1) ^ offset);
        for (let cursor = values.length - 1; cursor > 0; cursor -= 1) {
            const swapIndex = Math.floor(random() * (cursor + 1));
            [values[cursor], values[swapIndex]] = [
                values[swapIndex],
                values[cursor],
            ];
        }

        ranks = Array(PREVIEW_BASE_RESULT_COUNT);
        values.forEach((value, rank) => {
            ranks![value] = rank;
        });
        permutationCache.set(key, ranks);
    }
    return ranks[index];
}

function seededRandom(seed: number): () => number {
    let state = seed >>> 0;
    return () => {
        state = (state + 0x6d2b79f5) >>> 0;
        let value = state;
        value = Math.imul(value ^ (value >>> 15), value | 1);
        value ^= value + Math.imul(value ^ (value >>> 7), value | 61);
        return ((value ^ (value >>> 14)) >>> 0) / 4_294_967_296;
    };
}

function sampleHistogramValue(
    id: number,
    bins: number[],
    cap: number,
    multiplier: number,
    offset: number,
): number {
    const weightTotal = bins.reduce((sum, weight) => sum + weight, 0);
    const target =
        (permutedIndex(id, multiplier, offset) / PREVIEW_BASE_RESULT_COUNT) *
        weightTotal;
    let cumulative = 0;
    let binIndex = bins.length - 1;
    for (let index = 0; index < bins.length; index += 1) {
        cumulative += bins[index];
        if (target < cumulative) {
            binIndex = index;
            break;
        }
    }

    const withinBin =
        permutedIndex(id, multiplier + 12, offset + 37) /
        PREVIEW_BASE_RESULT_COUNT;
    const binWidth = cap / bins.length;
    const value = (binIndex + 0.08 + withinBin * 0.84) * binWidth;
    return Math.round(value);
}

function previewLocationDistance(
    id: number,
    location: PreviewLocation,
    locationIndex: number,
): number {
    const multipliers = [139, 145, 152, 160, 169, 175];
    const rank = permutedIndex(
        id,
        multipliers[locationIndex % multipliers.length],
        locationIndex * 29 + 11,
    );
    const at25 = Math.min(PREVIEW_BASE_RESULT_COUNT, location.countAt25Km);
    const at5 = Math.round(at25 * 0.38);
    const at10 = Math.round(at25 * 0.57);
    const at50 = Math.min(
        PREVIEW_BASE_RESULT_COUNT,
        Math.round(at25 * 1.55),
    );

    if (rank < at5) {
        return interpolateRank(rank, 0, at5, 0.5, 5);
    }
    if (rank < at10) {
        return interpolateRank(rank, at5, at10, 5.01, 10);
    }
    if (rank < at25) {
        return interpolateRank(rank, at10, at25, 10.01, 25);
    }
    if (rank < at50) {
        return interpolateRank(rank, at25, at50, 25.01, 50);
    }
    return interpolateRank(
        rank,
        at50,
        PREVIEW_BASE_RESULT_COUNT,
        50.01,
        180,
    );
}

function interpolateRank(
    rank: number,
    startRank: number,
    endRank: number,
    startValue: number,
    endValue: number,
): number {
    const span = Math.max(1, endRank - startRank);
    const progress = (rank - startRank + 0.5) / span;
    return startValue + (endValue - startValue) * progress;
}
