import { error, isHttpError, type RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";
import { Collection, handleError, update, uploadUpdate } from "./api_util";

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
});

describe.each(["JSON", "multipart"] as const)("trail %s update cancellation", (format) => {
    it("passes the request signal through the real PocketBase SDK", async () => {
        const controller = new AbortController();
        const fetch = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) =>
            new Response(JSON.stringify({ id: "trail0000000001", public: true }), {
                headers: { "Content-Type": "application/json" },
            }),
        );
        const event = trailUpdateEvent(format, controller.signal, fetch);

        await submitTrailUpdate(event, format);

        expect(fetch).toHaveBeenCalledOnce();
        const [url, options] = fetch.mock.calls[0];
        expect(String(url)).toContain("/api/collections/trails/records/trail0000000001");
        expect(String(url)).toContain("expand=category");
        expect(options?.method).toBe("PATCH");
        // Without requestKey:null the SDK replaces this signal with its own.
        expect(options?.signal).toBe(event.request.signal);
    });

    it("aborts the outgoing PocketBase update when the BFF request is canceled", async () => {
        const controller = new AbortController();
        const fetch = vi.fn((_url: RequestInfo | URL, options?: RequestInit) =>
            new Promise<Response>((_resolve, reject) => {
                const abort = () => reject(new DOMException("Aborted", "AbortError"));
                if (options?.signal?.aborted) {
                    abort();
                } else {
                    options?.signal?.addEventListener("abort", abort, { once: true });
                }
            }),
        );
        const event = trailUpdateEvent(format, controller.signal, fetch);
        const result = expect(submitTrailUpdate(event, format)).rejects.toMatchObject({ isAbort: true });
        await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());

        controller.abort();

        await result;
        expect(fetch.mock.calls[0][1]?.signal?.aborted).toBe(true);
    });
});

function trailUpdateEvent(format: "JSON" | "multipart", signal: AbortSignal, fetch: typeof globalThis.fetch): RequestEvent {
    const url = new URL("https://wanderer.example/api/v1/trail/trail0000000001?expand=category");
    const formData = new FormData();
    formData.set("public", "true");
    const request = new Request(url, {
        method: "POST",
        signal,
        ...(format === "multipart"
            ? { body: formData }
            : { body: JSON.stringify({ public: true }), headers: { "Content-Type": "application/json" } }),
    });
    return {
        url,
        request,
        params: { id: "trail0000000001" },
        locals: { pb: new PocketBase("https://pocketbase.example") },
        fetch,
    } as unknown as RequestEvent;
}

function submitTrailUpdate(event: RequestEvent, format: "JSON" | "multipart") {
    return format === "multipart"
        ? uploadUpdate(event, Collection.trails)
        : update(event, z.object({ public: z.boolean() }), Collection.trails);
}
