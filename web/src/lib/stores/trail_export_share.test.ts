import { expect, it, vi } from "vitest";
import { trails_show } from "./trail_store.ts";

it.each(["1", "a".repeat(32)])("loads GPX files without forwarding the page share %s", async (share) => {
    const gpx = '<gpx version="1.1"><trk><trkseg><trkpt lat="47" lon="8"/></trkseg></trk></gpx>';
    const requests: URL[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "https://a.example");
        requests.push(url);
        if (url.pathname === "/api/v1/trail/trail0000000001") {
            expect(url.searchParams.get("share")).toBe(share);
            return Response.json({
                id: "trail0000000001", collectionId: "trailcollection", name: "Public hike", public: true, gpx: "trail.gpx",
                expand: { waypoints_via_trail: [], summit_logs_via_trail: [{
                    id: "entry0000000001", collectionId: "entrycollection", date: "2026-10-01", gpx: "entry.gpx",
                }] },
            });
        }
        expect(url.pathname).toMatch(/^\/api\/v1\/files\/(?:trailcollection\/trail0000000001\/trail|entrycollection\/entry0000000001\/entry)\.gpx$/);
        if (url.searchParams.has("share")) return Response.json({ message: "Invalid file share" }, { status: 400 });
        return new Response(gpx);
    });
    const trail = await trails_show("trail0000000001", "@walker", share, true, fetch);
    expect(trail.expand?.gpx_data).toBe(gpx);
    expect(trail.expand?.summit_logs_via_trail?.[0].expand?.gpx_data).toBe(gpx);
    expect(requests).toHaveLength(3);
    expect(requests.slice(1).every((url) => !url.searchParams.has("share"))).toBe(true);
});
