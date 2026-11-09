import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import type { Asset } from "#lib/models/asset.ts";
import { Trail } from "#lib/models/trail.ts";
import { APIError } from "#lib/util/api_util.ts";
import { shownTrailPhotos, trails_create, trails_update } from "./trail_store";

const user: AuthRecord = {
    id: "test-user", collectionId: "users", collectionName: "users", actor: "actor",
};
const id = (name: string) => name.padEnd(15, "0");
const photo = (assetId: string) => `/api/v1/assets/${assetId}/file`;
const upload = (name: string) => new File(["photo"], `${name}.jpg`, { type: "image/jpeg" });

function asset(assetId: string): Asset {
    return {
        id: assetId, collectionId: "assets", collectionName: "assets",
        type: "photo", author: "actor", storage_mode: "link_private",
    };
}

function trail(assetIds: string[] = []): Trail {
    return {
        id: "trail-id", name: "Trail", author: "actor", public: false,
        completed: true, like_count: 0, tags: [], photos: assetIds.map(photo),
        expand: { waypoints_via_trail: [], assets_via_trail: assetIds.map(asset) },
    };
}

// A duplicate as Trail.from builds it: the copied photos are shown first and
// linked by ID when the copy is saved.
function duplicate(thumbnail: number): Trail {
    const copy = trail([id("copied1"), id("copied2")]);
    copy.id = undefined;
    copy._assetLinks = [id("copied1"), id("copied2")];
    copy.thumbnail = thumbnail;
    return copy;
}

// What the editor adds on save for a duplicated photo linked by ID.
function withLinkedPhoto(pending: Trail, assetId: string): Trail {
    pending._assetLinks = [...new Set([...(pending._assetLinks ?? []), assetId])];
    pending.photos = [...new Set([...(pending.photos ?? []), photo(assetId)])];
    return pending;
}

interface Failures { uploads?: number; links?: number; shows?: number }

