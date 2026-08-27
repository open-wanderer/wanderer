import { describe, expect, it } from "vitest";
import {
    PREVIEW_BASE_RESULT_COUNT,
    PREVIEW_LOCATIONS,
    buildPreviewTrailCorpus,
    countByKey,
    filterPreviewTrails,
    type PreviewFilterState,
} from "./trail_filter_preview_model";

const categories = ["hiking", "biking", "running"];
const subcategories = [
    { id: "alpine", categoryId: "hiking" },
    { id: "family", categoryId: "hiking" },
    { id: "gravel", categoryId: "biking" },
    { id: "mtb", categoryId: "biking" },
];
const trails = buildPreviewTrailCorpus(categories, subcategories);

describe("trail filter preview model", () => {
    it("builds a deterministic corpus with one primary category per route", () => {
        expect(trails).toHaveLength(PREVIEW_BASE_RESULT_COUNT);
        expect(buildPreviewTrailCorpus(categories, subcategories)).toEqual(trails);

        const categoryCounts = countByKey(trails, (trail) => trail.categoryId);
        expect([...categoryCounts.values()].reduce((sum, value) => sum + value, 0)).toBe(
            PREVIEW_BASE_RESULT_COUNT,
        );
    });

    it("uses the same evaluator for predictive facet counts and results", () => {
        const initial = defaultState();
        const routePool = filterPreviewTrails(trails, initial, "routeShape");
        const predictedLoops = routePool.filter(
            (trail) => trail.routeShape === "loop",
        ).length;
        expect(
            filterPreviewTrails(trails, {
                ...initial,
                routeShape: "loop",
            }),
        ).toHaveLength(predictedLoops);

        const photosPool = filterPreviewTrails(trails, initial, "photos");
        const predictedPhotos = photosPool.filter(
            (trail) => trail.hasPhotos,
        ).length;
        expect(
            filterPreviewTrails(trails, { ...initial, photosOnly: true }),
        ).toHaveLength(predictedPhotos);

        const transitPool = filterPreviewTrails(trails, initial, "transit");
        const predictedTransit = transitPool.filter(
            (trail) => trail.hasTransit,
        ).length;
        expect(
            filterPreviewTrails(trails, {
                ...initial,
                publicTransport: true,
            }),
        ).toHaveLength(predictedTransit);

        const difficultyPool = filterPreviewTrails(
            trails,
            initial,
            "difficulty",
        );
        const predictedEasy = difficultyPool.filter(
            (trail) => trail.difficulty === 0,
        ).length;
        expect(
            filterPreviewTrails(trails, { ...initial, difficulty: [0] }),
        ).toHaveLength(predictedEasy);

        const location = PREVIEW_LOCATIONS[0];
        const locationPool = filterPreviewTrails(trails, initial, "location");
        const predictedLocation = locationPool.filter(
            (trail) => trail.locationDistancesKm[location.id] <= 25,
        ).length;
        expect(
            filterPreviewTrails(trails, {
                ...initial,
                locationId: location.id,
            }),
        ).toHaveLength(predictedLocation);
    });

    it("keeps OR selections monotonic within the taxonomy facet", () => {
        const initial = defaultState();
        const hiking = filterPreviewTrails(trails, {
            ...initial,
            taxonomy: [{ categoryId: "hiking", subcategoryIds: [] }],
        }).length;
        const alpine = filterPreviewTrails(trails, {
            ...initial,
            taxonomy: [
                { categoryId: "hiking", subcategoryIds: ["alpine"] },
            ],
        }).length;
        const alpineAndFamily = filterPreviewTrails(trails, {
            ...initial,
            taxonomy: [
                {
                    categoryId: "hiking",
                    subcategoryIds: ["alpine", "family"],
                },
            ],
        }).length;
        const hikingAndBiking = filterPreviewTrails(trails, {
            ...initial,
            taxonomy: [
                { categoryId: "hiking", subcategoryIds: [] },
                { categoryId: "biking", subcategoryIds: [] },
            ],
        }).length;

        expect(alpine).toBeLessThan(hiking);
        expect(alpineAndFamily).toBeGreaterThan(alpine);
        expect(alpineAndFamily).toBeLessThanOrEqual(hiking);
        expect(hikingAndBiking).toBeGreaterThan(hiking);
    });

    it("widens location and numeric ranges monotonically", () => {
        const initial = defaultState();
        const locationId = PREVIEW_LOCATIONS[0].id;
        const at5Km = filterPreviewTrails(trails, {
            ...initial,
            locationId,
            locationRadius: 5,
        }).length;
        const at25Km = filterPreviewTrails(trails, {
            ...initial,
            locationId,
            locationRadius: 25,
        }).length;
        const at50Km = filterPreviewTrails(trails, {
            ...initial,
            locationId,
            locationRadius: 50,
        }).length;
        const halfDistance = filterPreviewTrails(trails, {
            ...initial,
            distanceMax: 25_000,
        }).length;

        expect(at5Km).toBeLessThan(at25Km);
        expect(at25Km).toBeLessThan(at50Km);
        expect(halfDistance).toBeLessThan(PREVIEW_BASE_RESULT_COUNT);
        expect(filterPreviewTrails(trails, initial)).toHaveLength(
            PREVIEW_BASE_RESULT_COUNT,
        );
    });
});

function defaultState(): PreviewFilterState {
    return {
        query: "",
        taxonomy: [],
        locationId: null,
        locationRadius: 25,
        routeShape: "all",
        photosOnly: false,
        publicTransport: false,
        distanceMin: 0,
        distanceMax: 50_000,
        durationMin: 0,
        durationMax: 36_000,
        difficulty: [],
        elevationMin: 0,
        elevationMax: 2_000,
        altitudeMin: 0,
        altitudeMax: 3_500,
        slope: "all",
        minimumRating: 0,
    };
}
