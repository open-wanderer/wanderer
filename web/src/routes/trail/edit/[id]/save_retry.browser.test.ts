import { mount, unmount, untrack } from "svelte";
import ComponentScriptTestHost from "../../../../../tests/component_script_test_host.svelte";
import { createForm } from "felte";
import { get } from "svelte/store";
import { afterEach, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import { APIError } from "#lib/util/api_util.ts";
import { componentFunctions } from "../../../../../tests/component_script_test_util.ts";
import { copyTrailEditorValues, initialTrailFormValues } from "#lib/util/trail_editor_util.ts";
import { currentUser } from "#lib/stores/user_store.ts";
import { shownTrailPhotos, trails_create, trails_update } from "#lib/stores/trail_store.ts";

vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));

type EditorValues = { id?: string; photos?: string[]; thumbnail?: number; _assetLinks?: string[] };
const actor = "actor0000000001";
const photoID = "upload000000001";
const photo = `/api/v1/files/assets/${photoID}/outing.jpg`;
const copiedID = "copied000000001";
const copiedPhoto = `/api/v1/files/assets/${copiedID}/copied.jpg`;
function saved(photos: string[] = []): Trail {
    return Object.assign(new Trail("Outing", { id: "trail000000001", photos, thumbnail: 0, public: false }), { author: actor });
}
const mounted: ReturnType<typeof mount>[] = [];
afterEach(async () => { await Promise.all(mounted.splice(0).map(component => unmount(component))); vi.unstubAllGlobals(); vi.restoreAllMocks(); currentUser.set(null); document.body.replaceChildren(); });

