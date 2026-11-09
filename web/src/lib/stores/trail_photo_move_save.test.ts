import { afterEach, expect, it, vi } from "vitest";
import type { Trail } from "#lib/models/trail.ts";
import { trails_update } from "./trail_store";
import { currentUser } from "./user_store";

const assetId = "movedphoto00001";
const photo = `/api/v1/files/assets/${assetId}/native.jpg`;
const owner = { id: "user", actor: "actor", collectionId: "users", collectionName: "users" };
afterEach(() => { vi.unstubAllGlobals(); currentUser.set(null); });

function server(source: string, failTarget = false, failPhase?: "trail-save" | "source-remove") {
    const links = new Map<string, Set<string>>([[source, new Set([assetId])]]);
    const files = new Map([[assetId, new Uint8Array([1, 2, 3, 4])]]);
    const waypoints = new Map(["a", "b"].map(id => [id, { id, trail: "trail", author: "actor", name: id, lat: 46, lon: 8 }]));
    const logs = new Map<string, Record<string, unknown>>();
    if (source === "summit_log:log") logs.set("log", { id: "log", trail: "trail", author: "actor", date: "2026-10-14" });
    let failed = false;
    const photos = (target: string) => [...(links.get(target) ?? [])].map(() => photo);
    const snapshot = (): Trail => ({ id: "trail", name: "Trail", author: "actor", public: false, completed: true, like_count: 0, tags: [], photos: photos("trail:trail"), expand: {
        waypoints_via_trail: [...waypoints.values()].map(w => ({ ...w, photos: photos(`waypoint:${w.id}`) })),
        summit_logs_via_trail: [...logs.values()].map(log => ({ ...log, photos: photos(`summit_log:${log.id}`) })) as never,
    } });
    const collect = () => { if (![...links.values()].some(ids => ids.has(assetId))) files.delete(assetId); };
    const request = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = new URL(String(input), "https://a.example"); const path = url.pathname; const method = init?.method ?? "GET";
        if (path.includes("/federation/")) return Response.json({ status: "ok" });
        if (path === "/api/v1/assets" && method === "PUT") {
            const body = init!.body as FormData;
            const kind = ["waypoint", "summit_log", "trail"].find(kind => body.has(kind))!;
            const target = `${kind}:${body.get(kind)}`;
            if (failTarget && !failed) { failed = true; return Response.json({ message: "link failed" }, { status: 500 }); }
            if (!files.has(assetId)) return Response.json({ message: "photo no longer exists" }, { status: 404 });
            links.set(target, new Set([...(links.get(target) ?? []), ...body.getAll("assetIds").map(String)]));
            return Response.json([{ id: assetId, collectionId: "assets", type: "photo", file: "native.jpg", author: "actor", storage_mode: "copy" }]);
        }
        if (failPhase === "trail-save" && !failed && path === "/api/v1/trail/form/trail" && method === "POST") {
            failed = true; return Response.json({ message: "save failed" }, { status: 500 });
        }
        if (path === `/api/v1/assets/${assetId}` && method === "DELETE") {
            if (failPhase === "source-remove" && !failed) {
                failed = true; return Response.json({ message: "unlink failed" }, { status: 500 });
            }
            const kind = ["waypoint", "summit_log", "trail"].find(kind => url.searchParams.has(kind))!;
            links.get(`${kind}:${url.searchParams.get(kind)}`)?.delete(assetId); collect();
            return Response.json({ acknowledged: true, assetDeleted: !files.has(assetId) });
        }
        if (path.startsWith("/api/v1/waypoint/")) {
            const id = path.split("/").at(-1)!;
            if (method === "DELETE") { waypoints.delete(id); links.delete(`waypoint:${id}`); collect(); return Response.json({ ok: true }); }
            if (method === "POST") waypoints.set(id, { ...waypoints.get(id), ...JSON.parse(String(init?.body)) });
            return Response.json({ ...waypoints.get(id), photos: photos(`waypoint:${id}`) });
        }
        if (path === "/api/v1/summit-log/form" && method === "PUT") {
            const form = init!.body as FormData; const log = { id: "created-log", trail: "trail", author: "actor", date: String(form.get("date")) };
            logs.set(log.id, log); return Response.json({ ...log, photos: [] });
        }
        if (path.startsWith("/api/v1/summit-log/form/")) {
            const id = path.split("/").at(-1)!; return Response.json({ ...logs.get(id), photos: photos(`summit_log:${id}`) });
        }
        if (path.startsWith("/api/v1/summit-log/") && method === "DELETE") {
            const id = path.split("/").at(-1)!; logs.delete(id); links.delete(`summit_log:${id}`); collect(); return Response.json({ ok: true });
        }
        if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
        if (path.startsWith("/api/v1/trail/")) return Response.json(snapshot());
        throw new Error(`Unexpected ${method} ${path}`);
    });
    currentUser.set(owner as never); vi.stubGlobal("fetch", request);
    return { snapshot, files, links, logs, request };
}

