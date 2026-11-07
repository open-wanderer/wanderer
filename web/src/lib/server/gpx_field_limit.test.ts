import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const env = vi.hoisted((): { BODY_SIZE_LIMIT?: string } => ({ BODY_SIZE_LIMIT: undefined }));
vi.mock("$app/env/private", () => env);
vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "http://backend.example" }));

const MiB = 1024 * 1024;
const GiB = 1024 * MiB;

function answer(maxSize: unknown) {
    return vi.fn(async () => Response.json({ maxSize }));
}

async function load() {
    vi.resetModules();
    const { gpxFieldLimit, trackBodyLimit } = await import("./gpx_field_limit.ts");
    const { limitRequestBodies } = await import("./request_body.ts");
    const { serverLimits } = await import("#lib/util/track_limits.ts");
    return { gpxFieldLimit, trackBodyLimit, limitRequestBodies, serverLimits };
}

describe("track limits from the gpx field", () => {
    beforeEach(() => { env.BODY_SIZE_LIMIT = undefined; });
    afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });

    it("keeps the measured limits at the field's default", async () => {
        vi.stubGlobal("fetch", answer(25 * MiB));
        const { gpxFieldLimit, serverLimits } = await load();

        expect(await gpxFieldLimit()).toBe(25 * MiB);
        expect(serverLimits()).toMatchObject({ fileBytes: 25 * MiB, kmlBytes: 4 * MiB, gpxPoints: 150_000, convertedPoints: 60_000 });
    });

    it("follows a field raised to 2 GiB", async () => {
        vi.stubGlobal("fetch", answer(2 * GiB));
        const { gpxFieldLimit, serverLimits } = await load();

        await gpxFieldLimit();

        expect(serverLimits().fileBytes).toBe(2 * GiB);
        expect(serverLimits().kmlBytes).toBe(4 * MiB * (2 * GiB) / (25 * MiB));
        expect(serverLimits().convertedPoints).toBe(60_000 * (2 * GiB) / (25 * MiB));
    });

    it("lowers the file limit without loosening the others", async () => {
        vi.stubGlobal("fetch", answer(MiB));
        const { gpxFieldLimit, serverLimits } = await load();

        await gpxFieldLimit();

        expect(serverLimits()).toMatchObject({ fileBytes: MiB, gpxPoints: 150_000 });
    });

    it("lets a request through the hook up to the field's size", async () => {
        vi.stubGlobal("fetch", answer(64 * MiB));
        const { limitRequestBodies } = await load();
        const request = (bytes: number) => new Request("https://local.example/api/v1/trail/convert", {
            method: "POST", body: "x", headers: { "content-length": String(bytes) },
        });
        const send = (bytes: number) => limitRequestBodies({
            event: { request: request(bytes), url: new URL(request(bytes).url), locals: {} } as never,
            resolve: async () => new Response("accepted"),
        });

        expect((await send(60 * MiB)).status).toBe(200);
        expect((await send(65 * MiB)).status).toBe(413);
    });

    it("stays within BODY_SIZE_LIMIT", async () => {
        env.BODY_SIZE_LIMIT = "10M";
        vi.stubGlobal("fetch", answer(2 * GiB));
        const { trackBodyLimit } = await load();

        expect(await trackBodyLimit()).toBe(10 * MiB);
    });

    it("asks the backend once a minute", async () => {
        vi.useFakeTimers();
        const fetch = answer(8 * MiB);
        vi.stubGlobal("fetch", fetch);
        const { gpxFieldLimit } = await load();

        await gpxFieldLimit();
        await gpxFieldLimit();
        expect(fetch).toHaveBeenCalledOnce();

        vi.advanceTimersByTime(61_000);
        await gpxFieldLimit();
        expect(fetch).toHaveBeenCalledTimes(2);
    });

    it.each([
        ["an unreachable backend", vi.fn(async () => { throw new TypeError("fetch failed"); })],
        ["an error status", vi.fn(async () => new Response("{}", { status: 500 }))],
        ["a value that is no size", answer("big")],
        ["a value below 1", answer(0)],
    ])("uses the default with %s", async (_name, fetch) => {
        vi.stubGlobal("fetch", fetch);
        const { gpxFieldLimit } = await load();

        expect(await gpxFieldLimit()).toBe(25 * MiB);
    });

    it("keeps the last known value while the backend cannot say", async () => {
        vi.useFakeTimers();
        vi.stubGlobal("fetch", answer(64 * MiB));
        const { gpxFieldLimit, serverLimits } = await load();
        await gpxFieldLimit();

        vi.stubGlobal("fetch", vi.fn(async () => { throw new TypeError("fetch failed"); }));
        vi.advanceTimersByTime(61_000);

        expect(await gpxFieldLimit()).toBe(64 * MiB);
        expect(serverLimits().fileBytes).toBe(64 * MiB);
    });
});
