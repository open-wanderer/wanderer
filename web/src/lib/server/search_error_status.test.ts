import { error, type RequestEvent } from "@sveltejs/kit";
import { MeilisearchApiError, MeilisearchRequestError, MeilisearchRequestTimeOutError } from "meilisearch";
import { ClientResponseError } from "pocketbase";
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

    it("reports an invalid search filter as a client error", async () => {
        const detail = { message: "Invalid filter", code: "invalid_search_filter", type: "invalid_request", link: "https://example.invalid/filter" };
        const failure = new MeilisearchApiError(new Response(null, { status: 400 }), detail);
        const engineRequest = vi.fn().mockRejectedValue(failure);

        await expect(handler(eventFor(body, engineRequest))).rejects.toMatchObject({ status: 400, body: detail });
        expect(engineRequest).toHaveBeenCalledOnce();
    });

    it.each([
        [403, "invalid_api_key"],
        [404, "index_not_found"],
        [503, "unavailable"],
        [400, "invalid_search_sort"],
        [400, "invalid_search_limit"],
        [403, "invalid_search_filter"],
    ] as const)("maps upstream %i (%s) to a generic 502", async (status, code) => {
        const failure = new MeilisearchApiError(new Response(null, { status }), {
            message: "Upstream access or configuration failure", code, type: "invalid_request", link: "https://example.invalid/internal",
        });
        const engineRequest = vi.fn().mockRejectedValue(failure);

        const failureResponse = await handler(eventFor(body, engineRequest)).catch(e => e);

        expect(failureResponse.status).toBe(502);
        expect(failureResponse.body).toEqual({ message: "Search service unavailable" });
        expect(engineRequest).toHaveBeenCalledOnce();
    });

    it.each([
        new MeilisearchRequestError("http://internal-search:7700", new TypeError("fetch failed")),
        new MeilisearchRequestTimeOutError(1000, {}),
        new MeilisearchRequestError("http://internal-search:7700", new MeilisearchRequestTimeOutError(1000, {})),
    ])("maps SDK transport and timeout failures to a generic 502", async (failure) => {
        const engineRequest = vi.fn().mockRejectedValue(failure);

        const failureResponse = await handler(eventFor(body, engineRequest)).catch(e => e);

        expect(failureResponse.status).toBe(502);
        expect(failureResponse.body).toEqual({ message: "Search service unavailable" });
    });

    it("keeps unknown local failures as 500", async () => {
        const engineRequest = vi.fn().mockRejectedValue(new Error("Local processing failed"));

        await expect(handler(eventFor(body, engineRequest))).rejects.toMatchObject({ status: 500 });
        expect(engineRequest).toHaveBeenCalledOnce();
    });

    it("preserves locally raised HTTP errors", async () => {
        let failure: unknown;
        try { error(409, "Local conflict"); } catch (e) { failure = e; }
        const engineRequest = vi.fn().mockRejectedValue(failure);

        await expect(handler(eventFor(body, engineRequest))).rejects.toBe(failure);
    });

    it("preserves PocketBase error status", async () => {
        const engineRequest = vi.fn().mockRejectedValue(new ClientResponseError({ status: 403, response: { message: "PocketBase access denied" } }));

        await expect(handler(eventFor(body, engineRequest))).rejects.toMatchObject({ status: 403 });
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
