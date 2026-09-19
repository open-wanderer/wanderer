import type { RequestEvent } from "@sveltejs/kit";
import { MeilisearchApiError } from "meilisearch";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { POST as search } from "../../routes/api/v1/search/[index]/+server";
import { POST as multiSearch } from "../../routes/api/v1/search/multi/+server";
import { POST as cluster } from "../../routes/api/v1/search/trails/cluster/+server";
import { GET as boundingBox } from "../../routes/api/v1/trail/bounding-box/+server";

vi.mock("$lib/server/category_preference_filter", () => ({
    withTrailPreferenceMeiliFilter: async (_event: unknown, filter: unknown) => filter,
}));

const routes = [
    { name: "single-index search", handler: search, body: { q: "ridge", options: {} } },
    { name: "multi-index search", handler: multiSearch, body: { queries: [{ indexUid: "trails", q: "ridge" }] } },
    { name: "map clusters", handler: cluster, body: { southWest: { lat: 46, lng: 7 }, northEast: { lat: 47, lng: 8 }, zoom: 8 } },
    { name: "trail bounding box", handler: boundingBox, body: {} },
];

function eventFor(body: unknown, engineRequest: ReturnType<typeof vi.fn>): RequestEvent {
    return {
        params: { index: "trails" },
        request: new Request("http://localhost/api/v1/search/trails", {
            method: "POST",
            body: JSON.stringify(body),
            headers: { "content-type": "application/json" },
        }),
        locals: {
            pb: { authStore: { record: { id: "alice" } } },
            ms: { index: () => ({ search: engineRequest }), multiSearch: engineRequest },
        },
    } as unknown as RequestEvent;
}

describe.each(routes)("$name error status", ({ handler, body }) => {
    beforeEach(() => vi.spyOn(console, "error").mockImplementation(() => {}));
    afterEach(() => vi.restoreAllMocks());

    it.each([400, 403, 404, 503])("preserves SDK status %i", async (status) => {
        const failure = new MeilisearchApiError(new Response(null, { status }));
        const engineRequest = vi.fn().mockRejectedValue(failure);

        await expect(handler(eventFor(body, engineRequest))).rejects.toMatchObject({ status });
        expect(engineRequest).toHaveBeenCalledOnce();
    });

    it("reports transport failures as 500", async () => {
        const engineRequest = vi.fn().mockRejectedValue(new Error("connection failed"));

        await expect(handler(eventFor(body, engineRequest))).rejects.toMatchObject({ status: 500 });
        expect(engineRequest).toHaveBeenCalledOnce();
    });
});

it("preserves a successful search response", async () => {
    const result = { hits: [{ id: "trail-one" }], estimatedTotalHits: 1 };
    const engineRequest = vi.fn().mockResolvedValue(result);

    const response = await search(eventFor({ q: "ridge", options: { limit: 7 } }, engineRequest));

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual(result);
    expect(engineRequest).toHaveBeenCalledWith("ridge", { limit: 7, filter: undefined });
});
