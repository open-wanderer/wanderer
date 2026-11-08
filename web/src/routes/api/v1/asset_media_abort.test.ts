import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { GET as linkedPhoto } from "./assets/[id]/file/+server.ts";
import { responseOverHttp } from "#lib/server/response_wire_test_util.ts";
import { GET as storedPhoto } from "./files/[collection]/[record]/[file]/+server.ts";

describe.each([
    ["linked photo", linkedPhoto],
    ["stored photo", storedPhoto],
] as const)("%s cancellation", (name, handler) => {
    it("aborts the upstream fetch when the viewer leaves", async () => {
        const controller = new AbortController();
        const request = new Request("https://a.example/photo", { signal: controller.signal });
        const fetch = vi.fn((_url: string, options: RequestInit) => new Promise<Response>((_resolve, reject) => {
            const signal = options.signal;
            if (!signal) {
                reject(new Error("upstream fetch has no cancellation signal"));
                return;
            }
            signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true });
        }));
        const event = {
            params: { id: "hcykc33z7upq76u", collection: "assets", record: "hcykc33z7upq76u", file: "photo.jpg", plugin: "immich" },
            request,
            url: new URL(request.url),
            locals: { pb: { authStore: { token: "" }, buildURL: (path: string) => `https://b.example/${path}` } },
            fetch,
        } as unknown as RequestEvent;
        const pending = handler(event);
        await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
        expect(fetch.mock.calls[0][1].signal).toBe(request.signal);
        controller.abort();
        const response = await responseOverHttp(await pending);
        expect(response.status).toBe(499);
        expect(await response.text()).toBe("");
        expect(fetch.mock.calls[0][1].signal?.aborted).toBe(true);
    });

    it("preserves upstream failures when the viewer has not cancelled", async () => {
        const request = new Request("https://a.example/photo");
        const failure = new Error("provider unavailable");
        const event = {
            params: { id: "hcykc33z7upq76u", collection: "assets", record: "hcykc33z7upq76u", file: "photo.jpg", plugin: "immich" },
            request,
            url: new URL(request.url),
            locals: { pb: { authStore: { token: "" }, buildURL: (path: string) => `https://b.example/${path}` } },
            fetch: vi.fn().mockRejectedValue(failure),
        } as unknown as RequestEvent;

        if (name === "stored photo") {
            await expect(handler(event)).rejects.toMatchObject({ status: 500, body: { message: failure.message } });
        } else {
            await expect(handler(event)).rejects.toBe(failure);
        }
    });
});
