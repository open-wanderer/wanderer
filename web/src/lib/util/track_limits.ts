const MiB = 1024 * 1024;

// PocketBase's default for a file field, and what the gpx field of the trails
// collection is set to on a new instance.
export const DEFAULT_GPX_FIELD_LIMIT = 5 * MiB;

// What one file may cost on the server, where parsing runs in the same process
// as everything else and running out of heap kills it. Measured on a production
// build with --max-old-space-size=512; see the numbers in PR #1397. A file may
// be as large as the gpx field allows, and the rest grows with it: an admin who
// raises the field accepts what parsing such files costs. Browsers parse their
// own uploads, so only the KMZ bound applies to them.
function limitsFor(gpxFieldLimit: number) {
    const scale = Math.max(1, gpxFieldLimit / DEFAULT_GPX_FIELD_LIMIT);
    return {
        fileBytes: gpxFieldLimit,
        // JSZip keeps an object per zip entry, and an empty entry costs ~80 bytes.
        kmzBytes: 8 * MiB * scale,
        // Every "1,2 " in a KML becomes an array in toGeoJSON.
        kmlBytes: 4 * MiB * scale,
        fitBytes: 8 * MiB * scale,
        // Elements in the xmldom tree that KML and TCX are read from.
        domTags: 400_000 * scale,
        // Elements and points in xml2js' objects, which GPX is read into.
        gpxTags: 750_000 * scale,
        gpxPoints: 150_000 * scale,
        // Points that a conversion may hand to the GPX serializer.
        convertedPoints: 60_000 * scale,
    };
}

let limits = limitsFor(DEFAULT_GPX_FIELD_LIMIT);

export function serverLimits() {
    return limits;
}

export function setGpxFieldLimit(gpxFieldLimit: number) {
    limits = limitsFor(gpxFieldLimit);
}
