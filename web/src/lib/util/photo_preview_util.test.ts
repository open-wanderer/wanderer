import { describe, expect, it, vi } from "vitest";
import * as files from "./file_util";
import { photoFilePreview, waypointFilePreviews } from "./photo_preview_util";

describe("waypoint photo previews", () => {
    it("shares an individual preview across cards and repeated picker form updates", async () => {
        const reader = vi.spyOn(files, "readAsDataURLAsync").mockResolvedValue("data:photo");
        try {
            const photo = new File(["photo"], "shared.jpg");
            expect(await Promise.all([photoFilePreview(photo), photoFilePreview(photo)])).toEqual(["data:photo", "data:photo"]);
            expect(await waypointFilePreviews([photo])).toEqual(["data:photo"]);
            expect(reader).toHaveBeenCalledOnce();
        } finally { reader.mockRestore(); }
    });
    it("reads only the three shown files and reuses them after form object copies", async () => {
        const reader = vi.spyOn(files, "readAsDataURLAsync").mockImplementation(async (file) => `data:${file.name}`);
        try {
            const photos = Array.from({ length: 30 }, (_, i) => new File(["photo"], `photo-${i}.jpg`));
            expect(await waypointFilePreviews(photos)).toEqual(["data:photo-0.jpg", "data:photo-1.jpg", "data:photo-2.jpg"]);
            for (let i = 0; i < 5; i++) await waypointFilePreviews([...photos]);
            expect(reader).toHaveBeenCalledTimes(3);

            photos[2] = new File(["replacement"], "replacement.jpg");
            expect(await waypointFilePreviews(photos)).toEqual(["data:photo-0.jpg", "data:photo-1.jpg", "data:replacement.jpg"]);
            expect(reader).toHaveBeenCalledTimes(4);
        } finally {
            reader.mockRestore();
        }
    });
});

it("reuses a large preview while its File remains in the picker", async () => {
    const oversized = "data:image/jpeg;base64," + "a".repeat(17 << 20);
    const reader = vi.spyOn(files, "readAsDataURLAsync").mockResolvedValue(oversized);
    try {
        const photo = new File(["photo"], "large.jpg");
        expect(await photoFilePreview(photo)).toBe(oversized);
        expect(await photoFilePreview(photo)).toBe(oversized);
        expect(reader).toHaveBeenCalledOnce();
    } finally { reader.mockRestore(); }
});
