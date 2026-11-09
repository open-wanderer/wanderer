import { afterEach, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import { trails_update } from "./trail_store";

afterEach(() => vi.unstubAllGlobals());
it("returns a metadata-only form response without loading the trail or its GPX", async () => {
    const baseline = new Trail("Original", { id: "trail000000001", public: false, photos: [] });
    baseline.gpx = "route.gpx"; baseline.expand = { waypoints_via_trail: [], summit_logs_via_trail: [] };
    const updated = { ...baseline, name: "Renamed" };
    const response = { ...updated, description: "Saved form response" };
    const request = vi.fn(async (_input: RequestInfo | URL, _options?: RequestInit) => Response.json(response));
    vi.stubGlobal("fetch", request);
    await expect(trails_update(baseline, updated)).resolves.toEqual(response);
    expect(request).toHaveBeenCalledOnce();
    expect(String(request.mock.calls[0][0])).toContain("/api/v1/trail/form/");
    expect(request.mock.calls[0][1]?.method).toBe("POST");
});
