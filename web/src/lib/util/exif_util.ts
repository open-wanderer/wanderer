import ExifReader from "exifreader";
import type { ExpandedTags } from "exifreader";
import { convertGPSTimestampToDate } from "./date_util.ts";

export interface GPSCoordinates {
    lat: number;
    lon: number;
}

export interface GPSCoordinatesWithTime extends GPSCoordinates {
    time?: Date;
}

type ExifLoadOptions = {
    expanded: true;
    includeTags?: {
        gps?: true;
        exif?: string[];
        file?: string[];
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

const DEVICE_OPTIONS: ExifLoadOptions = {
    expanded: true,
    includeTags: {
        gps: true,
        file: ["FileType"],
        exif: [
            "GPSDateStamp",
            "GPSTimeStamp",
            "DateTimeOriginal",
            "OffsetTimeOriginal",
            "OffsetTimeDigitized",
            "OffsetTime",
        ],
    },
    excludeTags: { mpf: true },
};

// EXIF dates are the camera's local time, "YYYY:MM:DD HH:MM:SS".
const EXIF_DATE_TIME = /^(\d{4}):(\d{2}):(\d{2})[ T](\d{2}):(\d{2}):(\d{2})/;

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

export async function extractDevicePhotoExif(source: Blob): Promise<{
    coordinates?: GPSCoordinatesWithTime;
    fileType?: string;
}> {
    try {
        const tags = await loadExifTags(source, DEVICE_OPTIONS);
        const coordinates = gpsCoordinatesFromExpandedTags(tags);
        return {
            coordinates: coordinates ? { ...coordinates, time: exactTimeFromExpandedTags(tags) } : undefined,
            fileType: tags.file?.FileType?.value,
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

// Devices without a GPS fix may leave the date zero.
function gpsTimeFromExpandedTags(tags: ExpandedTags): string | undefined {
    const dateStamp = tagString(tags.exif?.GPSDateStamp);
    const date = dateStamp?.match(/^(\d{4}):(\d{2}):(\d{2})$/);
    if (!date || !isExifDay(date[1], date[2], date[3])) {
        return undefined;
    }
    return convertGPSTimestampToDate(dateStamp, gpsTimeStampValues(tags.exif?.GPSTimeStamp))?.toISOString();
}

function exactTimeFromExpandedTags(tags: ExpandedTags): Date | undefined {
    const gpsTime = gpsTimeFromExpandedTags(tags);
    if (gpsTime) return new Date(gpsTime);
    const original = tagString(tags.exif?.DateTimeOriginal);
    const offset = validExifOffset(tags);
    const time = original && offset ? parseExifDate(original, offset) : undefined;
    return time ? new Date(time) : undefined;
}

// exifreader reads the hours, minutes and seconds of GPSTimeStamp as
// [numerator, denominator] pairs.
function gpsTimeStampValues(tag: { value?: unknown } | undefined): number[] | undefined {
    const value = tag?.value;
    if (!Array.isArray(value)) {
        return undefined;
    }
    return value.map((part: unknown) =>
        Array.isArray(part) ? Number(part[0]) / Number(part[1]) : Number(part),
    );
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
    const match = value.match(EXIF_DATE_TIME);
    if (!match) {
        return undefined;
    }

    const [, year, month, day, hour, minute, second] = match;
    if (!isExifDay(year, month, day)) {
        return undefined;
    }

    const isoDate = `${year}-${month}-${day}T${hour}:${minute}:${second}`;
    const date = new Date(`${isoDate}${normalizeExifOffset(offset) ?? "Z"}`);
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

// Cameras without a set clock write zeros.
function isExifDay(year: string, month: string, day: string): boolean {
    return year !== "0000" && month !== "00" && day !== "00";
}

function validExifOffset(tags: ExpandedTags): string | undefined {
    for (const tag of [tags.exif?.OffsetTimeOriginal, tags.exif?.OffsetTimeDigitized, tags.exif?.OffsetTime]) {
        const offset = normalizeExifOffset(tagString(tag));
        if (offset) return offset;
    }
    return undefined;
}

function normalizeExifOffset(offset?: string): string | undefined {
    if (!offset) {
        return undefined;
    }
    const match = offset.trim().match(/^([+-])(\d{2}):?(\d{2})$/);
    if (!match || Number(match[2]) > 23 || Number(match[3]) > 59) {
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
