import { describe, expect, it } from "vitest";
import { extractDevicePhotoExif } from "./exif_util";

async function deviceCoordinates(source: Blob | ArrayBuffer) {
    return (await extractDevicePhotoExif(source instanceof Blob ? source : new Blob([source]))).coordinates;
}

interface ExifEntry {
    tag: number;
    type: number;
    count: number;
    bytes: number[];
}

const ASCII = 2;
const LONG = 4;
const RATIONAL = 5;

function uint16(value: number) {
    return [value & 0xff, (value >> 8) & 0xff];
}

function uint32(value: number) {
    return [value & 0xff, (value >> 8) & 0xff, (value >> 16) & 0xff, (value >>> 24) & 0xff];
}

function ascii(tag: number, value: string): ExifEntry {
    const bytes = [...value].map((c) => c.charCodeAt(0)).concat(0);
    return { tag, type: ASCII, count: bytes.length, bytes };
}

function rationals(tag: number, values: [number, number][]): ExifEntry {
    return {
        tag,
        type: RATIONAL,
        count: values.length,
        bytes: values.flatMap(([numerator, denominator]) => [...uint32(numerator), ...uint32(denominator)]),
    };
}

function ifd(entries: ExifEntry[], offset: number) {
    const dataOffset = offset + 2 + entries.length * 12 + 4;
    const head = uint16(entries.length);
    const data: number[] = [];
    for (const entry of entries) {
        head.push(...uint16(entry.tag), ...uint16(entry.type), ...uint32(entry.count));
        if (entry.bytes.length <= 4) {
            head.push(...entry.bytes, ...new Array(4 - entry.bytes.length).fill(0));
        } else {
            head.push(...uint32(dataOffset + data.length));
            data.push(...entry.bytes);
        }
    }
    return [...head, ...uint32(0), ...data];
}

// A JPEG holding only an APP1 segment with the given Exif and GPS IFD entries.
function jpeg(exif: ExifEntry[], gps: ExifEntry[]) {
    const exifOffset = 8 + 2 + 2 * 12 + 4;
    const exifIfd = ifd(exif, exifOffset);
    const gpsOffset = exifOffset + exifIfd.length;
    const tiff = [
        0x49, 0x49, ...uint16(42), ...uint32(8),
        ...ifd([
            { tag: 0x8769, type: LONG, count: 1, bytes: uint32(exifOffset) },
            { tag: 0x8825, type: LONG, count: 1, bytes: uint32(gpsOffset) },
        ], 8),
        ...exifIfd,
        ...ifd(gps, gpsOffset),
    ];
    const app1 = [0x45, 0x78, 0x69, 0x66, 0, 0, ...tiff];
    const length = app1.length + 2;
    return new Uint8Array([0xff, 0xd8, 0xff, 0xe1, length >> 8, length & 0xff, ...app1, 0xff, 0xd9]).buffer;
}

const position = [
    ascii(0x0001, "N"),
    rationals(0x0002, [[46, 1], [30, 1], [0, 1]]),
    ascii(0x0003, "W"),
    rationals(0x0004, [[7, 1], [15, 1], [0, 1]]),
];

