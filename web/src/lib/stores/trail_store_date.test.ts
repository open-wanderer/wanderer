import type { TrailFilter } from "$lib/models/trail";
import { LngLat } from "maplibre-gl";
import { afterEach, describe, expect, it, vi } from "vitest";
import { trails_search_bounding_box, trails_search_filter } from "./trail_store";

type DateFilter = Pick<TrailFilter, "startDate" | "endDate">;

function trailFilter(dates: DateFilter): TrailFilter {
    return {
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty: [],
        near: { radius: 2000 },
        distanceMin: 0,
        distanceMax: 100000,
        distanceLimit: 100000,
        elevationGainMin: 0,
        elevationGainMax: 10000,
        elevationGainLimit: 10000,
        elevationLossMin: 0,
        elevationLossMax: 10000,
        elevationLossLimit: 10000,
        sort: "created",
        sortOrder: "-",
        ...dates,
    };
}

function searchRequest() {
    return vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
        Response.json({ hits: [], page: 1, totalPages: 0 }),
    );
}

function requestDateClauses(request: ReturnType<typeof searchRequest>): string[] {
    expect(request).toHaveBeenCalledOnce();
    const [url, options] = request.mock.calls[0];
    expect(url).toBe("/api/v1/search/trails");
    expect(options?.method).toBe("POST");
    const filter: string = JSON.parse(String(options?.body)).options.filter;
    expect(filter).toContain("distance >= 0");
    return filter.match(/\bdate (?:>=|<=|<) \S+/g) ?? [];
}

async function listDateClauses(dates: DateFilter): Promise<string[]> {
    const request = searchRequest();
    await trails_search_filter(trailFilter(dates), 1, 20, request);
    return requestDateClauses(request);
}

afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
});

describe("trail search date ranges", () => {
    it.each([
        "UTC",
        "Europe/Zurich",
        "America/Los_Angeles",
        "Asia/Kathmandu",
    ])("sends UTC calendar-day bounds and includes stored dates in %s", async (timezone) => {
        vi.stubEnv("TZ", timezone);

        const clauses = await listDateClauses({ startDate: "2026-09-07", endDate: "2026-09-07" });
        expect(clauses).toEqual([
            `date >= ${Date.parse("2026-09-07T00:00:00Z") / 1000}`,
            `date < ${Date.parse("2026-09-08T00:00:00Z") / 1000}`,
        ]);

        const [start, end] = clauses.map((clause) => Number(clause.split(" ").at(-1)));
        const storedDates = [
            "2026-09-06T23:59:59Z",
            "2026-09-07T00:00:00Z", // A manually entered calendar date.
            "2026-09-07T23:59:59Z", // An imported timestamp late on the end date.
            "2026-09-08T00:00:00Z",
        ];
        expect(storedDates.filter((date) => {
            const timestamp = Date.parse(date) / 1000;
            return timestamp >= start && timestamp < end;
        })).toEqual([
            "2026-09-07T00:00:00Z",
            "2026-09-07T23:59:59Z",
        ]);
    });

    it("uses the same UTC calendar-day bounds for the map search", async () => {
        vi.stubEnv("TZ", "America/Los_Angeles");
        const request = searchRequest();
        vi.stubGlobal("fetch", request);

        await trails_search_bounding_box(
            new LngLat(9, 48),
            new LngLat(7, 46),
            trailFilter({ startDate: "2026-09-07", endDate: "2026-09-09" }),
            1, 11, 20, false,
        );

        expect(requestDateClauses(request)).toEqual([
            `date >= ${Date.parse("2026-09-07T00:00:00Z") / 1000}`,
            `date < ${Date.parse("2026-09-10T00:00:00Z") / 1000}`,
        ]);
    });

    it("supports a start date without an end date", async () => {
        vi.stubEnv("TZ", "Europe/Zurich");
        expect(await listDateClauses({ startDate: "2026-09-07" })).toEqual([
            `date >= ${Date.parse("2026-09-07T00:00:00Z") / 1000}`,
        ]);
    });

    it("includes the whole end date without a start date", async () => {
        vi.stubEnv("TZ", "Europe/Zurich");
        expect(await listDateClauses({ endDate: "2026-09-07" })).toEqual([
            `date < ${Date.parse("2026-09-08T00:00:00Z") / 1000}`,
        ]);
    });

    it.each([undefined, "", "invalid", "2026-02-30", "2026-13-01", "2026-09-07T12:00:00Z"])(
        "omits invalid or absent calendar date %s",
        async (value) => {
            expect(await listDateClauses({ startDate: value, endDate: value })).toEqual([]);
        },
    );

    it("retains a valid boundary when the other boundary is invalid", async () => {
        vi.stubEnv("TZ", "UTC");
        expect(await listDateClauses({ startDate: "2026-02-30", endDate: "2026-09-07" })).toEqual([
            `date < ${Date.parse("2026-09-08T00:00:00Z") / 1000}`,
        ]);
    });
});
