import { describe, expect, it } from "vitest";
import { normalizePhotoLibraryBounds, photoLibraryBoundsContain } from "./photo_library_bounds";

describe("photo library viewport", () => {
    it("normalizes a viewport in another world copy", () => {
        expect(normalizePhotoLibraryBounds({ west: 367, east: 369, south: 46, north: 48 }))
            .toEqual({ west: 7, east: 9, south: 46, north: 48 });
    });

    it("includes both sides of a viewport crossing the date line", () => {
        const bounds = normalizePhotoLibraryBounds({ west: 170, east: 190, south: -10, north: 10 });
        expect(bounds).toEqual({ west: 170, east: -170, south: -10, north: 10 });
        expect(photoLibraryBoundsContain(bounds, 0, 175)).toBe(true);
        expect(photoLibraryBoundsContain(bounds, 0, -175)).toBe(true);
        expect(photoLibraryBoundsContain(bounds, 0, 0)).toBe(false);
        expect(photoLibraryBoundsContain(bounds, 20, 175)).toBe(false);
    });

    it("keeps a zoomed-out whole-world viewport unrestricted in longitude", () => {
        const bounds = normalizePhotoLibraryBounds({ west: -200, east: 200, south: -90, north: 90 });
        expect(bounds).toEqual({ west: -180, east: 180, south: -90, north: 90 });
        expect(photoLibraryBoundsContain(bounds, 0, -180)).toBe(true);
        expect(photoLibraryBoundsContain(bounds, 0, 180)).toBe(true);
    });

    it("includes boundaries and excludes photos outside the viewport", () => {
        const bounds = { west: 7, east: 9, south: 46, north: 48 };
        expect(photoLibraryBoundsContain(bounds, 46, 7)).toBe(true);
        expect(photoLibraryBoundsContain(bounds, 48, 9)).toBe(true);
        expect(photoLibraryBoundsContain(bounds, 47, 9.1)).toBe(false);
        expect(photoLibraryBoundsContain(bounds, 45.9, 8)).toBe(false);
    });
});
