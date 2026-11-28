import { describe, expect, it } from "vitest";
import { ASSET_UPLOAD_ACCEPT, isSupportedAssetUpload } from "./asset_upload_util";

describe("asset upload formats", () => {
    it.each(["image/gif", "image/avif", "image/bmp", "video/quicktime"])("rejects %s even with a supported-looking filename", (type) => {
        expect(isSupportedAssetUpload({ name: "photo.jpg", type })).toBe(false);
        expect(ASSET_UPLOAD_ACCEPT.split(",")).not.toContain(type);
    });
    it.each(["image/jpeg", "image/png", "image/vnd.mozilla.apng", "image/webp", "image/svg+xml", "image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence", "video/mp4", "video/webm", "video/ogg"])("accepts %s", (type) => {
        expect(isSupportedAssetUpload({ name: "photo", type })).toBe(true);
    });
    it("recognizes supported extensions when the device provides no media type", () => {
        expect(isSupportedAssetUpload({ name: "PHOTO.HEIC", type: "" })).toBe(true);
        expect(isSupportedAssetUpload({ name: "photo.jpg", type: "application/octet-stream" })).toBe(true);
        expect(isSupportedAssetUpload({ name: "photo.gif", type: "" })).toBe(false);
        expect(isSupportedAssetUpload({ name: "photo.avif", type: "" })).toBe(false);
    });
});
