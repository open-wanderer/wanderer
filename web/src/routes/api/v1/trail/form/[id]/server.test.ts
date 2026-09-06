import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fromFile, gpx2trail } from "$lib/util/gpx_util";
import { POST } from "./+server";

vi.mock("$lib/util/gpx_util", () => ({
    fromFile: vi.fn(),
    gpx2trail: vi.fn(),
}));

const derivedStats = {
    distance: 5980.53,
    duration: 3600,
    elevation_gain: 123,
    elevation_loss: 45,
};

describe("POST /api/v1/trail/form/[id]", () => {
    const update = vi.fn();

    beforeEach(() => {
        update.mockReset();
        update.mockResolvedValue({ id: "trail1234567890" });
        vi.mocked(fromFile).mockReset();
        vi.mocked(gpx2trail).mockReset();
        vi.mocked(fromFile).mockResolvedValue({ gpxData: "<gpx />" } as never);
        vi.mocked(gpx2trail).mockResolvedValue({ trail: derivedStats } as never);
    });

    it("recomputes the stored stats when replacing the GPX", async () => {
        const data = formDataWithId();
        data.set("gpx", new Blob(["replacement GPX"], { type: "application/gpx+xml" }));

        const response = await POST(eventFor(data));

        expect(response.status).toBe(200);
        expect(gpx2trail).toHaveBeenCalledWith("<gpx />", undefined, true, expect.any(Function));
        expect(Object.fromEntries(updatedData())).toMatchObject({
            distance: "5980.53",
            duration: "3600",
            elevation_gain: "123",
            elevation_loss: "45",
        });
    });

    it("leaves stats untouched for a metadata-only update", async () => {
        const data = formDataWithId();
        data.set("name", "renamed trail");

        const response = await POST(eventFor(data));

        expect(response.status).toBe(200);
        expect(gpx2trail).not.toHaveBeenCalled();
        expect(updatedData().get("distance")).toBeNull();
    });

    function formDataWithId() {
        const data = new FormData();
        data.set("id", "trail1234567890");
        return data;
    }

    function eventFor(data: FormData) {
        return {
            request: new Request("http://localhost/api/v1/trail/form/trail1234567890", {
                method: "POST",
                body: data,
            }),
            url: new URL("http://localhost/api/v1/trail/form/trail1234567890"),
            fetch: vi.fn(),
            locals: { pb: { collection: vi.fn(() => ({ update })) } },
        } as unknown as RequestEvent;
    }

    function updatedData() {
        return update.mock.calls[0][1] as FormData;
    }
});