describe("extractDevicePhotoExif track placement", () => {
    it("reads the position and prefers the GPS timestamp, which is UTC", async () => {
        const photo = jpeg(
            [ascii(0x9003, "2026:10:03 08:00:00"), ascii(0x9011, "+02:00")],
            [...position, rationals(0x0007, [[14, 1], [6, 1], [335, 10]]), ascii(0x001d, "2026:10:03")],
        );

        const result = await deviceCoordinates(new Blob([photo]));

        expect(result?.lat).toBeCloseTo(46.5);
        expect(result?.lon).toBeCloseTo(-7.25);
        expect(result?.time?.toISOString()).toBe("2026-10-03T14:06:33.500Z");
    });

    it("falls back to the local time with its offset", async () => {
        const photo = jpeg([ascii(0x9003, "2026:10:03 10:06:33"), ascii(0x9011, "-04:00")], position);

        expect((await deviceCoordinates(photo))?.time?.toISOString()).toBe("2026-10-03T14:06:33.000Z");
    });

    it("keeps coordinates without a time for non-EXIF capture dates", async () => {
        const photo = jpeg([ascii(0x9003, "2026-10-03 16:06:33"), ascii(0x9011, "+02:00")], position);
        const result = await deviceCoordinates(photo);

        expect(result?.lat).toBeCloseTo(46.5);
        expect(result?.lon).toBeCloseTo(-7.25);
        expect(result?.time).toBeUndefined();
    });

    it("converts an offset across the previous UTC day", async () => {
        const photo = jpeg([ascii(0x9003, "2026:10:03 01:30:00"), ascii(0x9011, "+05:30")], position);

        expect((await deviceCoordinates(photo))?.time?.toISOString()).toBe("2026-10-02T20:00:00.000Z");
    });

    it.each([
        ["    :  :     :  :  ", "-04:00"],
        ["2026:10:03 10:06:33", "Z"],
    ])("keeps coordinates without a time for capture date %s and offset %s", async (date, offset) => {
        const result = await deviceCoordinates(jpeg([ascii(0x9003, date), ascii(0x9011, offset)], position));

        expect(result?.lat).toBeCloseTo(46.5);
        expect(result?.lon).toBeCloseTo(-7.25);
        expect(result?.time).toBeUndefined();
    });

    it("uses the first valid fallback offset", async () => {
        const photo = jpeg(
            [ascii(0x9003, "2026:10:03 01:30:00"), ascii(0x9010, "+05:30"), ascii(0x9012, "+01:00")],
            position,
        );

        expect((await deviceCoordinates(photo))?.time?.toISOString()).toBe("2026-10-03T00:30:00.000Z");
    });

    it.each(["   :  ", "+1:00", "+24:00", "+02:60"])("rejects ambiguous device offset %s while retaining untimed GPS", async (offset) => {
        const photo = jpeg([ascii(0x9003, "2026:10:03 10:06:33"), ascii(0x9011, offset)], position);
        const result = await deviceCoordinates(photo);
        expect(result?.lat).toBeCloseTo(46.5);
        expect(result?.time).toBeUndefined();
    });

    it("skips an invalid original offset to use a valid fallback offset", async () => {
        const photo = jpeg([ascii(0x9003, "2026:10:03 10:06:33"), ascii(0x9011, "+1:00"), ascii(0x9010, "+02:00")], position);
        expect((await deviceCoordinates(photo))?.time?.toISOString()).toBe("2026-10-03T08:06:33.000Z");
    });

    it("keeps device coordinates without treating a camera time without an offset as UTC", async () => {
        const result = await deviceCoordinates(jpeg([ascii(0x9003, "2026:10:03 10:06:33")], position));

        expect(result?.lat).toBeCloseTo(46.5);
        expect(result?.lon).toBeCloseTo(-7.25);
        expect(result?.time).toBeUndefined();
    });

    it.each(["0000:00:00", "2026:02:30", "2026:13:01"])("falls back from invalid GPS date %s to the original capture time", async (date) => {
        const photo = jpeg(
            [ascii(0x9003, "2026:10:03 10:06:33"), ascii(0x9011, "+02:00")],
            [...position, rationals(0x0007, [[0, 1], [0, 1], [0, 1]]), ascii(0x001d, date)],
        );
        expect((await deviceCoordinates(photo))?.time?.toISOString()).toBe("2026-10-03T08:06:33.000Z");
    });

    it("ignores photos without a position", async () => {
        const photo = jpeg([ascii(0x9003, "2026:10:03 10:06:33"), ascii(0x9011, "-04:00")], []);

        expect(await deviceCoordinates(photo)).toBeUndefined();
    });
});