it.each(["waypoint-gallery", "deleted-waypoint-gallery", "waypoint-later-waypoint", "waypoint-summit-log", "summit-log-gallery", "gallery-waypoint", "later-earlier-waypoint"])("keeps the stored asset and its bytes when moving %s in one save", async flow => {
    const source = flow === "summit-log-gallery" ? "summit_log:log" : flow === "gallery-waypoint" ? "trail:trail" : flow === "later-earlier-waypoint" ? "waypoint:b" : "waypoint:a";
    const db = server(source); const before = db.snapshot(); const pending = structuredClone(before);
    const waypoints = pending.expand!.waypoints_via_trail!;
    if (source.startsWith("waypoint:")) {
        const id = source.split(":")[1];
        if (flow === "deleted-waypoint-gallery") pending.expand!.waypoints_via_trail = waypoints.filter(w => w.id !== id);
        else waypoints.find(w => w.id === id)!.photos = [];
    } else if (source.startsWith("summit_log:")) pending.expand!.summit_logs_via_trail![0].photos = [];
    else pending.photos = [];
    let destination: string;
    if (flow.endsWith("gallery")) { pending.photos = [photo]; pending._assetLinks = [assetId]; destination = "trail:trail"; }
    else if (flow === "waypoint-summit-log") {
        pending.expand!.summit_logs_via_trail = [{ date: "2026-10-14", author: "actor", photos: [photo], _assetLinks: [assetId] }]; destination = "summit_log:created-log";
    } else {
        const id = flow === "later-earlier-waypoint" ? "a" : "b";
        Object.assign(waypoints.find(w => w.id === id)!, { photos: [photo], _assetLinks: [assetId] }); destination = `waypoint:${id}`;
    }
    await trails_update(before, pending);
    expect(db.files.get(assetId)).toEqual(new Uint8Array([1, 2, 3, 4]));
    expect(db.links.get(source)?.has(assetId) ?? false).toBe(false);
    expect(db.links.get(destination)?.has(assetId)).toBe(true);
});

it("retains source links on a failed destination and retries without losing the asset", async () => {
    const db = server("waypoint:a", true); const before = db.snapshot(); const pending = structuredClone(before);
    pending.expand!.waypoints_via_trail![0].photos = [];
    pending.photos = [photo]; pending._assetLinks = [assetId];
    await expect(trails_update(before, pending)).rejects.toThrow();
    expect(db.files.has(assetId)).toBe(true); expect(db.links.get("waypoint:a")?.has(assetId)).toBe(true);
    await trails_update(before, pending);
    expect(db.files.has(assetId)).toBe(true); expect(db.links.get("waypoint:a")?.has(assetId)).toBe(false); expect(db.links.get("trail:trail")?.has(assetId)).toBe(true);
});


it.each(["trail-save", "source-remove"] as const)("retains a completed destination link and retries a later %s failure", async phase => {
    const db = server("waypoint:a", false, phase); const before = db.snapshot(); const pending = structuredClone(before);
    pending.expand!.waypoints_via_trail![0].photos = [];
    Object.assign(pending.expand!.waypoints_via_trail![1], { photos: [photo], _assetLinks: [assetId] });
    await expect(trails_update(before, pending)).rejects.toThrow();
    expect(db.files.get(assetId)).toEqual(new Uint8Array([1, 2, 3, 4]));
    expect(db.links.get("waypoint:a")?.has(assetId)).toBe(true);
    expect(db.links.get("waypoint:b")?.has(assetId)).toBe(true);
    expect(before.expand!.waypoints_via_trail![0].photos).toContain(photo);
    await trails_update(before, pending);
    expect(db.files.get(assetId)).toEqual(new Uint8Array([1, 2, 3, 4]));
    expect(db.links.get("waypoint:a")?.has(assetId)).toBe(false);
    expect(db.links.get("waypoint:b")?.has(assetId)).toBe(true);
});
