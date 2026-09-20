import { error, isHttpError } from "@sveltejs/kit";
import { describe, expect, it } from "vitest";
import { ClientResponseError } from "pocketbase";
import { MeilisearchApiError, MeilisearchRequestError, MeilisearchRequestTimeOutError } from "meilisearch";
import { assertFileField, handleError } from "./api_util";

describe("handleError", () => {
    it("reports a confirmed invalid Meilisearch filter as 400 with its details", async () => {
        const detail = { message: "Search failed", code: "invalid_search_filter", type: "invalid_request", link: "https://example.invalid/error" };
        const failure = new MeilisearchApiError(new Response(null, { status: 400 }), detail);

        const response = handleError(failure);

        expect(response.status).toBe(400);
        expect(await response.json()).toEqual(detail);
    });

    it.each([
        [403, "invalid_api_key"], [404, "index_not_found"], [503, "unavailable"],
        [400, "invalid_search_sort"], [400, "invalid_search_limit"], [403, "invalid_search_filter"],
    ] as const)("maps upstream %i (%s) to a generic 502", async (status, code) => {
        const failure = new MeilisearchApiError(new Response(null, { status }), {
            message: "Internal search failure", code, type: "invalid_request", link: "https://example.invalid/internal",
        });

        const response = handleError(failure);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual({ message: "Search service unavailable" });
    });

    it.each([undefined, null, "invalid_search_filter", [], {}, { code: 400 }, { code: "invalid_search_filter_typo" }])(
        "does not classify missing or malformed error detail %j as a filter error",
        async (cause) => {
            const failure = new MeilisearchApiError(new Response(null, { status: 400 }));
            Object.assign(failure, { cause });

            const response = handleError(failure);

            expect(response.status).toBe(502);
            expect(await response.json()).toEqual({ message: "Search service unavailable" });
        },
    );

    it.each([
        new MeilisearchRequestError("http://internal-search:7700", new TypeError("fetch failed")),
        new MeilisearchRequestTimeOutError(1000, {}),
        new MeilisearchRequestError("http://internal-search:7700", new MeilisearchRequestTimeOutError(1000, {})),
    ])("maps actual SDK transport and timeout errors to a generic 502", async (failure) => {
        const response = handleError(failure);

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual({ message: "Search service unavailable" });
    });

    it("does not trust a local error that merely resembles an SDK error", () => {
        const failure = Object.assign(new Error("Local failure"), {
            response: { status: 400 }, cause: { code: "invalid_search_filter" },
        });

        expect(handleError(failure).status).toBe(500);
    });

    it("preserves PocketBase status and response details", async () => {
        const failure = new ClientResponseError({ status: 403, response: { message: "PocketBase access denied", code: "pb_denied" } });

        const response = handleError(failure);

        expect(response.status).toBe(403);
        expect(await response.json()).toMatchObject({ message: "PocketBase access denied", code: "pb_denied" });
    });

    it("keeps unexpected failures as server errors", () => {
        expect(handleError(new Error("connection failed")).status).toBe(500);
    });

    it("preserves SvelteKit HTTP errors", () => {
        let httpError: unknown;
        try {
            error(404, { message: "Actor not found" });
        } catch (caught) {
            httpError = caught;
        }

        expect(isHttpError(httpError, 404)).toBe(true);
        expect(() => handleError(httpError)).toThrow(httpError);
    });
});

describe("assertFileField", () => {
    const fileFields = ["gpx", "photos"] as const;

    it.each(["gpx", "photos", "photos+", "photos-"])("accepts a `%s` part", (name) => {
        const data = new FormData();
        data.append(name, new Blob(["x"]), "x.bin");

        expect(() => assertFileField(data, fileFields)).not.toThrow();
    });

    it("rejects a body with none of the file fields", () => {
        const data = new FormData();
        data.append("file", new Blob(["x"]), "track.gpx");
        data.append("name", "Renamed");

        let caught: unknown;
        try {
            assertFileField(data, fileFields);
        } catch (e) {
            caught = e;
        }

        expect(caught).toBeInstanceOf(ClientResponseError);
        expect((caught as ClientResponseError).status).toBe(400);
        expect((caught as ClientResponseError).response).toEqual({ message: "missing_file", expected: fileFields });
    });

    it("rejects an empty body", () => {
        expect(() => assertFileField(new FormData(), fileFields)).toThrow(ClientResponseError);
    });
});
