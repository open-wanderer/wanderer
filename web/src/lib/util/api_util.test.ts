import { error, isHttpError } from "@sveltejs/kit";
import { describe, expect, it } from "vitest";
import { ClientResponseError } from "pocketbase";
import { MeilisearchApiError } from "meilisearch";
import { assertFileField, handleError } from "./api_util";

describe("handleError", () => {
    it.each([400, 403, 503])("preserves Meilisearch status %i and error details", async (status) => {
        const detail = { message: "Search failed", code: "invalid_search_filter", type: "invalid_request", link: "https://example.invalid/error" };
        const failure = new MeilisearchApiError(new Response(null, { status }), detail);

        const response = handleError(failure);

        expect(response.status).toBe(status);
        expect(await response.json()).toEqual(detail);
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
