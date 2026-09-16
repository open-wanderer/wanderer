import { afterEach, describe, expect, it, vi } from "vitest";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import { summit_logs_create, summit_logs_update } from "./summit_log_store";
import { trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";

const user = { id: "user", actor: "actor", collectionId: "users", collectionName: "users", password: "unused" };
const gpx = '<gpx version="1.1"><trk><name>Saved outing</name></trk></gpx>';
const photoURL = (id: string) => `/api/v1/assets/${id}/file`;
function log(withLocalPhoto = true): SummitLog {
    return {
        date: "2026-09-13", text: "Keep my outing", author: "actor", trail: "trail-id", photos: [],
        _gpx: new Blob([gpx], { type: "application/gpx+xml" }), expand: { gpx_data: gpx },
        _photos: withLocalPhoto ? [new File(["image"], "outing.jpg")] : undefined,
        _assetPluginLinks: [{ pluginId: "immich", assetIds: ["remote-photo"] }],
    };
}
function trail(logs: SummitLog[] = []): Trail {
    return {
        id: "trail-id", name: "Trail", author: "actor", public: false, completed: true,
        like_count: 0, tags: [], photos: [], expand: { summit_logs_via_trail: logs, waypoints_via_trail: [] },
    };
}

function server(options: { failPluginAt?: number; failDeleteAt?: number } = {}) {
    const logs = new Map<string, SummitLog>();
    const counts = { creates: 0, updates: 0, gpx: 0, uploads: 0, plugins: 0, deletes: 0, trailCreates: 0 };
    const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
        const path = String(url);
        if (path.startsWith("/api/v1/summit-log/form")) {
            const form = config!.body as FormData;
            const isCreate = config!.method === "PUT";
            if (isCreate) counts.creates++;
            else counts.updates++;
            const id = isCreate
                ? (form.get("id")?.toString() || `summit${String(counts.creates).padStart(9, "0")}`)
                : path.split("?")[0].split("/").at(-1)!;
            const previous = logs.get(id);
            const saved: SummitLog = {
                ...previous, id, date: form.get("date")!.toString(), text: form.get("text")?.toString(),
                author: "actor", trail: "trail-id", photos: [...(previous?.photos ?? [])],
            };
            if (form.has("gpx")) {
                counts.gpx++;
                saved.gpx = "route.gpx";
            }
            logs.set(id, saved);
            return Response.json({ ...saved, collectionId: "summit_logs" });
        }
        if (path === "/api/v1/assets" && config?.method === "PUT") {
            const form = config.body as FormData;
            const target = logs.get(form.get("summit_log")!.toString())!;
            counts.uploads++;
            const id = `photo${String(counts.uploads).padStart(10, "0")}`;
            target.photos.push(photoURL(id));
            return Response.json([{ id }]);
        }
        if (path.includes("/import-to-target")) {
            const { summitLogId } = JSON.parse(String(config?.body));
            counts.plugins++;
            if (counts.plugins === options.failPluginAt) {
                return Response.json({ imported: [], omitted: [{ assetId: "remote-photo", reason: "download_failed" }] });
            }
            const id = `remote${String(counts.plugins).padStart(9, "0")}`;
            logs.get(summitLogId)!.photos.push(photoURL(id));
            return Response.json({ imported: [{ asset: { id } }], omitted: [] });
        }
        if (path.startsWith("/api/v1/assets/") && config?.method === "DELETE") {
            counts.deletes++;
            if (counts.deletes === options.failDeleteAt) return Response.json({ message: "delete-failed" }, { status: 503 });
            const parsed = new URL(path, "http://wanderer.test");
            const id = parsed.pathname.split("/").at(-1)!;
            const target = logs.get(parsed.searchParams.get("summit_log")!)!;
            target.photos = target.photos.filter((photo) => photo !== photoURL(id));
            return Response.json({ acknowledged: true });
        }
        if (path.startsWith("/api/v1/files/")) return new Response(gpx);
        if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
        if (path.startsWith("/api/v1/trail/")) {
            if (config?.method === "PUT") counts.trailCreates++;
            return Response.json(trail([...logs.values()].map((saved) => ({ ...saved, photos: [...saved.photos], collectionId: "summit_logs" }))));
        }
        throw new Error(`Unexpected request ${config?.method ?? "GET"} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    currentUser.set(user);
    return { request, counts, logs };
}

afterEach(() => { vi.unstubAllGlobals(); currentUser.set(null); });

describe("summit log partial saves", () => {
    it.each([true, false])("keeps a created entry and its GPX after photo failure (local photo: %s)", async (withLocal) => {
        const remote = server({ failPluginAt: 1 });
        const pending = log(withLocal);
        const error = await summit_logs_create(pending, remote.request, user).catch((error: unknown) => error);
        expect(error).toBeInstanceOf(APIError);
        const saved = (error as APIError).detail.savedSummitLog as SummitLog;
        expect(pending).toMatchObject({ id: saved.id, text: "Keep my outing", date: "2026-09-13", gpx: "route.gpx" });
        expect(pending.expand?.gpx_data).toBe(gpx);
        expect(pending._gpx).toBeUndefined();
        expect(pending._photos).toBeUndefined();
        expect(pending._assetPluginLinks).toHaveLength(1);
        expect(remote.logs.size).toBe(1);
        expect(remote.counts.deletes).toBe(0);

        const result = await summit_logs_update(saved, pending);
        expect(result.id).toBe(saved.id);
        expect(result.photos).toHaveLength(withLocal ? 2 : 1);
        expect(remote.counts).toMatchObject({ creates: 1, updates: 1, gpx: 1, uploads: withLocal ? 1 : 0, plugins: 2 });
    });

    it.each(["create", "update"] as const)("reuses the partly saved log during a trail %s retry", async (mode) => {
        const remote = server({ failPluginAt: 1 });
        const pendingLog = log();
        const pendingTrail = trail([pendingLog]);
        let baseline = trail();
        const error = await (mode === "create"
            ? trails_create(pendingTrail, [], null, remote.request, user)
            : trails_update(baseline, pendingTrail)).catch((error: unknown) => error);
        expect(error).toBeInstanceOf(APIError);
        if (mode === "create") baseline = (error as APIError).detail.savedTrail;
        expect(baseline.expand!.summit_logs_via_trail).toHaveLength(1);
        expect(baseline.expand!.summit_logs_via_trail![0].id).toBe(pendingLog.id);
        expect(pendingTrail.photos).toEqual([]);

        const saved = await trails_update(baseline, pendingTrail);
        expect(saved.expand!.summit_logs_via_trail).toHaveLength(1);
        expect(remote.counts).toMatchObject({ creates: 1, updates: 1, gpx: 1, uploads: 1, plugins: 2, trailCreates: mode === "create" ? 1 : 0 });
    });

    it("does not recreate a successful earlier log when a later log fails", async () => {
        const remote = server({ failPluginAt: 2 });
        const first = log();
        const second = { ...log(), text: "Second outing" };
        const baseline = trail();
        const pending = trail([first, second]);
        await expect(trails_update(baseline, pending)).rejects.toMatchObject({ message: "asset_import_failed" });
        expect(baseline.expand!.summit_logs_via_trail).toHaveLength(2);
        expect(first._assetPluginLinks).toBeUndefined();
        expect(second._assetPluginLinks).toHaveLength(1);

        await trails_update(baseline, pending);
        expect(remote.logs.size).toBe(2);
        expect(remote.counts).toMatchObject({ creates: 2, updates: 1, gpx: 2, uploads: 2, plugins: 3 });
    });

    it.each(["import", "delete"] as const)("keeps update content, photo removal intent and completed uploads after %s failure", async (failure) => {
        const remote = server(failure === "import" ? { failPluginAt: 1 } : { failDeleteAt: 1 });
        const old: SummitLog = { id: "existing0000001", date: "2026-09-01", text: "Original", author: "actor", trail: "trail-id", photos: [photoURL("oldphoto0000001")], gpx: "old.gpx" };
        remote.logs.set(old.id!, { ...old, photos: [...old.photos] });
        const pendingLog = { ...log(), id: old.id };
        const baseline = trail([old]);
        const pending = trail([pendingLog]);
        await expect(trails_update(baseline, pending)).rejects.toBeInstanceOf(APIError);
        expect(pendingLog).toMatchObject({ id: old.id, text: "Keep my outing", date: "2026-09-13", gpx: "route.gpx" });
        expect(pendingLog.photos).not.toContain(photoURL("oldphoto0000001"));
        expect(pendingLog._gpx).toBeUndefined();
        expect(pendingLog._photos).toBeUndefined();

        await trails_update(baseline, pending);
        expect(remote.logs.size).toBe(1);
        expect(remote.logs.get(old.id!)!.photos).toHaveLength(2);
        expect(remote.counts).toMatchObject({ creates: 0, updates: 2, gpx: 1, uploads: 1, plugins: failure === "import" ? 2 : 1, deletes: failure === "import" ? 1 : 2 });
    });
});