it.each(["update-refresh", "create-waypoint"] as const)("retains completed photo work through two Felte submits after %s fails", async (mode) => {
    const baseline = saved(mode === "update-refresh" ? [copiedPhoto] : []);
    baseline.expand = { waypoints_via_trail: [], summit_logs_via_trail: [] };
    const initial = copyTrailEditorValues(baseline);
    if (mode === "create-waypoint") {
        initial.id = undefined;
        initial.photos = [copiedPhoto]; initial._assetLinks = [copiedID];
        initial.expand!.assets_via_trail = [{ id: copiedID, collectionId: "assets", collectionName: "assets", type: "photo", author: actor, storage_mode: "copy", file: "copied.jpg" }];
        initial.expand!.waypoints_via_trail = [{ lat: 47, lon: 8, name: "Summit", author: actor, photos: [], _photos: [] }];
    }
    initial.thumbnail = 1;
    let storedPhotos = mode === "update-refresh" ? [copiedPhoto] : [];
    let firstFailure = true;
    let cover: string | null = mode === "update-refresh" ? copiedID : null;
    const uploads: string[][] = [];
    const marks: (string | null)[] = [];
    const methods: string[] = [];
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const url = new URL(String(input), "https://a.example"); const method = options?.method ?? "GET";
        if (url.pathname.includes("/federation/")) return Response.json({ status: "not_federated" });
        if (url.pathname.startsWith("/api/v1/trail/form")) { methods.push(method); return Response.json(saved(storedPhotos)); }
        if (url.pathname === "/api/v1/assets") {
            const body = options!.body as FormData;
            if (body.has("files")) {
                uploads.push(body.getAll("files").map(file => (file as File).name));
                storedPhotos.push(photo);
                cover ??= photoID;
                return Response.json([{ id: photoID, collectionId: "assets", type: "photo", file: "outing.jpg" }]);
            }
            storedPhotos.push(copiedPhoto);
            return Response.json([{ id: copiedID, collectionId: "assets", type: "photo", file: "copied.jpg" }]);
        }
        if (url.pathname === "/api/v1/waypoint") {
            if (firstFailure) { firstFailure = false; return Response.json({ message: "waypoint unavailable" }, { status: 503 }); }
            return Response.json({ ...initial.expand!.waypoints_via_trail![0], id: "waypoint0000001", trail: baseline.id });
        }
        if (url.pathname.endsWith("/thumbnail")) { cover = JSON.parse(String(options?.body)).asset; marks.push(cover); return Response.json({ acknowledged: true }); }
        if (url.pathname === `/api/v1/trail/${baseline.id}`) {
            if (mode === "update-refresh" && firstFailure) { firstFailure = false; return Response.json({ message: "refresh unavailable" }, { status: 503 }); }
            return Response.json(saved([...storedPhotos]));
        }
        if (method === "DELETE") { storedPhotos = storedPhotos.filter(value => !value.includes(url.pathname.split("/").at(-1)!)); return Response.json({ acknowledged: true }); }
        throw new Error(`Unexpected ${method} ${url.pathname}`);
    });
    currentUser.set({ id: "user", actor, collectionName: "users", collectionId: "users" } as never);
    vi.stubGlobal("fetch", request);
    vi.spyOn(console, "error").mockImplementation(() => {});
    document.body.innerHTML = '<form id="trail-form"></form>';
    let felte!: ReturnType<typeof createForm<EditorValues>>;
    const files = [new File(["photo"], "outing.jpg", { type: "image/jpeg" })];
    mounted.push(mount(ComponentScriptTestHost, { target: document.body, props: { setup: () => componentFunctions("src/routes/trail/edit/[id]/+page.svelte", ["form"], {
        createForm: (config: Parameters<typeof createForm<EditorValues>>[0]) => (felte = createForm<EditorValues>(config)),
        initialTrailFormValues, untrack, data: { trail: initial }, defaultCategoryId: () => "",
        validator: () => () => ({}), ClientTrailCreateSchema: {},
        loading: false, document, FormData, valhallaStore: { route: { flatten: () => [] } },
        routeToStoredGpx: () => undefined, overwriteGPX: false, isNewTrail: mode === "create-waypoint",
        savedAtLeastOnce: false, gpxFile: null, routeHasTrackPoints: () => false,
        page: { params: { id: mode === "create-waypoint" ? "new" : baseline.id }, data: {} },
        trails_create, trails_update, photoFiles: files, copyTrailEditorValues,
        completionDateValue: (value: unknown) => value,
        trail: { set: (value: Trail) => Object.assign(baseline, value) }, $trail: baseline, $formData: initial,
        savedPublic: false, startPublic: false, show_toast: vi.fn(), $_: (key: string) => key,
        APIError, shownTrailPhotos, console,
    }) } }));
    await felte.handleSubmit();
    expect(get(felte.data).photos).toContain(photo);
    expect(get(felte.data).photos?.[get(felte.data).thumbnail!]).toBe(photo);
    expect(get(felte.data)._assetLinks).toBeUndefined();
    expect(files).toEqual([]);
    if (mode === "create-waypoint") expect(get(felte.data).id).toBe(baseline.id);
    await felte.handleSubmit();
    expect(uploads).toEqual([["outing.jpg"]]);
    expect(storedPhotos).toContain(photo);
    expect(cover).toBe(photoID);
    expect(methods).toEqual(mode === "create-waypoint" ? ["PUT", "POST"] : ["POST", "POST"]);
    expect(request.mock.calls.filter(([, options]) => options?.method === "DELETE")).toEqual([]);
});

