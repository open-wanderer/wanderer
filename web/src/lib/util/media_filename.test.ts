import { describe, expect, it } from "vitest";
import { assetCopyFilename, canonicalMediaFilename } from "./media_filename.ts";

describe("media filenames", () => {
    it.each([
        ["../my.trip", "image/jpeg", "my.trip.jpg"],
        ["photo.jpg", "image/webp", "photo.webp"],
        ["photo.jfif", "image/jpeg", "photo.jpg"],
        ["C:\\private\\holiday.jpg", "image/png", "holiday.png"],
        ["photo.HEIC", "image/jpeg", "photo.jpg"],
        ["photo.?", "image/jpeg", "photo.-.jpg"],
    ])("canonicalizes %s for %s", (name, contentType, expected) => {
        expect(canonicalMediaFilename(name, contentType)).toBe(expected);
    });
    it("preserves the provider filename and removes only the storage suffix as fallback", () => {
        const source = { id: "source", file: "trail_a_473eqqcflx.jpg", metadata: {} };
        expect(assetCopyFilename(source, "image/jpeg")).toBe("trail_a.jpg");
        expect(assetCopyFilename({ ...source, metadata: { remote: { filename: "provider.jpg" } } }, "image/webp")).toBe("provider.webp");
        expect(assetCopyFilename({ ...source, metadata: { original_filename: "original.jpg" } }, "image/jpeg")).toBe("trail_a.jpg");
    });
});
