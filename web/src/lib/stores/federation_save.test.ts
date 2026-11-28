import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SummitLog } from "#lib/models/summit_log.ts";
import type { Trail } from "#lib/models/trail.ts";
import { APIError } from "#lib/util/api_util.ts";
import { federationSaveRenewalInterval } from "#lib/util/federation_save_util.ts";
import { assets_attach_to_target, assets_delete_removed, assets_set_trail_thumbnail, has_asset_attachments } from "./asset_store";
import { summit_logs_create } from "./summit_log_store";
import { trails_update } from "./trail_store";
import { currentUser } from "./user_store";

vi.mock("./asset_store", () => ({
    assets_attach_to_target: vi.fn(), assets_delete_removed: vi.fn(),
    assets_set_trail_thumbnail: vi.fn(), has_asset_attachments: vi.fn(),
}));

const user = { id: "local-user", actor: "actor", collectionId: "users", collectionName: "users", password: "unused" };
const photo = "/api/v1/assets/photorecord00001/file";
const trail = (): Trail => ({
    id: "trailrecord0001", name: "Public trail", author: "actor", public: true,
    completed: true, like_count: 0, tags: [], photos: [],
    expand: { waypoints_via_trail: [], summit_logs_via_trail: [] },
});

beforeEach(() => {
    vi.resetAllMocks();
    currentUser.set(user);
    vi.mocked(assets_delete_removed).mockResolvedValue(undefined);
    vi.mocked(assets_set_trail_thumbnail).mockResolvedValue(undefined);
    vi.mocked(has_asset_attachments).mockReturnValue(true);
});
afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    currentUser.set(null);
});

describe("federation save completion in the editor stores", () => {
    it("holds a new summit-log Create through a slow photo upload", async () => {
        vi.useFakeTimers();
        const actions: string[] = [];
        const batchIds: string[] = [];
        let createBatch = "";
        const saved: SummitLog = { id: "summitrecord001", author: "actor", trail: "trailrecord0001", date: "2026-10-12", photos: [] };
        const request = vi.fn(async (url: RequestInfo | URL, options?: RequestInit) => {
            const path = new URL(String(url), "https://a.example");
            if (path.pathname === "/api/v1/summit-log/form") {
                actions.push("create");
                createBatch = path.searchParams.get("federationBatch")!;
                return Response.json(saved);
            }
            actions.push(path.pathname.split("/").at(-1)!);
            batchIds.push(JSON.parse(String(options?.body)).batchId);
            return Response.json({ status: "ok" });
        });
        let finish!: (photos: string[]) => void;
        const upload = new Promise<string[]>((resolve) => { finish = resolve; });
        vi.mocked(assets_attach_to_target).mockImplementation(async () => { actions.push("upload"); return upload; });
        const pending: SummitLog = { ...saved, id: undefined, _photos: [new File(["photo"], "photo.jpg")] };
        const saving = summit_logs_create(pending, request, user);
        await vi.advanceTimersByTimeAsync(2 * federationSaveRenewalInterval);
        expect(actions).toEqual(["create", "upload", "begin", "begin"]);
        expect(createBatch).toMatch(/^[a-f0-9]{32}$/);
        expect(batchIds).toEqual([createBatch, createBatch]);
        finish([photo]);
        const result = await saving;
        expect(result.photos).toEqual([photo]);
        expect(actions.at(-1)).toBe("complete");
        expect(batchIds.at(-1)).toBe(createBatch);
    });

    it("begins a public-trail save before metadata and completes after all asset work", async () => {
        const actions: string[] = [];
        const baseline = trail();
        const pending = trail();
        const request = vi.fn(async (url: RequestInfo | URL) => {
            const path = String(url);
            if (path.startsWith("/api/v1/federation/")) {
                actions.push(path.split("/").at(-1)!);
                return Response.json({ status: "ok" });
            }
            actions.push(path.includes("/form/") ? "metadata" : "refresh");
            return Response.json({ ...trail(), photos: [photo] });
        });
        vi.stubGlobal("fetch", request);
        vi.mocked(assets_attach_to_target).mockImplementation(async () => { actions.push("upload"); return [photo]; });
        vi.mocked(assets_delete_removed).mockImplementation(async () => { actions.push("remove"); });
        vi.mocked(assets_set_trail_thumbnail).mockImplementation(async () => { actions.push("thumbnail"); });
        await trails_update(baseline, pending, [new File(["photo"], "photo.jpg")]);
        expect(actions).toEqual(["begin", "metadata", "upload", "remove", "refresh", "thumbnail", "refresh", "complete"]);
    });

    it("completes a public-trail save after partial photo failure", async () => {
        const actions: string[] = [];
        vi.stubGlobal("fetch", vi.fn(async (url: RequestInfo | URL) => {
            const path = String(url);
            actions.push(path.startsWith("/api/v1/federation/") ? path.split("/").at(-1)! : "metadata");
            return Response.json(path.includes("/federation/") ? { status: "ok" } : trail());
        }));
        const failure = new APIError(500, "asset_import_failed");
        vi.mocked(assets_attach_to_target).mockRejectedValue(failure);
        await expect(trails_update(trail(), trail(), [new File(["photo"], "photo.jpg")])).rejects.toBe(failure);
        expect(actions).toEqual(["begin", "metadata", "complete"]);
    });
});


