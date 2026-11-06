import type { RequestEvent } from "@sveltejs/kit";
import JSZip from "jszip";
import { describe, expect, it, vi } from "vitest";

vi.mock("#lib/stores/trail_store.ts", () => ({ trails_create: vi.fn() }));
vi.mock("#lib/stores/search_store.ts", () => ({ searchLocationReverse: vi.fn() }));

import { PUT } from "./+server";

function event(body: BodyInit, headers?: Record<string, string>) {
    return {
        request: new Request("http://localhost/api/v1/trail/upload", { method: "PUT", body, headers }),
        locals: { user: { id: "user" } },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
}

function upload(content: BlobPart) {
    const form = new FormData();
    form.set("file", new Blob([content]), "track.bin");
    return event(form);
}

describe("PUT /api/v1/trail/upload with an unusable file", () => {
    it("answers 413 for a file that is too large to parse", async () => {
        const zip = new JSZip();
        zip.file("doc.kml", " ".repeat(5 * 1024 * 1024));

        const response = await PUT(upload(await zip.generateAsync({ type: "uint8array", compression: "DEFLATE" }) as BlobPart));

        expect(response.status).toBe(413);
    });

    it.each<[string, () => RequestEvent]>([
        ["a corrupt zip", () => upload(new Uint8Array([0x50, 0x4b, 0x03, 0x04, 1, 2, 3]))],
        ["a broken multipart body", () => event("nonsense", { "content-type": "multipart/form-data; boundary=x" })],
        ["a missing file", () => event(new FormData())],
    ])("answers 400 for %s", async (_name, request) => {
        const log = vi.spyOn(console, "error").mockImplementation(() => {});
        try {
            const response = await PUT(request());

            expect(response.status).toBe(400);
            expect((await response.json()).message).toMatch(/^Invalid /);
        } finally {
            log.mockRestore();
        }
    });
});
