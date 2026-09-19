import type { PhotoLibraryBounds } from "$lib/models/photo_library";

export function normalizePhotoLibraryBounds(bounds: PhotoLibraryBounds): PhotoLibraryBounds {
    const wrap = (longitude: number) => ((longitude + 180) % 360 + 360) % 360 - 180;
    const round = (coordinate: number) => Number(coordinate.toFixed(7));
    const fullWorld = bounds.east - bounds.west >= 360;
    return {
        west: fullWorld ? -180 : round(wrap(bounds.west)),
        south: round(Math.max(-90, bounds.south)),
        east: fullWorld ? 180 : round(wrap(bounds.east)),
        north: round(Math.min(90, bounds.north)),
    };
}

export function photoLibraryBoundsContain(bounds: PhotoLibraryBounds, lat: number, lon: number): boolean {
    return lat >= bounds.south && lat <= bounds.north && (
        bounds.west > bounds.east
            ? lon >= bounds.west || lon <= bounds.east
            : lon >= bounds.west && lon <= bounds.east
    );
}
