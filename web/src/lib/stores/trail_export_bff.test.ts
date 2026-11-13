import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GET } from "../../routes/api/v1/trail/+server";
import { GET as detailGET } from "../../routes/api/v1/trail/[id]/+server";
import { ownedTrailExportPages } from "./trail_export_store";
import { trail2gpx } from "#lib/util/gpx_util.ts";
import GPX from "#lib/models/gpx/gpx.ts";

afterEach(() => vi.unstubAllGlobals());
const owner = "ownactor0000001";
const foreign = "otheractor00001";
const trails = [
    { id: "trailhidden0001", author: owner, category: "hiddenclass0001", name: "Hidden own trail", gpx: "", collectionId: "trails", collectionName: "trails", expand: { author: { preferred_username: "owner" } } },
    { id: "trailvisible001", author: owner, category: "", name: "Visible own trail", gpx: "", collectionId: "trails", collectionName: "trails", expand: { author: { preferred_username: "owner" } } },
    { id: "trailforeign001", author: foreign, category: "", name: "Foreign public trail", public: true, gpx: "", collectionId: "trails", collectionName: "trails", expand: { author: { preferred_username: "other" } } },
];

// Real BFF handlers and PocketBase SDK serialize, page, filter and enrich the
// responses. This HTTP fixture only supplies the lower-level PB API, including
// its omitted backrelations; the outer export API is never replaced with data.
function exportFixture(proofFailure = false) {
    const backendCalls: URL[] = [];
    const detailCalls: URL[] = [];
    const backend = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input));
        backendCalls.push(url);
        const collection = url.pathname.split("/")[3];
        const filter = url.searchParams.get("filter") ?? "";
        let items: unknown[] = [];
        if (collection === "trails") {
            if (url.pathname.endsWith("/records")) {
                items = trails.filter((trail) => filter.includes(`author = "${trail.author}"`) && !(filter.includes('category != "hiddenclass0001"') && trail.category === "hiddenclass0001"));
            } else return Response.json(structuredClone(trails.find((trail) => url.pathname.endsWith(trail.id))));
        } else if (collection === "user_category_preferences") items = [{ id: "preference", category: "hiddenclass0001", visible: false }];
        else if (collection === "waypoints") {
            if (proofFailure) return Response.json({ message: "Child data unavailable" }, { status: 503 });
            if (filter.includes("trailvisible001")) items = [{ id: "waypoint0000001", trail: "trailvisible001", name: "Recorded summit", lat: 47, lon: 8, distance_from_start: 10 }];
        }
        return Response.json({ page: 1, perPage: Number(url.searchParams.get("perPage") ?? 100), totalItems: items.length, totalPages: items.length ? 1 : 0, items: structuredClone(items) });
    });
    const pb = new PocketBase("https://a.example");
    pb.beforeSend = (url, options) => ({ url, options: { ...options, fetch: backend } });
    const event = (input: RequestInfo | URL, authenticated = true): RequestEvent => {
        const url = new URL(String(input), "https://b.example");
        return { url, request: new Request(url), params: { id: url.pathname.split("/").at(-1) }, locals: { pb, user: authenticated ? { id: "owner-user", actor: owner } : undefined }, fetch: backend } as unknown as RequestEvent;
    };
    const request = async (input: RequestInfo | URL) => {
        const e = event(input);
        if (e.url.pathname === "/api/v1/trail") return GET(e);
        if (e.url.pathname.startsWith("/api/v1/trail/")) { detailCalls.push(e.url); return detailGET(e); }
        throw new Error(`Unexpected export request ${input}`);
    };
    return { backendCalls, detailCalls, event, request };
}

describe("owner export through actual BFF and PocketBase transport", () => {
    it("exports hidden own trails and prevents a caller filter from widening the owner scope", async () => {
        const f = exportFixture();
        const normal = await GET(f.event('/api/v1/trail?photos=false&filter=' + encodeURIComponent(`author = "${owner}"`)));
        expect((await normal.json()).items.map((trail: { id: string }) => trail.id)).toEqual(["trailvisible001"]);
        f.backendCalls.length = 0;
        const exported = [];
        // The client actor/filter is deliberately foreign. The server's
        // authenticated actor must decide ownership independently.
        for await (const page of ownedTrailExportPages(foreign, false, f.request)) exported.push(...page);
        expect(exported.map((trail) => trail.id)).toEqual(["trailhidden0001", "trailvisible001"]);
        const query = f.backendCalls.find((url) => url.pathname === "/api/collections/trails/records")!;
        expect(query.searchParams.get("filter")).toBe(`author = "${owner}"`);
        expect(query.searchParams.get("sort")).toBe("id");
        expect(f.backendCalls.some((url) => url.pathname.includes("preferences"))).toBe(false);
        expect((await GET(f.event("/api/v1/trail?export=owned", false))).status).toBe(401);
        const large = await GET(f.event("/api/v1/trail?export=owned&photos=false&perPage=1000"));
        expect(large.status).toBe(200);
        expect(f.backendCalls.filter((url) => url.pathname.endsWith("/trails/records")).at(-1)?.searchParams.get("perPage")).toBe("100");
    });

    it("writes metadata-only GPX without a photo/detail fallback for PB's missing child expands", async () => {
        const f = exportFixture();
        vi.stubGlobal("fetch", f.request);
        const names: string[] = [];
        for await (const page of ownedTrailExportPages(owner, false, f.request)) {
            for (const trail of page) {
                // Both native PB records omitted waypoints. The BFF must prove
                // emptiness or recover actual children before GPX generation.
                trail.expand!.gpx_data = "";
                const parsed = GPX.parse(await trail2gpx(trail));
                expect(parsed).not.toBeInstanceOf(Error);
                names.push((parsed as GPX).metadata!.name!);
                expect(f.detailCalls).toEqual([]);
                expect(f.backendCalls.some((url) => url.pathname.includes("_assets") || (url.searchParams.get("expand") ?? "").includes("_assets"))).toBe(false);
                if (trail.id === "trailvisible001") expect((parsed as GPX).wpt?.[0].name).toBe("Recorded summit");
                else expect((parsed as GPX).wpt?.length ?? 0).toBe(0);
            }
        }
        expect(names).toEqual(["Hidden own trail", "Visible own trail"]);
        expect(f.detailCalls).toEqual([]);
        expect(f.backendCalls.some((url) => url.pathname.includes("_assets") || (url.searchParams.get("expand") ?? "").includes("_assets"))).toBe(false);
    });

    it("fails instead of exporting incomplete metadata when the child proof is unavailable", async () => {
        const f = exportFixture(true);
        const response = await GET(f.event("/api/v1/trail?export=owned&photos=false"));
        expect(response.status).toBe(503);
        expect(f.detailCalls).toEqual([]);
    });
});
