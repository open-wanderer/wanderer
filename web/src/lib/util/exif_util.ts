import ExifReader from "exifreader";
import type { ExpandedTags } from "exifreader";

export interface GPSCoordinates {
    lat: number;
    lon: number;
}

export interface PhotoExifMetadata {
    coordinates?: GPSCoordinates;
    takenAt?: string;
}

type ExifLoadOptions = {
    expanded: true;
    includeTags?: {
        gps?: true;
        exif?: string[];
    };
    excludeTags?: {
        mpf?: true;
    };
};

const GPS_OPTIONS: ExifLoadOptions = {
    expanded: true,
    includeTags: { gps: true },
    excludeTags: { mpf: true },
};

const PHOTO_METADATA_OPTIONS: ExifLoadOptions = {
    expanded: true,
    includeTags: {
        gps: true,
        exif: [
            "DateTimeOriginal",
            "DateTimeDigitized",
            "DateTime",
            "OffsetTimeOriginal",
            "OffsetTimeDigitized",
            "OffsetTime",
        ],
    },
    excludeTags: { mpf: true },
};

export async function extractGPSCoordinates(
    source: Blob | string | ArrayBuffer | SharedArrayBuffer,
): Promise<GPSCoordinates | undefined> {
    try {
        const tags = await loadExifTags(source, GPS_OPTIONS);
        return gpsCoordinatesFromExpandedTags(tags);
    } catch {
        return undefined;
    }
}

export async function extractPhotoExifMetadata(
    source: Blob | string | ArrayBuffer | SharedArrayBuffer,
): Promise<PhotoExifMetadata> {
    try {
        const tags = await loadExifTags(source, PHOTO_METADATA_OPTIONS);
        return {
            coordinates: gpsCoordinatesFromExpandedTags(tags),
            takenAt: takenAtFromExpandedTags(tags),
        };
    } catch {
        return {};
    }
}

async function loadExifTags(
    source: Blob | string | ArrayBuffer | SharedArrayBuffer,
    options: ExifLoadOptions,
): Promise<ExpandedTags> {
    if (typeof source === "string") {
        return ExifReader.load(source, options);
    }
    if (isBlobLike(source)) {
        return ExifReader.load(await source.arrayBuffer(), options);
    }
    return ExifReader.load(source, options);
}

function gpsCoordinatesFromExpandedTags(tags: ExpandedTags): GPSCoordinates | undefined {
    const lat = tags.gps?.Latitude;
    const lon = tags.gps?.Longitude;
    if (
        typeof lat !== "number" ||
        typeof lon !== "number" ||
        !Number.isFinite(lat) ||
        !Number.isFinite(lon)
    ) {
        return undefined;
    }
    return { lat, lon };
}

function takenAtFromExpandedTags(tags: ExpandedTags): string | undefined {
    const value =
        tagString(tags.exif?.DateTimeOriginal) ??
        tagString(tags.exif?.DateTimeDigitized) ??
        tagString(tags.exif?.DateTime);
    if (!value) {
        return undefined;
    }

    const offset =
        tagString(tags.exif?.OffsetTimeOriginal) ??
        tagString(tags.exif?.OffsetTimeDigitized) ??
        tagString(tags.exif?.OffsetTime);
    return parseExifDate(value, offset);
}

function tagString(tag: { value?: unknown; description?: string } | undefined): string | undefined {
    if (!tag) {
        return undefined;
    }
    const value = Array.isArray(tag.value) ? tag.value[0] : tag.value;
    if (typeof value === "string" && value.trim()) {
        return value.trim();
    }
    if (typeof tag.description === "string" && tag.description.trim()) {
        return tag.description.trim();
    }
    return undefined;
}

function parseExifDate(value: string, offset?: string): string | undefined {
    const match = value.match(/^(\d{4}):(\d{2}):(\d{2})[ T](\d{2}):(\d{2}):(\d{2})/);
    if (!match) {
        const fallback = new Date(value);
        return Number.isNaN(fallback.getTime()) ? undefined : fallback.toISOString();
    }

    const [, year, month, day, hour, minute, second] = match;
    if (year === "0000" || month === "00" || day === "00") {
        return undefined;
    }

    const isoDate = `${year}-${month}-${day}T${hour}:${minute}:${second}`;
    const normalizedOffset = normalizeExifOffset(offset);
    const date = normalizedOffset
        ? new Date(`${isoDate}${normalizedOffset}`)
        : new Date(
            Number(year),
            Number(month) - 1,
            Number(day),
            Number(hour),
            Number(minute),
            Number(second),
        );
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function normalizeExifOffset(offset?: string): string | undefined {
    if (!offset) {
        return undefined;
    }
    const match = offset.trim().match(/^([+-])(\d{2}):?(\d{2})$/);
    if (!match) {
        return undefined;
    }
    return `${match[1]}${match[2]}:${match[3]}`;
}

function isBlobLike(source: unknown): source is Blob {
    return (
        typeof source === "object" &&
        source !== null &&
        "arrayBuffer" in source &&
        typeof source.arrayBuffer === "function"
    );
}
