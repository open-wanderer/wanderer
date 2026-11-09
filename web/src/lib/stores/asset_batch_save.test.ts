import { afterEach, describe, expect, it, vi } from "vitest";
import type { Asset } from "#lib/models/asset.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { Trail } from "#lib/models/trail.ts";
import { Waypoint } from "#lib/models/waypoint.ts";
import { APIError } from "#lib/util/api_util.ts";
import { assets_attach_to_target, assets_create } from "./asset_store";
import { summit_logs_create, summit_logs_update } from "./summit_log_store";
import { trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";
import { waypoints_create, waypoints_update } from "./waypoint_store";

const user = { id: "user", actor: "actor", collectionId: "users", collectionName: "users", password: "unused" };
const photo = (id: string) => `/api/v1/assets/${id}/file`;
const asset = (name: string) => ({ id: `photo${name.match(/\d+/)![0].padStart(10, "0")}` } as Asset);
function selection(count = 53): File[] {
    return Array.from({ length: count }, (_, index) => {
        const file = new File(["photo"], `photo${index}.jpg`, { type: "image/jpeg" });
        // Exercise realistic aggregate sizes without allocating a GiB of bytes.
        Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 });
        return file;
    });
}
function trail(photos: string[] = []): Trail {
    return Object.assign(new Trail("Outing", { id: "trail", public: false, photos }), { author: user.actor, expand: {} });
}

