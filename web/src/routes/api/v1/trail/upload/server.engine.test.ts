// Run with DUPLICATE_TEST_MEILI_URL and DUPLICATE_TEST_MEILI_KEY pointing to a test engine.
// Each run creates and removes its own index and search key.
import { randomUUID } from "node:crypto";
import type { RequestEvent } from "@sveltejs/kit";
import { Meilisearch, type SearchParams } from "meilisearch";
import { generateTenantToken } from "meilisearch/token";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const external = vi.hoisted(() => ({ gpx2trail: vi.fn(), trails_create: vi.fn() }));
vi.mock("$lib/util/gpx_util", () => ({
    fromFile: async () => ({ gpxData: "<gpx/>", gpxFile: new Blob(["<gpx/>"]) }),
    gpx2trail: external.gpx2trail,
}));
vi.mock("$lib/stores/search_store", () => ({ searchLocationReverse: async () => "Test location" }));
vi.mock("$lib/stores/trail_store", () => ({ trails_create: external.trails_create }));
import { PUT } from "./+server";

const host = process.env.DUPLICATE_TEST_MEILI_URL;
const apiKey = process.env.DUPLICATE_TEST_MEILI_KEY;
const uid = `duplicate_regression_${randomUUID().replaceAll("-", "")}`;
const base = { distance: 1000, elevation_gain: 200, elevation_loss: 150, lat: 47, lon: 8 };
type Measurements = typeof base;

function document(id: string, measurements: Measurements = base, visible = true) {
    const { lat, lon, ...metrics } = measurements;
    return { id, ...metrics, _geo: { lat, lng: lon }, public: visible, name: "Existing route", author_name: "another-author", domain: "" };
}

describe.skipIf(!host || !apiKey)("duplicate search against Meilisearch", () => {
    let admin: Meilisearch;
    let tenantIndex: ReturnType<Meilisearch["index"]>;
    let keyUid: string | undefined;
    let created = false;
    const boundaries = (["distance", "elevation_gain", "elevation_loss"] as const).flatMap((field, i) =>
        [-50, 50].map((difference, j) => ({ field, difference, input: { ...base, distance: 20000 + 1000 * (i * 2 + j) } })),
    );

    beforeAll(async () => {
        admin = new Meilisearch({ host: host!, apiKey });
        expect((await admin.createIndex(uid, { primaryKey: "id" }).waitTask()).status).toBe("succeeded");
        created = true;
        const index = admin.index(uid);
        expect((await index.updateSettings({
            filterableAttributes: ["id", "public", "_geo", "distance", "elevation_gain", "elevation_loss"],
            pagination: { maxTotalHits: 1000 },
        }).waitTask()).status).toBe("succeeded");
        const docs = Array.from({ length: 20000 }, (_, i) => document(`unrelated-${i}`, { ...base, distance: 100000 + i }));
        docs.push(document("match"));
        docs.push(document("private", { ...base, distance: 5000 }, false));
        docs.push(document("far", { ...base, distance: 6000, lat: 47.002 }));
        docs.push(document("zero", { ...base, distance: 7000, lat: 0, lon: 0 }));
        // Meilisearch's radius includes its boundary, unlike the former JS < 100 comparison.
        docs.push(document("radius-boundary", { ...base, distance: 8000, lat: 47 + 100 / 6371000 * 180 / Math.PI }));
        for (const { field, difference, input } of boundaries) {
            docs.push(document(`${field}-${difference}`, { ...input, [field]: input[field] + difference }));
        }
        expect((await index.addDocuments(docs).waitTask({ timeout: 30000 })).status).toBe("succeeded");
        const key = await admin.createKey({ name: uid, actions: ["search"], indexes: [uid], expiresAt: null });
        keyUid = key.uid;
        const token = await generateTenantToken({
            apiKey: key.key, apiKeyUid: key.uid, searchRules: { [uid]: { filter: "public = true" } },
            expiresAt: new Date(Date.now() + 300000),
        });
        tenantIndex = new Meilisearch({ host: host!, apiKey: token }).index(uid);
    }, 45000);

    afterAll(async () => {
        if (created) await admin.deleteIndex(uid).waitTask();
        if (keyUid) await admin.deleteKey(keyUid);
    });

    beforeEach(() => {
        vi.resetAllMocks();
        external.trails_create.mockResolvedValue({ id: "created" });
    });

    async function upload(measurements: Measurements) {
        external.gpx2trail.mockResolvedValue({ trail: { name: "New upload", ...measurements } });
        const search = vi.fn((q: string, options: SearchParams) => tenantIndex.search(q, options));
        const form = new FormData();
        form.set("file", new File(["<gpx/>"], "upload.gpx"));
        const event = {
            request: new Request("http://localhost/api/v1/trail/upload", { method: "PUT", body: form }),
            locals: { user: { id: "uploader" }, settings: { privacy: { trails: "private" } }, ms: { index: () => ({ search }) } },
            fetch: vi.fn(),
        } as unknown as RequestEvent;
        const response = await PUT(event);
        expect(search).toHaveBeenCalledOnce();
        expect(search.mock.calls[0][1].limit).toBe(1);
        return { status: response.status, body: await response.json() };
    }

    it("finds a matching trail among 20,000 unrelated trails in one request", async () => {
        expect(await upload(base)).toMatchObject({ status: 400, body: { id: "match" } });
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it("does not expose a private duplicate through the tenant search", async () => {
        expect(await upload({ ...base, distance: 5000 })).toMatchObject({ status: 200, body: { id: "created" } });
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it.each(boundaries)("excludes the exact $difference m boundary for $field", async ({ input }) => {
        expect((await upload(input)).status).toBe(200);
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("excludes a distant starting point", async () => {
        expect((await upload({ ...base, distance: 6000 })).status).toBe(200);
    });

    it("searches at zero coordinates", async () => {
        expect(await upload({ ...base, distance: 7000, lat: 0, lon: 0 })).toMatchObject({ status: 400, body: { id: "zero" } });
    });

    it("uses the engine's inclusive 100 m radius", async () => {
        expect(await upload({ ...base, distance: 8000 })).toMatchObject({ status: 400, body: { id: "radius-boundary" } });
    });
});
