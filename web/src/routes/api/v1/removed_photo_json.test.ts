import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";
import { PUT as createTrail } from "./trail/+server";
import { POST as updateTrail } from "./trail/[id]/+server";
import { PUT as createWaypoint } from "./waypoint/+server";
import { POST as updateWaypoint } from "./waypoint/[id]/+server";
import { PUT as createSummitLog } from "./summit-log/+server";
import { POST as updateSummitLog } from "./summit-log/[id]/+server";

const routes = [
    { path: "trail", method: "PUT", handler: createTrail, code: "trail", body: { name: "Trail", description: "A route", author: "author000000001", public: false, completed: false, lat: 46, lon: 8, tags: [] } },
    { path: "trail/trail0000000001", method: "POST", handler: updateTrail, code: "trail", body: { name: "Changed trail", description: "Updated route", public: false, completed: false, lat: 46, lon: 8, tags: [] } },
    { path: "waypoint", method: "PUT", handler: createWaypoint, code: "waypoint", body: { name: "Waypoint", description: "", lat: 46, lon: 8, distance_from_start: null, icon: "circle", author: "author000000001", trail: "trail0000000001" } },
    { path: "waypoint/trail0000000001", method: "POST", handler: updateWaypoint, code: "waypoint", body: { name: "Changed waypoint", description: "", lat: 46, lon: 8, distance_from_start: null, icon: "circle", author: "author000000001", trail: "trail0000000001" } },
    { path: "summit-log", method: "PUT", handler: createSummitLog, code: "summit_log", body: { date: "2026-11-10", text: "Entry", author: "author000000001", trail: "trail0000000001", distance: 100, duration: 60 } },
    { path: "summit-log/trail0000000001", method: "POST", handler: updateSummitLog, code: "summit_log", body: { date: "2026-11-10", text: "Changed entry", author: "author000000001", trail: "trail0000000001", distance: 100, duration: 60 } },
];

describe.each(routes)("$method /api/v1/$path", ({ path, method, handler, code, body }) => {
    it("ignores a plain photos key and saves the record once", async () => {
        const photos = method === "PUT" ? [] : ["/api/v1/files/assets/photo0000000001/photo.jpg"];
        const record = {
            id: "trail0000000001", name: "Trail", date: "2026-11-10", photos,
            expand: { trail_assets_via_trail: photos.map(() => ({
                asset: "photo0000000001",
                expand: { asset: { id: "photo0000000001", collectionId: "assets", type: "photo", file: "photo.jpg" } },
            })) },
        };
        const create = vi.fn().mockResolvedValue(record);
        const update = vi.fn().mockResolvedValue(record);
        const collection = vi.fn(() => ({ create, update }));
        const url = new URL("https://a.example/api/v1/" + path);
        const event = {
            url,
            params: { id: record.id },
            request: new Request(url, {
                method,
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ ...body, photos }),
            }),
            locals: { pb: { collection } },
        } as unknown as RequestEvent;

        const response = await handler(event);

        expect(response.status).toBe(200);
        const write = method === "PUT" ? create : update;
        const otherWrite = method === "PUT" ? update : create;
        expect(write).toHaveBeenCalledOnce();
        expect(otherWrite).not.toHaveBeenCalled();
        const forwarded = write.mock.calls[0][method === "PUT" ? 0 : 1];
        expect(forwarded).not.toHaveProperty("photos");
        expect(forwarded).toHaveProperty("text" in body ? "text" : "name", "text" in body ? body.text : body.name);
        expect((await response.json()).photos).toEqual(photos);
    });

    it.each(["photos+", "+photos", "photos-", "-photos", "++photos--", "-+photos+-"])("rejects JSON %s before writing", async (key) => {
        const create = vi.fn().mockResolvedValue({ id: "trail0000000001", name: "Trail", date: "2026-11-10" });
        const update = vi.fn().mockResolvedValue({ id: "trail0000000001", name: "Trail", date: "2026-11-10" });
        const collection = vi.fn(() => ({ create, update }));
        const url = new URL("https://a.example/api/v1/" + path);
        const event = {
            url,
            params: { id: "trail0000000001" },
            request: new Request(url, {
                method,
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ ...body, [key]: ["photo.jpg"] }),
            }),
            locals: { pb: { collection } },
        } as unknown as RequestEvent;

        const response = await handler(event);

        expect(create, "legacy photo keys must be rejected before a create").not.toHaveBeenCalled();
        expect(update, "legacy photo keys must be rejected before an update").not.toHaveBeenCalled();
        expect(response.status).toBe(400);
        expect(await response.json()).toMatchObject({ message: `${code}_photos_form_field_removed` });
    });
});