function server(failure: "none" | "response" | "network" = "response", partial = 0, failAt = 2) {
    const stored = new Map<string, string[]>();
    const uploaded: string[] = [];
    const batches: string[][] = [];
    const covers: string[] = [];
    const creates = { trail: 0, waypoint: 0, summit_log: 0 };
    let failed = false;
    const photos = (target: string) => [...(stored.get(target) ?? [])];
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const url = new URL(String(input), "https://a.example");
        const path = url.pathname;
        if (path.startsWith("/api/v1/federation/")) return Response.json({ status: "ok" });
        if (path === "/api/v1/assets") {
            const form = options!.body as FormData;
            const target = ["summit_log", "waypoint", "trail"].find(key => form.has(key))!;
            const files = form.getAll("files") as File[];
            batches.push(files.map(file => file.name));
            const fail = failure !== "none" && !failed && batches.length === failAt;
            if (fail) failed = true;
            if (fail && failure === "network") throw new TypeError("Connection interrupted");
            const saved = (fail ? files.slice(0, partial) : files).map(file => {
                uploaded.push(file.name);
                return asset(file.name);
            });
            stored.set(target, [...photos(target), ...saved.map(record => photo(record.id))]);
            return fail ? Response.json({ message: "Photo save interrupted", detail: {
                uploadedAssets: saved, attachedAssets: saved, completedFileCount: saved.length,
            } }, { status: 503 }) : Response.json(saved);
        }
        if (path.endsWith("/thumbnail")) {
            covers.push(JSON.parse(String(options!.body)).asset);
            return Response.json({ acknowledged: true });
        }
        if (path.startsWith("/api/v1/trail/form")) {
            if (options?.method === "PUT") creates.trail++;
            return Response.json(trail(photos("trail")));
        }
        if (path === "/api/v1/trail/trail") return Response.json(trail(photos("trail")));
        if (path.startsWith("/api/v1/waypoint")) {
            if (options?.method === "PUT") creates.waypoint++;
            return Response.json({ id: "waypoint", lat: 47, lon: 8, trail: "trail", author: user.actor, photos: photos("waypoint"),
                expand: { waypoint_assets_via_waypoint: photos("waypoint").map(value => ({ asset: value.split("/").at(-2), expand: { asset: { id: value.split("/").at(-2), type: "photo", author: user.actor, storage_mode: "link_private" } } })) } });
        }
        if (path.startsWith("/api/v1/summit-log/form")) {
            if (options?.method === "PUT") creates.summit_log++;
            return Response.json({ id: "entry", date: "2026-10-01", trail: "trail", author: user.actor, photos: photos("summit_log") });
        }
        throw new Error(`Unexpected ${options?.method ?? "GET"} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    currentUser.set(user);
    return { request, photos, uploaded, batches, covers, creates };
}

afterEach(() => { vi.unstubAllGlobals(); currentUser.set(null); });

describe("photo upload batches", () => {
    it("sends a selection above one GiB as sequential requests in selection order", async () => {
        const remote = server("none");
        const files = selection();
        let release!: () => void;
        const firstResponse = new Promise<void>(resolve => { release = resolve; });
        const gatedRequest = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
            if (gatedRequest.mock.calls.length === 1) await firstResponse;
            return remote.request(input, options);
        });
        const saving = assets_create(files, { trail: "trail" }, gatedRequest);
        await Promise.resolve();
        expect(gatedRequest).toHaveBeenCalledTimes(1);
        release();
        const saved = await saving;
        expect(remote.batches.map(batch => batch.length)).toEqual([12, 12, 12, 12, 5]);
        expect(remote.uploaded).toEqual(files.map(file => file.name));
        expect(saved.map(record => record.id)).toEqual(files.map(file => asset(file.name).id));
        expect(remote.request.mock.calls.every(([, options]) => (options!.body as FormData).get("trail") === "trail")).toBe(true);
    });

    it.each([["response", 2], ["response", 3], ["network", 2], ["network", 3]] as const)("reports prior successful batches after a %s failure in request %s", async (failure, failAt) => {
        const remote = server(failure, 3, failAt);
        const files = selection();
        const error = await assets_attach_to_target({ files, target: { trail: "trail" }, f: remote.request }).catch(error => error);
        const count = (failAt - 1) * 12 + (failure === "response" ? 3 : 0);
        expect(error).toBeInstanceOf(APIError);
        expect(error.detail.completedAttachments).toMatchObject({ files: false, uploadedFiles: count });
        expect(error.detail.attachedPhotos).toEqual(files.slice(0, count).map(file => photo(asset(file.name).id)));
        const result = await assets_attach_to_target({ files: files.slice(count), existingPhotos: error.detail.attachedPhotos, target: { trail: "trail" }, f: remote.request });
        expect(remote.uploaded).toEqual(files.map(file => file.name));
        expect(new Set(result).size).toBe(files.length);
    });
});

describe("large photo selection save retries", () => {
    for (const mode of ["create", "update"] as const) {
        it.each([5, 26])(`keeps the trail cover and order across a ${mode} retry (cover %s)`, async cover => {
            const remote = server("response", 3);
            const files = selection();
            const originalNames = files.map(file => file.name);
            const expected = originalNames.map(name => photo(asset(name).id));
            const pending = trail();
            pending.thumbnail = cover;
            let baseline = trail();
            const error = await (mode === "create"
                ? trails_create(pending, files, null, remote.request, user)
                : trails_update(baseline, pending, files)).catch(error => error);
            expect(error).toBeInstanceOf(APIError);
            if (mode === "create") baseline = error.detail.savedTrail;
            expect(pending.photos).toEqual(expected.slice(0, 15));
            expect(files.map(file => file.name)).toEqual(originalNames.slice(15));
            const result = await trails_update(baseline, pending, files);
            expect(result.photos).toEqual(expected);
            expect(remote.uploaded).toEqual(originalNames);
            expect(remote.covers.at(-1)).toBe(asset(originalNames[cover]).id);
            expect(files).toHaveLength(0);
            expect(remote.creates.trail).toBe(mode === "create" ? 1 : 0);
        });

        it(`keeps completed waypoint photos after a ${mode} batch failure`, async () => {
            const remote = server("response", 3);
            const pending = new Waypoint(47, 8, { id: mode === "update" ? "waypoint" : undefined, trail: "trail", photos: [] });
            const files = selection();
            pending._photos = files;
            const error = await (mode === "create"
                ? waypoints_create(pending, remote.request, user)
                : waypoints_update({ ...pending, _photos: undefined }, pending)).catch(error => error);
            expect(error).toBeInstanceOf(APIError);
            expect(error.detail.savedWaypoint.photos).toHaveLength(15);
            expect(error.detail.remainingAttachments._photos.map((file: File) => file.name)).toEqual(files.slice(15).map(file => file.name));
            const baseline = error.detail.savedWaypoint;
            Object.assign(pending, baseline, error.detail.remainingAttachments);
            const result = await waypoints_update(baseline, pending);
            expect(result.photos).toEqual(files.map(file => photo(asset(file.name).id)));
            expect(remote.uploaded).toEqual(files.map(file => file.name));
            expect(remote.creates.waypoint).toBe(mode === "create" ? 1 : 0);
        });

        it(`keeps completed summit-log photos after a ${mode} batch failure`, async () => {
            const remote = server("response", 3);
            const files = selection();
            const pending: SummitLog = { id: mode === "update" ? "entry" : undefined, trail: "trail", author: user.actor, date: "2026-10-01", photos: [], _photos: files };
            const error = await (mode === "create"
                ? summit_logs_create(pending, remote.request, user)
                : summit_logs_update({ ...pending, _photos: undefined }, pending)).catch(error => error);
            expect(error).toBeInstanceOf(APIError);
            expect(pending.photos).toHaveLength(15);
            expect(pending._photos?.map(file => file.name)).toEqual(files.slice(15).map(file => file.name));
            const result = await summit_logs_update(error.detail.savedSummitLog, pending);
            expect(result.photos).toEqual(files.map(file => photo(asset(file.name).id)));
            expect(remote.uploaded).toEqual(files.map(file => file.name));
            expect(remote.creates.summit_log).toBe(mode === "create" ? 1 : 0);
        });
    }
});