it.each(["create", "update"] as const)("retries only unfinished photo batches through the %s editor's Felte form", async mode => {
    const baseline = saved();
    baseline.expand = { waypoints_via_trail: [], summit_logs_via_trail: [] };
    const initial = copyTrailEditorValues(baseline);
    if (mode === "create") initial.id = undefined;
    initial.thumbnail = 15;
    const files = Array.from({ length: 16 }, (_, index) => {
        const file = new File(["photo"], `photo${index}.jpg`, { type: "image/jpeg" });
        Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 });
        return file;
    });
    const names = files.map(file => file.name);
    const id = (name: string) => `photo${name.match(/\d+/)![0].padStart(10, "0")}`;
    const asset = (name: string) => ({ id: id(name), collectionId: "assets", type: "photo", file: name });
    const photos = names.map(name => `/api/v1/files/assets/${id(name)}/${name}`);
    const storedPhotos: string[] = [];
    const batches: string[][] = [];
    const uploaded: string[] = [];
    const methods: string[] = [];
    let cover: string | null = null;
    const snapshot = () => ({ ...saved([...storedPhotos]), thumbnail: Math.max(0, storedPhotos.findIndex(photo => photo.includes(`/${cover}/`))) });
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) return Response.json({ status: "not_federated" });
        if (path.startsWith("/api/v1/trail/form")) { methods.push(options!.method!); return Response.json(snapshot()); }
        if (path === "/api/v1/assets") {
            const selected = (options!.body as FormData).getAll("files") as File[];
            batches.push(selected.map(file => file.name));
            const failing = batches.length === 2;
            const completed = failing ? selected.slice(0, 2) : selected;
            const records = completed.map(file => asset(file.name));
            uploaded.push(...completed.map(file => file.name));
            storedPhotos.push(...completed.map(file => photos[names.indexOf(file.name)]));
            return failing ? Response.json({ message: "Photo save interrupted", detail: {
                uploadedAssets: records, attachedAssets: records, completedFileCount: completed.length,
            } }, { status: 503 }) : Response.json(records);
        }
        if (path.endsWith("/thumbnail")) { cover = JSON.parse(String(options?.body)).asset; return Response.json({ acknowledged: true }); }
        if (path === `/api/v1/trail/${baseline.id}`) return Response.json(snapshot());
        throw new Error(`Unexpected ${options?.method ?? "GET"} ${path}`);
    });
    currentUser.set({ id: "user", actor, collectionName: "users", collectionId: "users" } as never);
    vi.stubGlobal("fetch", request);
    vi.spyOn(console, "error").mockImplementation(() => {});
    document.body.innerHTML = '<form id="trail-form"></form>';
    let felte!: ReturnType<typeof createForm<EditorValues>>;
    mounted.push(mount(ComponentScriptTestHost, { target: document.body, props: { setup: () => componentFunctions("src/routes/trail/edit/[id]/+page.svelte", ["form"], {
        createForm: (config: Parameters<typeof createForm<EditorValues>>[0]) => (felte = createForm<EditorValues>(config)),
        initialTrailFormValues, untrack, data: { trail: initial }, defaultCategoryId: () => "",
        validator: () => () => ({}), ClientTrailCreateSchema: {},
        loading: false, document, FormData, valhallaStore: { route: { flatten: () => [] } },
        routeToStoredGpx: () => undefined, overwriteGPX: false, isNewTrail: mode === "create",
        savedAtLeastOnce: false, gpxFile: null, routeHasTrackPoints: () => false,
        page: { params: { id: mode === "create" ? "new" : baseline.id }, data: {} },
        trails_create, trails_update, photoFiles: files, copyTrailEditorValues,
        completionDateValue: (value: unknown) => value,
        trail: { set: (value: Trail) => Object.assign(baseline, value) }, $trail: baseline, $formData: initial,
        savedPublic: false, startPublic: false, show_toast: vi.fn(), $_: (key: string) => key,
        APIError, shownTrailPhotos, console,
    }) } }));
    await felte.handleSubmit();
    expect(get(felte.data).id).toBe(baseline.id);
    expect(get(felte.data).photos).toEqual(photos.slice(0, 14));
    expect(files.map(file => file.name)).toEqual(names.slice(14));
    await felte.handleSubmit();
    expect(batches).toEqual([names.slice(0, 12), names.slice(12), names.slice(14)]);
    expect(uploaded).toEqual(names);
    expect(storedPhotos).toEqual(photos);
    expect(get(felte.data).photos).toEqual(photos);
    expect(get(felte.data).photos?.[get(felte.data).thumbnail!]).toBe(photos[15]);
    expect(cover).toBe(id(names[15]));
    expect(methods).toEqual(mode === "create" ? ["PUT", "POST"] : ["POST", "POST"]);
    expect(files).toEqual([]);
    expect(request.mock.calls.filter(([, options]) => options?.method === "DELETE")).toEqual([]);
});
