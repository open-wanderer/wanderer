import type { RequestEvent } from "@sveltejs/kit";
import PocketBase from "pocketbase";
import { describe, expect, it, vi } from "vitest";
import { PUT as createTrailForm } from "./trail/form/+server";
import { POST as updateTrailForm } from "./trail/form/[id]/+server";
import { PUT as createWaypoint } from "./waypoint/+server";
import { POST as updateWaypoint } from "./waypoint/[id]/+server";

const trailId = "trail0000000001";
const waypointId = "point0000000001";
const authorId = "author000000001";

// These app request builders are identical in the released and current app:
// util/trail/form_data.dart, provider/waypoint/waypoint_provider.dart and
// models/waypoint.g.dart. Trail forms serialize fields individually; only
// waypoint JSON echoes the model's existing photos when no photo changed.
const appVersions = ["released app beta.25", "current app"];

function trailForm(isCreate: boolean): FormData {
    const data = new FormData();
    if (!isCreate) data.append("id", trailId);
    for (const [key, value] of [
        ["name", "Summit walk"],
        ["location", "Valley"],
        ["date", "2026-11-10T09:00:00.000"],
        ["description", "A saved route"],
        ["public", "false"],
        ["completed", "false"],
        ["difficulty", ""],
        ["distance", "1200.0"],
        ["elevation_gain", "100.0"],
        ["elevation_loss", "80.0"],
        ["duration", "1800.0"],
        ["moving_duration", "1700.0"],
        ["lat", "46.8"],
        ["lon", "8.2"],
        ["category", "category0000001"],
        ["subcategory", "subcat000000001"],
        ["author", authorId],
        ["tags", "tag000000000001"],
        ["tags", "tag000000000002"],
    ]) data.append(key, value);
    return data;
}

// WaypointSave removes record metadata from Waypoint.toJson(), adds the
// create's author/trail, and removes those two bindings on update.
function waypointBody(isCreate: boolean, photos: string[], parent = trailId) {
    return {
        name: "Viewpoint",
        description: "Above the valley",
        lat: 46.8,
        lon: 8.2,
        distance_from_start: 125.0,
        icon: "circle",
        photos,
        ...(isCreate ? { author: authorId, trail: parent } : {}),
    };
}

function backend(collection: "trails" | "waypoints") {
    const id = collection === "trails" ? trailId : waypointId;
    const saved = {
        id,
        collectionId: collection,
        collectionName: collection,
        name: collection === "trails" ? "Summit walk" : "Viewpoint",
        description: "Saved description",
        author: authorId,
        trail: collection === "waypoints" ? trailId : undefined,
        lat: 46.8,
        lon: 8.2,
        date: "2026-11-10 09:00:00.000Z",
        created: "2026-11-10 09:00:00.000Z",
        updated: "2026-11-10 09:00:00.000Z",
        expand: collection === "trails" ? { trail_assets_via_trail: [] } : {},
    };
    const requests: { url: URL; method: string; body?: BodyInit | null }[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const url = new URL(String(input));
        requests.push({ url, method: options?.method ?? "GET", body: options?.body });
        return Response.json(saved);
    });
    const pb = new PocketBase("https://b.example");
    pb.beforeSend = (url, options) => ({ url, options: { ...options, fetch } });
    return { pb, fetch, requests };
}