// Lists a trail's photos in the order they were linked, as the server does,
// and records which asset each thumbnail request marks. `failures` counts the
// requests of each kind that fail before they succeed.
function fakeServer(linked: string[] = [], failures: Failures = {}, copies = false, omitted = new Set<string>(), omittedReason = "download_failed") {
    const marked: (string | null)[] = [];
    const failsNow = (kind: keyof Failures) => {
        if (!failures[kind]) return false;
        failures[kind] = failures[kind]! - 1;
        return true;
    };
    const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
        const path = String(url);
        if (path.includes("/federation/")) return Response.json({ status: "not_federated" });
        if (path.startsWith("/api/v1/trail/form")) return Response.json(trail(linked));
        if (path === "/api/v1/assets" && config?.method === "PUT") {
            const form = config.body as FormData;
            if (failsNow(form.has("files") ? "uploads" : "links")) {
                return Response.json({ message: form.has("files") ? "upload-failed" : "link-failed" }, { status: 500 });
            }
            const assetIds = form.has("files")
                ? form.getAll("files").map((file) => id((file as File).name.replace(/\.jpg$/, "")))
                : form.getAll("assetIds").map(String);
            const assets = assetIds.filter((assetId) => !omitted.has(assetId)).map((assetId) => copies && !form.has("files")
                ? { ...asset(id("local" + assetId.replace(/0+$/, ""))), sourceAssetId: assetId }
                : asset(assetId));
            linked.push(...assets.map((asset) => asset.id));
            return Response.json(omitted.size ? { attached: assets, omitted: assetIds.filter((assetId) => omitted.has(assetId)).map((assetId) => ({ assetId, reason: omittedReason })) } : assets);
        }
        if (path === "/api/v1/trail/trail-id/thumbnail" && config?.method === "PATCH") {
            marked.push(JSON.parse(String(config.body)).asset);
            return Response.json({ acknowledged: true });
        }
        if (path.startsWith("/api/v1/assets/") && config?.method === "DELETE") {
            const assetId = new URL(path, "https://a.example").pathname.split("/").pop()!;
            const index = linked.indexOf(assetId);
            if (index >= 0) linked.splice(index, 1);
            return Response.json({ acknowledged: true, assetDeleted: true });
        }
        if (path.startsWith("/api/v1/trail/trail-id?")) {
            // trails_show expands the author, the refresh right after attaching does not.
            const expand = new URL(path, "https://a.example").searchParams.get("expand")?.split(",");
            if (expand?.includes("author") && failsNow("shows")) {
                return Response.json({ message: "show-failed" }, { status: 500 });
            }
            return Response.json(trail(linked));
        }
        throw new Error(`Unexpected request: ${config?.method ?? "GET"} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    return { request, marked, linked };
}

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("trail thumbnail on save", () => {




    it("marks the new local copy of the selected foreign photo and preserves it on retry", async () => {
        const server = fakeServer([], { shows: 1 }, true);
        const pending = duplicate(1);

        const failure = await trails_create(pending, [], null, server.request, user).catch((error) => error);
        expect(failure.message).toBe("show-failed");
        expect(server.marked).toEqual([id("localcopied2")]);
        const saved: Trail = failure.detail.savedTrail;
        pending.id = saved.id;
        await trails_update(saved, pending);

        expect(server.linked).toEqual([id("localcopied1"), id("localcopied2")]);
        expect(server.marked).toEqual([id("localcopied2"), id("localcopied2")]);
        expect(pending.photos).toEqual(server.linked.map(photo));
    });

    it.each([
        ["the first copied photo", 0, "copied1"],
        ["the second copied photo", 1, "copied2"],
        ["the upload", 2, "upload"],
    ])("marks %s of a duplicate that also uploads a photo", async (_, thumbnail, want) => {
        const server = fakeServer();

        await trails_create(duplicate(thumbnail), [upload("upload")], null, server.request, user);

        expect(server.linked).toEqual([id("upload"), id("copied1"), id("copied2")]);
        expect(server.marked).toEqual([id(want)]);
    });

    it.each([
        ["the first upload", 0, "first"],
        ["the second upload", 1, "second"],
    ])("marks %s of a new trail", async (_, thumbnail, want) => {
        const server = fakeServer();
        const pending = trail();
        pending.id = undefined;
        pending.thumbnail = thumbnail;

        await trails_create(pending, [upload("first"), upload("second")], null, server.request, user);

        expect(server.marked).toEqual([id(want)]);
    });

    it.each([
        ["a saved photo", 0, "keep"],
        ["the upload", 1, "upload"],
    ])("marks %s when a photo linked by ID (duplicated photo) is linked in the same update", async (_, thumbnail, want) => {
        const server = fakeServer([id("keep")]);
        const baseline = trail([id("keep")]);
        const pending = trail([id("keep")]);
        // What the editor adds on save for a duplicated photo linked by ID.
        pending._assetLinks = [id("linked")];
        pending.photos = [...(pending.photos ?? []), photo(id("linked"))];
        pending.thumbnail = thumbnail;

        await trails_update(baseline, pending, [upload("upload")]);

        expect(server.linked).toEqual([id("keep"), id("upload"), id("linked")]);
        expect(server.marked).toEqual([id(want)]);
    });

    it("marks the chosen copied photo when a duplicate is saved again after its upload failed", async () => {
        const server = fakeServer([], { uploads: 1 });
        const pending = duplicate(1);
        const files = [upload("upload")];

        const failure = await trails_create(pending, files, null, server.request, user).catch((error) => error);
        expect(failure).toBeInstanceOf(APIError);
        expect(failure.message).toBe("upload-failed");
        // As the editor does, the retry updates the trail that was created.
        const saved: Trail = failure.detail.savedTrail;
        pending.id = saved.id;

        await trails_update(saved, pending, files);

        expect(server.linked).toEqual([id("upload"), id("copied1"), id("copied2")]);
        expect(server.marked).toEqual([id("copied2")]);
    });

    it("marks the chosen upload of a duplicate again when the save is retried after a late failure", async () => {
        const server = fakeServer([], { shows: 1 });
        const pending = duplicate(2);
        const files = [upload("upload")];

        const failure = await trails_create(pending, files, null, server.request, user).catch((error) => error);
        expect(failure.message).toBe("show-failed");
        const saved: Trail = failure.detail.savedTrail;
        pending.id = saved.id;
        await trails_update(saved, pending, files);

        expect(server.marked).toEqual([id("upload"), id("upload")]);
    });

    it("marks the chosen upload when an update that links a photo linked by ID (duplicated photo) is retried after a late failure", async () => {
        const server = fakeServer([id("keep")], { shows: 1 });
        const baseline = trail([id("keep")]);
        const pending = withLinkedPhoto(trail([id("keep")]), id("linked"));
        pending.thumbnail = 1;
        const files = [upload("upload")];

        await expect(trails_update(baseline, pending, files)).rejects.toMatchObject({ message: "show-failed" });
        await trails_update(baseline, pending, files);

        expect(server.marked).toEqual([id("upload")]);
    });

    it("marks the chosen photo again when an update with an unchanged index is retried after a late failure", async () => {
        // The marked photo is removed and the upload takes its place in the
        // editor's order, so the index stays the same.
        const server = fakeServer([id("keep"), id("old")], { shows: 1 });
        const baseline = trail([id("keep"), id("old")]);
        baseline.thumbnail = 1;
        const pending = trail([id("keep")]);
        pending.thumbnail = 1;
        const files = [upload("upload")];

        await expect(trails_update(baseline, pending, files)).rejects.toMatchObject({ message: "show-failed" });
        await trails_update(baseline, pending, files);

        expect(server.linked).toEqual([id("keep"), id("upload")]);
        expect(server.marked).toEqual([id("upload")]);
    });


});


it.each(["create", "update"])("keeps the chosen pending upload as thumbnail after a failed %s and retry", async (mode) => {
    const copied = [id("copied1"), id("copied2")];
    const server = fakeServer(mode === "create" ? [] : copied, { uploads: 1 });
    const pending = mode === "create" ? duplicate(3) : { ...trail(copied), thumbnail: 3 };
    const files = [upload("new1"), upload("new2")];
    const baseline = trail(copied);
    const failure = await (mode === "create"
        ? trails_create(pending, files, null, server.request, user)
        : trails_update(baseline, pending, files)).catch(error => error);
    expect(failure).toBeInstanceOf(APIError);
    expect(pending.thumbnail).toBe(3);
    if (mode === "create") pending.id = failure.detail.savedTrail.id;
    await trails_update(mode === "create" ? failure.detail.savedTrail : baseline, pending, files);
    expect(server.marked.at(-1)).toBe(id("new2"));
    expect(pending.photos![pending.thumbnail!]).toBe(photo(id("new2")));
});
