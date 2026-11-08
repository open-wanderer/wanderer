import { afterEach, expect, it, vi } from "vitest";
import GPX from "#lib/models/gpx/gpx.ts";
import { trail2gpx } from "./gpx_util";
import type { Trail } from "#lib/models/trail.ts";

afterEach(() => vi.unstubAllGlobals());
it.each(["", "track.gpx"])("loads omitted waypoints for a projected trail (gpx=%s)", async gpx => {
    const request = vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).includes("/files/")) return new Response('<gpx version="1.1" creator="test"><trk><trkseg/></trk></gpx>');
        return Response.json({ id: "trail", collectionId: "trails", name: "Hike", gpx, expand: {
        waypoints_via_trail: [{ name: "Summit", lat: 46, lon: 8, description: "View" }],
        ...(gpx ? { gpx_data: '<gpx version="1.1" creator="test"><trk><trkseg/></trk></gpx>' } : {}),
    } }); });
    vi.stubGlobal("fetch", request);
    const output = GPX.parse(await trail2gpx({ id: "trail", name: "Hike", gpx, expand: { author: { preferred_username: "alice" } } } as unknown as Trail)) as GPX;
    expect(output.wpt?.[0]).toMatchObject({ name: "Summit", desc: "View", $: { lat: 46, lon: 8 } });
    expect(request).toHaveBeenCalledTimes(gpx ? 2 : 1);
    const url = new URL(String(request.mock.calls[0][0]), "http://a.example");
    expect(url.pathname).toBe("/api/v1/trail/trail");
    expect(url.searchParams.get("expand")?.split(",")).toContain("waypoints_via_trail");
});
it("exports a proven empty trail without a GPX file or extra request", async () => {
    const request = vi.fn(); vi.stubGlobal("fetch", request);
    const output = GPX.parse(await trail2gpx({ id: "trail", name: "Empty", gpx: "", expand: { waypoints_via_trail: [] } } as unknown as Trail));
    expect(output).not.toBeInstanceOf(Error);
    expect((output as GPX).wpt ?? []).toEqual([]);
    expect(request).not.toHaveBeenCalled();
});