function event(path: string, method: string, body: FormData | Record<string, unknown>, server: ReturnType<typeof backend>): RequestEvent {
    const url = new URL("https://a.example/api/v1/" + path);
    return {
        url,
        params: { id: path.startsWith("waypoint") ? waypointId : trailId },
        request: new Request(url, {
            method,
            ...(body instanceof FormData
                ? { body }
                : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
        }),
        locals: { pb: server.pb, user: { id: "user00000000001" } },
        fetch: server.fetch,
    } as unknown as RequestEvent;
}

function oneWrite(server: ReturnType<typeof backend>, collection: string, update = false) {
    const writes = server.requests.filter(({ method }) => method !== "GET");
    expect(writes).toHaveLength(1);
    expect(writes[0].method).toBe(update ? "PATCH" : "POST");
    expect(writes[0].url.pathname).toBe(`/api/collections/${collection}/records${update ? "/" + (collection === "trails" ? trailId : waypointId) : ""}`);
    return writes[0];
}

describe.each(appVersions)("%s saves without photo changes", () => {
    it("creates a trail from the app multipart body", async () => {
        const server = backend("trails");
        const response = await createTrailForm(event("trail/form?expand=category,tags,author", "PUT", trailForm(true), server));

        expect(response.status).toBe(200);
        expect((await response.json()).id).toBe(trailId);
        const write = oneWrite(server, "trails");
        const body = write.body as FormData;
        expect(body.get("name")).toBe("Summit walk");
        expect(body.getAll("tags")).toEqual(["tag000000000001", "tag000000000002"]);
        expect(body.has("photos")).toBe(false);
        expect(body.has("@jsonPayload")).toBe(false);
    });

    it("updates a trail from the app multipart body", async () => {
        const server = backend("trails");
        const response = await updateTrailForm(event(`trail/form/${trailId}?expand=category,tags,author`, "POST", trailForm(false), server));

        expect(response.status).toBe(200);
        expect((await response.json()).id).toBe(trailId);
        const write = oneWrite(server, "trails", true);
        const body = write.body as FormData;
        expect(body.get("id")).toBe(trailId);
        expect(body.get("name")).toBe("Summit walk");
        expect(body.has("photos")).toBe(false);
        expect(body.has("@jsonPayload")).toBe(false);
    });

    it("creates a waypoint while echoing the empty photo list", async () => {
        const server = backend("waypoints");
        const response = await createWaypoint(event("waypoint", "PUT", waypointBody(true, []), server));

        expect(response.status).toBe(200);
        expect((await response.json()).id).toBe(waypointId);
        const body = JSON.parse(String(oneWrite(server, "waypoints").body));
        expect(body).toMatchObject({ name: "Viewpoint", lat: 46.8, lon: 8.2, author: authorId, trail: trailId });
        expect(body).not.toHaveProperty("photos");
    });

    it("updates a waypoint while echoing its current photo list", async () => {
        const server = backend("waypoints");
        const response = await updateWaypoint(event(`waypoint/${waypointId}`, "POST", waypointBody(false, ["existing_photo.png"]), server));

        expect(response.status).toBe(200);
        expect((await response.json()).id).toBe(waypointId);
        const body = JSON.parse(String(oneWrite(server, "waypoints", true).body));
        expect(body).toMatchObject({ name: "Viewpoint", lat: 46.8, lon: 8.2 });
        expect(body).not.toHaveProperty("photos");
        expect(body).not.toHaveProperty("author");
        expect(body).not.toHaveProperty("trail");
    });

    it("drains a pending trail and creates its pending waypoint", async () => {
        // TrailSync creates a local trail, then passes the returned server ID
        // to WaypointSave.create for each pending WaypointEntity.toModel().
        const trailServer = backend("trails");
        const trailResponse = await createTrailForm(event("trail/form?expand=category,tags,author", "PUT", trailForm(true), trailServer));
        expect(trailResponse.status).toBe(200);
        const trail = await trailResponse.json();
        oneWrite(trailServer, "trails");

        const waypointServer = backend("waypoints");
        const waypointResponse = await createWaypoint(event("waypoint", "PUT", waypointBody(true, [], trail.id), waypointServer));
        expect(waypointResponse.status).toBe(200);
        expect((await waypointResponse.json()).id).toBe(waypointId);
        const body = JSON.parse(String(oneWrite(waypointServer, "waypoints").body));
        expect(body.trail).toBe(trail.id);
        expect(body).not.toHaveProperty("photos");
    });
});
