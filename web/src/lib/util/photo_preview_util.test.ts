import { describe, expect, it, vi } from "vitest";
import * as files from "./file_util";
import { photoFilePreview } from "./photo_preview_util";

describe("waypoint photo previews", () => {
    it("shares an individual preview across cards and repeated picker form updates", async () => {
        const reader = vi.spyOn(files, "readAsDataURLAsync").mockResolvedValue("data:photo");
        try {
            const photo = new File(["photo"], "shared.jpg");
            expect(await Promise.all([photoFilePreview(photo), photoFilePreview(photo)])).toEqual(["data:photo", "data:photo"]);
            expect(await photoFilePreview(photo)).toBe("data:photo");
            expect(reader).toHaveBeenCalledOnce();
        } finally { reader.mockRestore(); }
    });
});
