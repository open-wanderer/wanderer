import { afterEach, describe, expect, it, vi } from "vitest";
import type { AuthRecord } from "pocketbase";
import type { Trail } from "#lib/models/trail.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import { trails_create, trails_update } from "./trail_store";
import { currentUser } from "./user_store";

const user: AuthRecord = {
    id: "test-user", collectionId: "users", collectionName: "users", actor: "actor",
};
const assetId = (id: string) => id.padEnd(15, "0");
const photo = (id: string) => `/api/v1/assets/${assetId(id)}/file`;
function trail(photos: string[] = []): Trail {
    return {
        id: "trail-id", name: "Trail", author: "actor", public: false,
        completed: true, like_count: 0, tags: [], photos,
        expand: { waypoints_via_trail: [] },
    };
}

afterEach(() => {
    vi.unstubAllGlobals();
    currentUser.set(null);
});

describe("trail photo saves", () => {
    it("retries a file batch after dropping its failed file without reuploading the saved file", async () => {
        const pending = trail();
        const files = [new File(["ok"], "a.jpg", { type: "image/jpeg" }), new File(["invalid"], "b.gif", { type: "image/gif" })];
        const stored: string[] = [];
        const uploads: string[][] = [];
        const request = vi.fn(async (url: RequestInfo | URL, config?: RequestInit) => {
            const path = String(url);
            if (path.includes("/federation/")) return Response.json({ status: "not_federated" });
            if (path.startsWith("/api/v1/trail/form")) return Response.json(trail(stored));
            if (path === "/api/v1/assets") {
                uploads.push((config!.body as FormData).getAll("files").map((file) => (file as File).name));
                stored.push(photo("saved"));
                return Response.json({ message: "Unsupported file", detail: { uploadedAssets: [{ id: assetId("saved") }], completedFileCount: 1 } }, { status: 400 });
            }
            if (path.endsWith("/thumbnail")) return Response.json({ acknowledged: true });
            if (path.startsWith("/api/v1/trail/trail-id?")) return Response.json(trail([...stored]));
            throw new Error(`Unexpected request: ${path}`);
        });
        vi.stubGlobal("fetch", request);
        const failure = await trails_create(pending, files, null, request, user).catch((error) => error);

        expect(files.map((file) => file.name)).toEqual(["b.gif"]);
        expect(failure.detail.savedTrail.photos).toEqual([photo("saved")]);
        expect(pending.photos).toEqual([photo("saved")]);
        files.splice(0); // The user removes the rejected upload.
        pending.id = failure.detail.savedTrail.id;
        await trails_update(failure.detail.savedTrail, pending, files);

        expect(uploads).toEqual([["a.jpg", "b.gif"]]);
        expect(stored).toEqual([photo("saved")]);
    });






});
