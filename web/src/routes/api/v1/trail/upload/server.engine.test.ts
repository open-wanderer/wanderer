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
const ownActor = "own-actor";
const base = { distance: 1000, elevation_gain: 200, elevation_loss: 150, lat: 47, lon: 8 };
type Measurements = typeof base;
type DuplicateCheckOptions = { includePublic?: boolean; includeShared?: boolean } | null;
type Visibility = { author?: string; public?: boolean; shares?: string[] };

function document(id: string, measurements: Measurements = base, visibility: Visibility = {}) {
    const { lat, lon, ...metrics } = measurements;
    const author = visibility.author ?? ownActor;
    return {
        id, ...metrics, _geo: { lat, lng: lon },
        author, public: visibility.public ?? false, shares: visibility.shares ?? [],
        name: "Existing route", author_name: author === ownActor ? "uploader" : "another-author", domain: "",
    };
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
            filterableAttributes: ["id", "public", "author", "shares", "_geo", "distance", "elevation_gain", "elevation_loss"],
            pagination: { maxTotalHits: 1000 },
        }).waitTask()).status).toBe("succeeded");
        const docs = Array.from({ length: 20000 }, (_, i) => document(`unrelated-${i}`, { ...base, distance: 100000 + i }));
        docs.push(document("match"));
        docs.push(document("private", { ...base, distance: 5000 }, { author: "private-owner" }));
        docs.push(document("far", { ...base, distance: 6000, lat: 47.002 }));
        docs.push(document("zero", { ...base, distance: 7000, lat: 0, lon: 0 }));
        // Meilisearch's radius includes its boundary, unlike the former JS < 100 comparison.
        docs.push(document("radius-boundary", { ...base, distance: 8000, lat: 47 + 100 / 6371000 * 180 / Math.PI }));
        docs.push(document("own-public", { ...base, distance: 9000 }, { public: true }));
        docs.push(document("foreign-public", { ...base, distance: 10000 }, { author: "public-owner", public: true }));
        docs.push(document("foreign-shared", { ...base, distance: 11000 }, { author: "shared-owner", shares: [ownActor] }));
        docs.push(document("shared-with-someone-else", { ...base, distance: 12000 }, { author: "private-owner", shares: ["other-actor"] }));
        for (const { field, difference, input } of boundaries) {
            docs.push(document(`${field}-${difference}`, { ...input, [field]: input[field] + difference }));
        }
        expect((await index.addDocuments(docs).waitTask({ timeout: 30000 })).status).toBe("succeeded");
        const key = await admin.createKey({ name: uid, actions: ["search"], indexes: [uid], expiresAt: null });
        keyUid = key.uid;
        const token = await generateTenantToken({
            apiKey: key.key, apiKeyUid: key.uid,
            searchRules: { [uid]: { filter: `public = true OR author = "${ownActor}" OR shares = "${ownActor}"` } },
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

    async function upload(measurements: Measurements, uploadDuplicateCheck?: DuplicateCheckOptions) {
        external.gpx2trail.mockResolvedValue({ trail: { name: "New upload", ...measurements } });
        const search = vi.fn((q: string, options: SearchParams) => tenantIndex.search(q, options));
        const form = new FormData();
        form.set("file", new File(["<gpx/>"], "upload.gpx"));
        const event = {
            request: new Request("http://localhost/api/v1/trail/upload", { method: "PUT", body: form }),
            locals: {
                user: { id: "uploader", actor: ownActor },
                settings: { privacy: { trails: "private" }, uploadDuplicateCheck },
                ms: { index: () => ({ search }) },
            },
            fetch: vi.fn(),
        } as unknown as RequestEvent;
        const response = await PUT(event);
        expect(search).toHaveBeenCalledOnce();
        expect(search.mock.calls[0][1].limit).toBe(1);
        return { status: response.status, body: await response.json() };
    }

    it("finds an own private duplicate among 20,000 unrelated trails in one request by default", async () => {
        expect(await upload(base)).toMatchObject({ status: 400, body: { id: "match", author: ownActor } });
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it("also includes own public trails by default", async () => {
        expect(await upload({ ...base, distance: 9000 })).toMatchObject({ status: 400, body: { id: "own-public", author: ownActor } });
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it.each<DuplicateCheckOptions | undefined>([undefined, null, {}])(
        "excludes other authors' public trails with default settings %j",
        async (options) => {
            expect(await upload({ ...base, distance: 10000 }, options)).toMatchObject({ status: 200, body: { id: "created" } });
            expect(external.trails_create).toHaveBeenCalledOnce();
        },
    );

    it("excludes privately shared trails from other authors by default", async () => {
        expect(await upload({ ...base, distance: 11000 })).toMatchObject({ status: 200, body: { id: "created" } });
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it.each([
        { name: "public only", options: { includePublic: true }, distance: 10000, id: "foreign-public", author: "public-owner" },
        { name: "shared only", options: { includeShared: true }, distance: 11000, id: "foreign-shared", author: "shared-owner" },
        { name: "both options for public", options: { includePublic: true, includeShared: true }, distance: 10000, id: "foreign-public", author: "public-owner" },
        { name: "both options for shared", options: { includePublic: true, includeShared: true }, distance: 11000, id: "foreign-shared", author: "shared-owner" },
        { name: "both options still include owned", options: { includePublic: true, includeShared: true }, distance: 1000, id: "match", author: ownActor },
    ])("includes selected candidates: $name", async ({ options, distance, id, author }) => {
        expect(await upload({ ...base, distance }, options)).toMatchObject({ status: 400, body: { id, author } });
        expect(external.trails_create).not.toHaveBeenCalled();
    });

    it.each([
        { name: "public does not enable shared", options: { includePublic: true, includeShared: false }, distance: 11000 },
        { name: "shared does not enable public", options: { includePublic: false, includeShared: true }, distance: 10000 },
    ])("keeps options independent: $name", async ({ options, distance }) => {
        expect(await upload({ ...base, distance }, options)).toMatchObject({ status: 200, body: { id: "created" } });
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it.each<DuplicateCheckOptions | undefined>([
        undefined, { includePublic: true }, { includeShared: true }, { includePublic: true, includeShared: true },
    ])("never exposes another author's private unshared trail with options %j", async (options) => {
        expect(await upload({ ...base, distance: 5000 }, options)).toMatchObject({ status: 200, body: { id: "created" } });
        expect(external.trails_create).toHaveBeenCalledOnce();
    });

    it("does not include private trails shared only with another actor", async () => {
        expect(await upload({ ...base, distance: 12000 }, { includePublic: true, includeShared: true })).toMatchObject({ status: 200, body: { id: "created" } });
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
