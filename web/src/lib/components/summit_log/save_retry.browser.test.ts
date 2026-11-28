import { mount, unmount } from "svelte";
import ComponentScriptTestHost from "../../../../tests/component_script_test_host.svelte";
import { createForm } from "felte";
import { get } from "svelte/store";
import { afterEach, expect, it, vi } from "vitest";
import { SummitLog } from "#lib/models/summit_log.ts";
import { APIError } from "#lib/util/api_util.ts";
import { componentFunctions } from "../../../../tests/component_script_test_util.ts";
import { summit_logs_create, summit_logs_update, summitLogs } from "#lib/stores/summit_log_store.ts";
import { currentUser } from "#lib/stores/user_store.ts";

vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));

type EntryValues = { id?: string; photos?: string[]; _photos?: File[] };
const mounted: ReturnType<typeof mount>[] = [];
afterEach(async () => { await Promise.all(mounted.splice(0).map(component => unmount(component))); vi.unstubAllGlobals(); currentUser.set(null); summitLogs.set([]); });
it("retries the trail-page summit-book form without creating a second entry", async () => {
    currentUser.set({ id: "user", actor: "actor", collectionName: "users", collectionId: "users" } as never);
    const log = new SummitLog("2026-11-09"); log._photos = [new File(["photo"], "outing.jpg")];
    const saved = { id: "entry0000000001", author: "actor", trail: "trail", date: log.date, photos: [] };
    let failed = false;
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) return Response.json({ status: "ok" });
        if (path === "/api/v1/assets") {
            if (!failed) { failed = true; return Response.json({ message: "upload unavailable" }, { status: 503 }); }
            return Response.json([{ id: "asset0000000001", collectionId: "assets", type: "photo", file: "outing.jpg" }]);
        }
        if (path.startsWith("/api/v1/summit-log/form")) return Response.json(saved);
        throw new Error(`Unexpected ${options?.method} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    const logs: SummitLog[] = [];
    const { saveSummitLog } = componentFunctions<{ saveSummitLog: (value: SummitLog) => Promise<boolean | void> }>(
        "src/lib/components/trail/trail_info_panel.svelte", ["saveSummitLog"], {
            summitLogCreateLoading: false, pendingSummitLogCompletionID: undefined, summitLogModalGeneration: 0, $summitLogs: logs,
            summitLogs: { set: (value: SummitLog[]) => { logs.splice(0, logs.length, ...value); summitLogs.set(value); } },
            trail: { id: "trail", author: "actor", completed: true }, $currentUser: { actor: "actor" },
            summit_logs_create, summit_logs_update, APIError, show_toast: vi.fn(), $_: (key: string) => key,
        },
    );
    let felte!: ReturnType<typeof createForm<EntryValues>>;
    const close = vi.fn();
    mounted.push(mount(ComponentScriptTestHost, { target: document.body, props: { setup: () => componentFunctions("src/lib/components/summit_log/summit_log_modal.svelte", ["form"], {
        createForm: (config: Parameters<typeof createForm<EntryValues>>[0]) => (felte = createForm<EntryValues>(config)),
        $summitLog: log, ClientSummitLogCreateSchema: {}, validator: () => () => ({}),
        onsave: saveSummitLog, saving: false, openGeneration: 0, modal: { closeModal: close, isOpen: () => true },
    }) } }));
    await felte.handleSubmit();
    expect(logs).toHaveLength(1);
    expect(get(felte.data).id).toBe(saved.id);
    expect(close).not.toHaveBeenCalled();
    await felte.handleSubmit();
    expect(close).toHaveBeenCalledOnce();
    expect(logs).toHaveLength(1);
    expect(logs[0].photos).toEqual(["/api/v1/files/assets/asset0000000001/outing.jpg"]);
    expect(request.mock.calls.filter(([url, options]) => String(url).includes("/summit-log/form") && options?.method === "PUT")).toHaveLength(1);
    expect(request.mock.calls.filter(([url, options]) => String(url).includes("/summit-log/form/") && options?.method === "POST")).toHaveLength(1);
});

it.each(["create", "update"] as const)("keeps completed photo batches through two %s summit-book Felte submits", async mode => {
    currentUser.set({ id: "user", actor: "actor", collectionName: "users", collectionId: "users" } as never);
    const log = new SummitLog("2026-11-09", { id: mode === "update" ? "entry0000000001" : undefined });
    const files = Array.from({ length: 16 }, (_, index) => {
        const file = new File(["photo"], `photo${index}.jpg`, { type: "image/jpeg" });
        Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 });
        return file;
    });
    log._photos = files;
    const names = files.map(file => file.name);
    const id = (name: string) => `photo${name.match(/\d+/)![0].padStart(10, "0")}`;
    const photos = names.map(name => `/api/v1/files/assets/${id(name)}/${name}`);
    const storedPhotos: string[] = [];
    const batches: string[][] = [];
    const uploaded: string[] = [];
    const saved = () => ({ id: "entry0000000001", author: "actor", trail: "trail", date: log.date, photos: [...storedPhotos] });
    const request = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const path = new URL(String(input), "https://a.example").pathname;
        if (path.includes("/federation/")) return Response.json({ status: "ok" });
        if (path === "/api/v1/assets") {
            const selected = (options!.body as FormData).getAll("files") as File[];
            batches.push(selected.map(file => file.name));
            const failing = batches.length === 2;
            const completed = failing ? selected.slice(0, 2) : selected;
            const records = completed.map(file => ({ id: id(file.name), collectionId: "assets", type: "photo", file: file.name }));
            uploaded.push(...completed.map(file => file.name));
            storedPhotos.push(...completed.map(file => photos[names.indexOf(file.name)]));
            return failing ? Response.json({ message: "Photo save interrupted", detail: {
                uploadedAssets: records, attachedAssets: records, completedFileCount: completed.length,
            } }, { status: 503 }) : Response.json(records);
        }
        if (path.startsWith("/api/v1/summit-log/form")) return Response.json(saved());
        throw new Error(`Unexpected ${options?.method} ${path}`);
    });
    vi.stubGlobal("fetch", request);
    const logs: SummitLog[] = mode === "update" ? [saved()] : [];
    const { saveSummitLog } = componentFunctions<{ saveSummitLog: (value: SummitLog) => Promise<boolean | void> }>(
        "src/lib/components/trail/trail_info_panel.svelte", ["saveSummitLog"], {
            summitLogCreateLoading: false, pendingSummitLogCompletionID: undefined, summitLogModalGeneration: 0, $summitLogs: logs,
            summitLogs: { set: (value: SummitLog[]) => { logs.splice(0, logs.length, ...value); summitLogs.set(value); } },
            trail: { id: "trail", author: "actor", completed: true }, $currentUser: { actor: "actor" },
            summit_logs_create, summit_logs_update, APIError, show_toast: vi.fn(), $_: (key: string) => key,
        },
    );
    let felte!: ReturnType<typeof createForm<EntryValues>>;
    const close = vi.fn();
    mounted.push(mount(ComponentScriptTestHost, { target: document.body, props: { setup: () => componentFunctions("src/lib/components/summit_log/summit_log_modal.svelte", ["form"], {
        createForm: (config: Parameters<typeof createForm<EntryValues>>[0]) => (felte = createForm<EntryValues>(config)),
        $summitLog: log, ClientSummitLogCreateSchema: {}, validator: () => () => ({}),
        onsave: saveSummitLog, saving: false, openGeneration: 0, modal: { closeModal: close, isOpen: () => true },
    }) } }));
    await felte.handleSubmit();
    expect(close).not.toHaveBeenCalled();
    expect(logs).toHaveLength(1);
    expect(get(felte.data).id).toBe("entry0000000001");
    expect(get(felte.data).photos).toEqual(photos.slice(0, 14));
    expect(get(felte.data)._photos?.map(file => file.name)).toEqual(names.slice(14));
    await felte.handleSubmit();
    expect(close).toHaveBeenCalledOnce();
    expect(logs).toHaveLength(1);
    expect(logs[0].photos).toEqual(photos);
    expect(batches).toEqual([names.slice(0, 12), names.slice(12), names.slice(14)]);
    expect(uploaded).toEqual(names);
    expect(request.mock.calls.filter(([url, options]) => String(url).includes("/summit-log/form") && options?.method === "PUT")).toHaveLength(mode === "create" ? 1 : 0);
    expect(request.mock.calls.filter(([url, options]) => String(url).includes("/summit-log/form/") && options?.method === "POST")).toHaveLength(mode === "create" ? 1 : 2);
    expect(request.mock.calls.filter(([, options]) => options?.method === "DELETE")).toEqual([]);
});
