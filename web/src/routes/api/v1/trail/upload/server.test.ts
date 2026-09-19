import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

const external = vi.hoisted(() => ({
    fromFile: vi.fn(),
    gpx2trail: vi.fn(),
    searchLocationReverse: vi.fn(),
    trails_create: vi.fn(),
}));

vi.mock("$lib/util/gpx_util", () => ({
    fromFile: external.fromFile,
    gpx2trail: external.gpx2trail,
}));
vi.mock("$lib/stores/search_store", () => ({ searchLocationReverse: external.searchLocationReverse }));
vi.mock("$lib/stores/trail_store", () => ({ trails_create: external.trails_create }));

import { PUT } from "./+server";

function candidate(id: string) {
    return {
        id, name: "Existing route", author_name: "another-author", domain: "remote.example",
        distance: 1000, elevation_gain: 200, elevation_loss: 150,
        _geo: { lat: 47, lng: 8 },
    };
}

function unrelated(count: number, offset = 0) {
    return Array.from({ length: count }, (_, i) => ({
        ...candidate(`unrelated-${offset + i}`), distance: 10000 + i,
    }));
}

function request(pages: unknown[][], options: { ignoreDuplicates?: string; authenticated?: boolean } = {}) {
    const search = vi.fn();
    for (const hits of pages) search.mockResolvedValueOnce({ hits });
    search.mockResolvedValue({ hits: [] });
    const index = vi.fn(() => ({ search }));
    const form = new FormData();
    form.set("file", new File(["<gpx/>"], "upload.gpx"));
    if (options.ignoreDuplicates !== undefined) form.set("ignoreDuplicates", options.ignoreDuplicates);
    const event = {
        request: new Request("http://localhost/api/v1/trail/upload", { method: "PUT", body: form }),
        locals: {
            user: options.authenticated === false ? null : { id: "uploader", actor: "own-actor" },
            settings: { privacy: { trails: "private" } },
            ms: { index },
        },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
    return { event, search, index };
}

async function expectDuplicate(response: Response, id = "match") {
    expect(response.status).toBe(400);
    expect(await response.json()).toMatchObject({
        message: "Duplicate trail", id, name: "Existing route", domain: "another-author@remote.example",
    });
    expect(external.trails_create).not.toHaveBeenCalled();
}

describe("upload duplicate detection", () => {
    beforeEach(() => {
        vi.resetAllMocks();
        external.fromFile.mockResolvedValue({ gpxData: "<gpx/>", gpxFile: new Blob(["<gpx/>"]) });
        external.gpx2trail.mockResolvedValue({
            trail: { name: "New upload", distance: 1000, elevation_gain: 200, elevation_loss: 150, lat: 47, lon: 8 },
        });
        external.searchLocationReverse.mockResolvedValue("Test location");
        external.trails_create.mockResolvedValue({ id: "created" });
    });

    it.each([20, 1000])("finds a duplicate after %i unrelated visible search hits", async (count) => {
        const preceding = unrelated(count);
        const pages = [];
        for (let offset = 0; offset < preceding.length; offset += 500) pages.push(preceding.slice(offset, offset + 500));
        pages.push([candidate("match")]);
        const { event, search, index } = request(pages);

        await expectDuplicate(await PUT(event));

        expect(index).toHaveBeenCalledWith("trails");
        expect(search).toHaveBeenCalledTimes(pages.length);
        expect(search).toHaveBeenNthCalledWith(1, "", {
            attributesToRetrieve: ["id", "name", "author_name", "domain", "distance", "elevation_gain", "elevation_loss", "_geo"],
            filter: [], offset: 0, limit: 500,
        });
        expect(search.mock.calls.at(-1)?.[1].filter).toEqual([
            "id NOT IN " + JSON.stringify(preceding.map(hit => hit.id)),
        ]);
    });

    it("continues after short pages and creates once only after an empty remainder", async () => {
        const first = unrelated(3);
        const second = unrelated(2, 3);
        const { event, search } = request([first, second]);
        search.mockImplementationOnce(async () => {
            expect(external.trails_create).not.toHaveBeenCalled();
            return { hits: [] };
        });

        const response = await PUT(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ id: "created" });
        expect(search).toHaveBeenCalledTimes(3);
        expect(search.mock.calls[2][1].filter).toEqual([
            "id NOT IN " + JSON.stringify([...first, ...second].map(hit => hit.id)),
        ]);
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("stops searching immediately after a matching first page", async () => {
        const { event, search } = request([[candidate("match")], unrelated(20)]);

        await expectDuplicate(await PUT(event));

        expect(search).toHaveBeenCalledOnce();
    });

    it("allows an explicit forced upload without searching", async () => {
        const { event, index } = request([[candidate("match")]], { ignoreDuplicates: "true" });

        const response = await PUT(event);

        expect(response.status).toBe(200);
        expect(index).not.toHaveBeenCalled();
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("requires authentication even when duplicate detection is bypassed", async () => {
        const { event, index } = request([], { authenticated: false, ignoreDuplicates: "true" });

        const response = await PUT(event);

        expect(response.status).toBe(401);
        expect(index).not.toHaveBeenCalled();
        expect(external.fromFile).not.toHaveBeenCalled();
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it("does not create a trail when a later search request fails", async () => {
        const { event, search } = request([unrelated(20)]);
        search.mockRejectedValueOnce(new Error("Search unavailable"));

        const response = await PUT(event);

        expect(response.status).toBe(500);
        expect(await response.json()).toMatchObject({ message: "Error checking for duplicates" });
        expect(search).toHaveBeenCalledTimes(2);
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it.each([
        { name: "a repeated ID in a later page", pages: [[...unrelated(1)], [...unrelated(1)]], calls: 2 },
        { name: "a repeated ID in the same page", pages: [[...unrelated(1), ...unrelated(1)]], calls: 1 },
        { name: "a missing document ID", pages: [[{ ...unrelated(1)[0], id: undefined }]], calls: 1 },
    ])("fails without creating a trail on $name", async ({ pages, calls }) => {
        const { event, search } = request(pages);

        const response = await PUT(event);

        expect(response.status).toBe(500);
        expect(await response.json()).toMatchObject({ message: "Error checking for duplicates" });
        expect(search).toHaveBeenCalledTimes(calls);
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it.each(["distance", "elevation_gain", "elevation_loss"] as const)(
        "retains the strict 50 metre difference boundary for %s",
        async (field) => {
            const hit = candidate("boundary");
            hit[field] += 50;
            const { event } = request([[hit], []]);

            const response = await PUT(event);

            expect(response.status).toBe(200);
            expect(external.trails_create).toHaveBeenCalledOnce();
        },
    );

    it("does not consider a matching-length route with a distant start a duplicate", async () => {
        const hit = candidate("distant-start");
        hit._geo.lat += 0.002; // About 222 metres, safely beyond the 100 metre radius.
        const { event } = request([[hit], []]);

        const response = await PUT(event);

        expect(response.status).toBe(200);
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("preserves approximate matching regardless of author or name", async () => {
        const hit = candidate("match");
        hit.distance += 49;
        hit.elevation_gain -= 49;
        hit.elevation_loss += 49;
        hit._geo.lat += 0.0005; // About 56 metres, inside the existing radius.
        const { event } = request([[hit]]);

        await expectDuplicate(await PUT(event));
    });
});