it("holds a newly public trail Create through a slow photo upload", async () => {
    vi.useFakeTimers();
    const actions: string[] = [];
    const activities: string[][] = [];
    let leased = false;
    let pendingCreate = false;
    let photos: string[] = [];
    const baseline = { ...trail(), public: false };
    const request = vi.fn(async (input: RequestInfo | URL) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) {
            const action = path.split("/").at(-1)!;
            actions.push(action);
            leased = action === "begin";
            if (!leased && pendingCreate) activities.push([...photos]);
            return Response.json({ status: "ok" });
        }
        if (path.includes("/form/")) {
            actions.push("metadata");
            setTimeout(() => {
                if (leased) pendingCreate = true;
                else activities.push([...photos]);
            }, 10_000);
        } else actions.push("refresh");
        return Response.json({ ...trail(), photos: [...photos] });
    });
    vi.stubGlobal("fetch", request);
    let finish!: () => void;
    const uploaded = new Promise<void>(resolve => { finish = resolve; });
    vi.mocked(assets_attach_to_target).mockImplementation(async () => {
        actions.push("upload");
        await uploaded;
        photos = [photo];
        return [...photos];
    });
    const saving = trails_update(baseline, trail(), [new File(["photo"], "slow.jpg")]);
    await vi.advanceTimersByTimeAsync(12_000);
    expect(activities).toEqual([]);
    expect(actions).toEqual(["begin", "metadata", "upload"]);
    finish();
    await saving;
    expect(actions.at(-1)).toBe("complete");
    expect(activities).toEqual([[photo]]);
});

it("leaves an excluded publication private without starting a lease", async () => {
    const baseline = { ...trail(), public: false };
    let form: FormData | undefined;
    const request = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input).includes("/federation/")) return Response.json({ status: "ok" });
        if (init?.body instanceof FormData) form = init.body;
        return Response.json(baseline);
    });
    vi.stubGlobal("fetch", request);
    vi.mocked(assets_attach_to_target).mockResolvedValue([]);
    vi.mocked(has_asset_attachments).mockReturnValue(false);
    const result = await trails_update(baseline, trail(), undefined, undefined, ["public"]);
    expect(request.mock.calls.filter(([input]) => String(input).includes("/federation/"))).toEqual([]);
    expect(form).toBeInstanceOf(FormData);
    expect(form!.has("public")).toBe(false);
    expect(result.public).toBe(false);
});

it("completes an entry save after its deferred photo unlink in the trail editor", async () => {
    const baseline = trail();
    baseline.expand!.summit_logs_via_trail = [{ id: "entry0000000001", date: "2026-11-09", author: "actor", trail: baseline.id, photos: [photo, "/api/v1/assets/second000000001/file"] }];
    const pending = structuredClone(baseline);
    pending.expand!.summit_logs_via_trail![0].photos = [photo];
    const actions: string[] = [];
    vi.mocked(has_asset_attachments).mockReturnValue(false);
    vi.mocked(assets_attach_to_target).mockImplementation(async input => input.existingPhotos ?? []);
    vi.mocked(assets_delete_removed).mockImplementation(async (_old, _next, target) => {
        if (target.summit_log) actions.push("entry:unlink");
    });
    const request = vi.fn(async (input: RequestInfo | URL) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) {
            actions.push(`${path.includes("/summit-log/") ? "entry" : "trail"}:${path.split("/").at(-1)}`);
            return Response.json({ status: "ok" });
        }
        if (path.startsWith("/api/v1/summit-log/form/")) { actions.push("entry:metadata"); return Response.json(baseline.expand!.summit_logs_via_trail![0]); }
        return Response.json(pending);
    });
    vi.stubGlobal("fetch", request);
    await trails_update(baseline, pending);
    expect(actions).toEqual(["trail:begin", "entry:begin", "entry:metadata", "entry:unlink", "entry:complete", "trail:complete"]);
});
