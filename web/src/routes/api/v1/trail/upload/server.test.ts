import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

const external = vi.hoisted(() => ({
    fromFile: vi.fn(), gpx2trail: vi.fn(), searchLocationReverse: vi.fn(), trails_create: vi.fn(),
}));
vi.mock("$lib/util/gpx_util", () => ({ fromFile: external.fromFile, gpx2trail: external.gpx2trail }));
vi.mock("$lib/stores/search_store", () => ({ searchLocationReverse: external.searchLocationReverse }));
vi.mock("$lib/stores/trail_store", () => ({ trails_create: external.trails_create }));

import { PUT } from "./+server";

const trail = { name: "New upload", distance: 1000, elevation_gain: 200, elevation_loss: 150, lat: 47, lon: 8 };
const duplicate = { id: "existing", name: "Existing route", author_name: "another-author", domain: "remote.example" };

function request(hits: unknown[] = [], options: { force?: boolean; authenticated?: boolean } = {}) {
    const search = vi.fn().mockResolvedValue({ hits });
    const index = vi.fn(() => ({ search }));
    const form = new FormData();
    form.set("file", new File(["<gpx/>"], "upload.gpx"));
    if (options.force) form.set("ignoreDuplicates", "true");
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

describe("targeted upload duplicate detection", () => {
    beforeEach(() => {
        vi.resetAllMocks();
        external.fromFile.mockResolvedValue({ gpxData: "<gpx/>", gpxFile: new Blob(["<gpx/>"]) });
        external.gpx2trail.mockResolvedValue({ trail: { ...trail } });
        external.searchLocationReverse.mockResolvedValue("Test location");
        external.trails_create.mockResolvedValue({ id: "created" });
    });

    it("asks the tenant search client for one matching candidate using every duplicate criterion", async () => {
        const { event, index, search } = request([duplicate]);
        const response = await PUT(event);

        expect(response.status).toBe(400);
        expect(await response.json()).toMatchObject({
            message: "Duplicate trail", id: duplicate.id, name: duplicate.name, domain: "another-author@remote.example",
        });
        expect(external.trails_create).not.toHaveBeenCalled();
        expect(index).toHaveBeenCalledWith("trails");
        expect(search).toHaveBeenCalledExactlyOnceWith("", {
            filter: [
                "_geoRadius(47, 8, 100)",
                "distance > 950 AND distance < 1050",
                "elevation_gain > 150 AND elevation_gain < 250",
                "elevation_loss > 100 AND elevation_loss < 200",
            ],
            attributesToRetrieve: ["id", "name", "author_name", "domain"],
            limit: 1,
        });
    });

    it("creates once after one empty filtered answer, without paging through other trails", async () => {
        const { event, search } = request();
        const response = await PUT(event);
        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ id: "created" });
        expect(search).toHaveBeenCalledOnce();
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it.each([{ lat: 0, lon: 8 }, { lat: 47, lon: 0 }])("keeps zero coordinates for %j", async (coordinates) => {
        external.gpx2trail.mockResolvedValue({ trail: { ...trail, ...coordinates } });
        const { event, search } = request();
        await PUT(event);
        expect(search).toHaveBeenCalledOnce();
        expect(search.mock.calls[0][1].filter[0]).toBe(`_geoRadius(${coordinates.lat}, ${coordinates.lon}, 100)`);
    });

    it("retains zero defaults for absent track measurements", async () => {
        external.gpx2trail.mockResolvedValue({ trail: { name: "Missing measurements" } });
        const { event, search } = request();
        await PUT(event);
        expect(search.mock.calls[0][1].filter).toEqual([
            "_geoRadius(0, 0, 100)",
            "distance > -50 AND distance < 50",
            "elevation_gain > -50 AND elevation_gain < 50",
            "elevation_loss > -50 AND elevation_loss < 50",
        ]);
    });

    it("allows force upload without searching", async () => {
        const { event, index } = request([duplicate], { force: true });
        const response = await PUT(event);
        expect(response.status).toBe(200);
        expect(index).not.toHaveBeenCalled();
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("requires authentication even for force upload", async () => {
        const { event, index } = request([], { authenticated: false, force: true });
        expect((await PUT(event)).status).toBe(401);
        expect(index).not.toHaveBeenCalled();
        expect(external.fromFile).not.toHaveBeenCalled();
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it("does not create a trail if the candidate search fails", async () => {
        const { event, search } = request();
        search.mockRejectedValue(new Error("Search unavailable"));
        const response = await PUT(event);
        expect(response.status).toBe(500);
        expect(await response.json()).toMatchObject({ message: "Error checking for duplicates" });
        expect(search).toHaveBeenCalledOnce();
        expect(external.trails_create).not.toHaveBeenCalled();
    });
});
