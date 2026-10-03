import { error, isHttpError } from "@sveltejs/kit";
import { getRequest } from "@sveltejs/kit/node";
import { IncomingMessage } from "node:http";
import { Socket } from "node:net";
import { describe, expect, it } from "vitest";
import { ClientResponseError } from "pocketbase";
import { assertFileField, handleError } from "./api_util";

describe("handleError", () => {
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

    it.each(["content-length", "chunked"])("preserves body size errors for %s uploads", async (framing) => {
        const incoming = new IncomingMessage(new Socket());
        incoming.method = "POST";
        incoming.url = "/api/v1/user/user00000000001/file";
        incoming.httpVersionMajor = 1;
        incoming.headers = {
            "content-type": "multipart/form-data; boundary=avatar",
            ...(framing === "content-length"
                ? { "content-length": String(2 * 1024 * 1024) }
                : { "transfer-encoding": "chunked" }),
        };

        try {
            const request = await getRequest({
                request: incoming,
                base: "http://localhost:3000",
                bodySizeLimit: 512 * 1024,
            });
            if (framing === "chunked") {
                incoming.push(Buffer.alloc(2 * 1024 * 1024));
                incoming.push(null);
            }
            const uploadError = await request.formData().catch((caught) => caught);

            expect(uploadError).toMatchObject({ status: 413 });
            expect(isHttpError(uploadError)).toBe(false);
            const response = handleError(uploadError);
            expect(response.status).toBe(413);
            expect(await response.json()).toEqual({ message: uploadError.message });
        } finally {
            incoming.destroy();
        }
    });

    it("keeps unexpected errors as server errors", async () => {
        const response = handleError(new Error("Unexpected failure"));

        expect(response.status).toBe(500);
        expect(await response.json()).toEqual({ message: "Unexpected failure" });
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
