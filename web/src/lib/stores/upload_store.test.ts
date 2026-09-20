import { beforeEach, describe, expect, it, vi } from "vitest";
import { APIError } from "$lib/util/api_util";
import { processUploadQueue, uploadStore, type Upload } from "./upload_store.svelte";

beforeEach(() => {
    uploadStore.enqueuedUploads = [];
    uploadStore.completedUploads = [];
    uploadStore.uploading = false;
});

describe("upload duplicate ownership", () => {
    it.each([
        ["own-actor", "me"],
        ["other-actor", "other@remote.example"],
    ])("preserves actor %s and the complete author handle", async (author, domain) => {
        const duplicate = { id: "existing-trail", name: "Mountain trail", author, domain };
        const upload: Upload = {
            file: new File(["track"], "trail.gpx"),
            progress: 0,
            status: "enqueued",
            function: vi.fn().mockRejectedValue(new APIError(400, "Duplicate trail", duplicate)),
        };
        uploadStore.enqueuedUploads.push(upload);

        await processUploadQueue();

        expect(uploadStore.completedUploads).toHaveLength(1);
        expect(uploadStore.completedUploads[0]).toMatchObject({ status: "duplicate", duplicate });
        expect(uploadStore.enqueuedUploads).toHaveLength(0);
        expect(uploadStore.uploading).toBe(false);
    });

    it("keeps force upload available for a duplicate", async () => {
        const upload = vi.fn().mockResolvedValue({ id: "new-own-trail" });
        uploadStore.enqueuedUploads.push({
            file: new File(["track"], "trail.gpx"),
            progress: 0,
            status: "enqueued",
            function: upload,
        });

        await processUploadQueue(undefined, true);

        expect(upload).toHaveBeenCalledWith(expect.any(File), true, expect.any(Function));
        expect(uploadStore.completedUploads[0].status).toBe("success");
    });
});
