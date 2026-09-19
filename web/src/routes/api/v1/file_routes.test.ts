import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it, vi } from "vitest";

vi.mock("$lib/util/gpx_util", () => ({
    applyGpxToForm: vi.fn(async () => {}),
    trailGpxFields: [],
    summitLogGpxFields: [],
}));

import { POST as trailPOST } from "./trail/[id]/file/+server";
import { POST as summitLogPOST } from "./summit-log/[id]/file/+server";
import { POST as waypointPOST } from "./waypoint/[id]/file/+server";

function fileEvent(route: string, field: string) {
    const id = "record000000001";
    const data = new FormData();
    if (field.endsWith("-")) {
        data.set(field, "old-file.gpx");
    } else {
        data.set(field, new File(["file contents"], "upload.gpx"));
    }
    const url = new URL(`https://wanderer.example/api/v1/${route}/${id}/file`);
    const result = { id, date: "2026-09-15 00:00:00.000Z" };
    const update = vi.fn(async (_id: string, _data: FormData, _options?: unknown) => ({ ...result }));
    const getOne = vi.fn(async () => ({ ...result }));
    const collection = vi.fn(() => ({ update, getOne }));
    const event = {
        params: { id },
        url,
        request: new Request(url, { method: "POST", body: data }),
        locals: { user: { id: "user" }, pb: { collection } },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
    return { event, update, collection };
}

describe.each([
    ["trail", trailPOST, "trails"],
    ["summit-log", summitLogPOST, "summit_logs"],
] as const)("legacy %s file endpoint", (route, POST, collectionName) => {
    it.each(["photos", "photos+", "photos-"])("rejects `%s` without updating PocketBase", async (field) => {
        const { event, update } = fileEvent(route, field);

        const response = await POST(event);

        expect(response.status).toBe(400);
        expect(await response.json()).toMatchObject({ message: "missing_file", expected: ["gpx"] });
        expect(update).not.toHaveBeenCalled();
    });

    it.each(["gpx", "gpx+", "gpx-"])("forwards `%s` to the form endpoint with its body intact", async (field) => {
        const { event, update, collection } = fileEvent(route, field);

        const response = await POST(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toMatchObject({ id: event.params.id, date: "2026-09-15" });
        expect(collection).toHaveBeenCalledWith(collectionName);
        expect(update).toHaveBeenCalledOnce();
        const [id, data] = update.mock.calls[0];
        expect(id).toBe(event.params.id);
        const part = data.get(field);
        if (field.endsWith("-")) {
            expect(part).toBe("old-file.gpx");
        } else {
            expect(part).toBeInstanceOf(File);
            expect(await (part as File).text()).toBe("file contents");
        }
    });
});

describe("removed waypoint file endpoint", () => {
    it.each(["photos", "photos+", "photos-"])("rejects `%s` without updating PocketBase", async (field) => {
        const { event, collection } = fileEvent("waypoint", field);

        const response = await waypointPOST(event);

        expect(response.status).toBe(400);
        expect(await response.json()).toEqual({ message: "waypoint_photos_form_field_removed" });
        expect(collection).not.toHaveBeenCalled();
    });

    it("preserves the unauthorized response", async () => {
        const { event, collection } = fileEvent("waypoint", "photos");
        event.locals.user = null;

        const response = await waypointPOST(event);

        expect(response.status).toBe(401);
        expect(await response.json()).toEqual({ message: "Unauthorized" });
        expect(collection).not.toHaveBeenCalled();
    });
});
