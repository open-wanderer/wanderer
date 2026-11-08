import { describe, expect, it, vi } from "vitest";
import JSZip from "jszip";
import { fetchPhotoForExport, getFileURL, isURL, isVideoURL, photoExportFilename } from "./file_util";

describe("file_util", () => {
    describe("photo export filenames", () => {
        it("keeps every remote and stored photo in a ZIP, including duplicate filenames", async () => {
            const urls = [
                "/api/v1/assets/asset0000000001/file",
                "https://a.example/api/v1/assets/asset0000000002/file?share=token",
                "/api/v1/files/assets/asset0000000003/photo.jpg",
                "/api/v1/files/assets/asset0000000004/photo.jpg",
                "/api/v1/assets/asset0000000001/file",
            ];
            const used = new Set<string>();
            const names = urls.map((url) => photoExportFilename(url, "image/jpeg", used));
            const zip = new JSZip();
            names.forEach((name, i) => zip.file(`photos/${name}`, `original-${i}`));
            const restored = await JSZip.loadAsync(await zip.generateAsync({ type: "uint8array" }));

            expect(names).toEqual(["asset0000000001.jpg", "asset0000000002.jpg", "photo.jpg", "photo-2.jpg", "asset0000000001-2.jpg"]);
            expect(Object.keys(restored.files).filter((name) => !restored.files[name].dir)).toHaveLength(urls.length);
            expect(await Promise.all(names.map((name) => restored.file(`photos/${name}`)!.async("string"))))
                .toEqual(urls.map((_, i) => `original-${i}`));
        });
        it("uses the downloaded media type for extensionless asset URLs", () => {
            expect(photoExportFilename("/api/v1/assets/asset0000000001/file", "image/png; charset=binary")).toBe("asset0000000001.png");
            expect(photoExportFilename("/api/v1/assets/asset0000000001/file", "video/mp4")).toBe("asset0000000001.mp4");
            expect(photoExportFilename("/api/v1/assets/asset0000000001/file", "image/bmp")).toBe("asset0000000001.bmp");
            expect(photoExportFilename("/api/v1/assets/asset0000000001/file", "image/x-icon")).toBe("asset0000000001.ico");
        });
        it("never converts an error response into an exported photo", async () => {
            const response = Response.json({ message: "Not found" }, { status: 404 });
            const read = vi.spyOn(response, "blob");
            const request = vi.fn().mockResolvedValue(response);
            await expect(fetchPhotoForExport("/api/v1/assets/asset0000000001/file", request)).rejects.toThrow("Photo download failed");
            expect(request).toHaveBeenCalledWith("/api/v1/assets/asset0000000001/file");
            expect(read).not.toHaveBeenCalled();
        });
    });

    describe("getFileURL", () => {
        const dummyRecord = {
            collectionId: "e864strfxo14pm4",
            id: "253ac8164a00ad3",
        };

        it("returns empty string when filename is empty", () => {
            expect(getFileURL(dummyRecord, "")).toBe("");
            expect(getFileURL(dummyRecord, undefined)).toBe("");
        });

        it("returns full external URL as is", () => {
            expect(getFileURL(dummyRecord, "https://example.com/photo.jpg")).toBe(
                "https://example.com/photo.jpg",
            );
        });

        it("constructs PocketBase file URL without thumb", () => {
            expect(getFileURL(dummyRecord, "photo.jpg")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/photo.jpg",
            );
        });

        it("appends thumb query parameter when provided", () => {
            expect(getFileURL(dummyRecord, "photo.jpg", "600x0")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/photo.jpg?thumb=600x0",
            );
            expect(getFileURL(dummyRecord, "avatar.png", "100x100")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/avatar.png?thumb=100x100",
            );
        });

        it("does not append thumb query parameter for video files", () => {
            expect(getFileURL(dummyRecord, "video.mp4", "600x0")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/video.mp4",
            );
        });

        it("still appends thumb for photos whose name contains a video extension", () => {
            expect(getFileURL(dummyRecord, "doggo.jpg", "300x300")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/doggo.jpg?thumb=300x300",
            );
            expect(getFileURL(dummyRecord, "mp4-comparison.png", "600x0")).toBe(
                "/api/v1/files/e864strfxo14pm4/253ac8164a00ad3/mp4-comparison.png?thumb=600x0",
            );
        });

        it("appends thumb query parameter to asset file paths and wanderer file URLs", () => {
            expect(getFileURL(dummyRecord, "/api/v1/files/c/r/photo.jpg", "600x0")).toBe(
                "/api/v1/files/c/r/photo.jpg?thumb=600x0",
            );
            expect(getFileURL(dummyRecord, "https://a.example/api/v1/files/c/r/photo.jpg", "600x0")).toBe(
                "https://a.example/api/v1/files/c/r/photo.jpg?thumb=600x0",
            );
        });

        it("does not append thumb query parameter to video asset file paths or URLs", () => {
            expect(getFileURL(dummyRecord, "/api/v1/files/c/r/clip.mp4", "600x0")).toBe(
                "/api/v1/files/c/r/clip.mp4",
            );
            expect(getFileURL(dummyRecord, "https://a.example/api/v1/files/c/r/clip.webm", "600x0")).toBe(
                "https://a.example/api/v1/files/c/r/clip.webm",
            );
        });
    });

    describe("isURL", () => {
        it("identifies valid http/https URLs", () => {
            expect(isURL("http://example.com")).toBe(true);
            expect(isURL("https://example.com/img.png")).toBe(true);
            expect(isURL("img.png")).toBe(false);
            expect(isURL("/local/path.jpg")).toBe(false);
        });
    });

    describe("isVideoURL", () => {
        it("identifies video extensions and data URIs", () => {
            expect(isVideoURL("trail_video.mp4")).toBe(true);
            expect(isVideoURL("clip.webm")).toBe(true);
            expect(isVideoURL("clip.ogg")).toBe(true);
            expect(isVideoURL("clip.ogv")).toBe(true);
            expect(isVideoURL("CLIP.MP4")).toBe(true);
            expect(isVideoURL("photo.jpg")).toBe(false);
            expect(isVideoURL("data:video/mp4;base64,...")).toBe(true);
            expect(isVideoURL("data:image/png;base64,...")).toBe(false);
        });

        it("matches the extension rather than a substring", () => {
            expect(isVideoURL("doggo.jpg")).toBe(false);
            expect(isVideoURL("mp4-comparison.png")).toBe(false);
            expect(isVideoURL("webm_vs_mp4.webp")).toBe(false);
        });

        it("ignores a query string or fragment", () => {
            expect(isVideoURL("/api/v1/files/c/r/clip.mp4?token=abc")).toBe(true);
            expect(isVideoURL("/api/v1/files/c/r/photo.jpg?thumb=600x0")).toBe(
                false,
            );
        });

        it("does not treat object URLs as videos", () => {
            expect(isVideoURL("blob:http://localhost/8f2a-4c11")).toBe(false);
        });
    });
});
