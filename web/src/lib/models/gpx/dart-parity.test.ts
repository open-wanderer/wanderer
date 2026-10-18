import { describe, expect, it } from "vitest";
import GPX from "./gpx";

// Inputs the shared corpus does not cover, where the web used to compute
// different metrics than the app's Dart reader (app/lib/vendor/gpx).
function gpx(points: string): string {
    return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg>${points}</trkseg></trk></gpx>`;
}

const a = `<trkpt lat="47.0000" lon="11.0000"><ele>1000</ele><time>2024-05-01T10:00:00Z</time></trkpt>`;
const b = `<trkpt lat="47.0010" lon="11.0010"><ele>1010</ele><time>2024-05-01T10:30:00Z</time></trkpt>`;
const c = `<trkpt lat="47.0020" lon="11.0020"><ele>1030</ele><time>2024-05-01T11:00:00Z</time></trkpt>`;

function totals(points: string) {
    return GPX.parse(gpx(points)).getTotals();
}

describe("GPX metrics on malformed input match the app", () => {
    const reference = totals(a + b + c);

    it.each([
        ["missing lat", `<trkpt lon="11.0015"><ele>1020</ele></trkpt>`],
        ["NaN lat", `<trkpt lat="NaN" lon="11.0015"><ele>1020</ele></trkpt>`],
        ["hex lat", `<trkpt lat="0x10" lon="11.0015"><ele>1020</ele></trkpt>`],
    ])("skips a point with a %s", (_name, bad) => {
        const result = totals(a + b + bad + c);

        expect(result.distance).toBeCloseTo(reference.distance, 6);
        expect(result.boundingBox).toEqual(reference.boundingBox);
        expect(result.elevationGain).toBeCloseTo(reference.elevationGain ?? 0, 6);
    });

    it("treats a hex elevation as no reading", () => {
        const hex = `<trkpt lat="47.0010" lon="11.0010"><ele>0x10</ele><time>2024-05-01T10:30:00Z</time></trkpt>`;
        const withoutEle = `<trkpt lat="47.0010" lon="11.0010"><time>2024-05-01T10:30:00Z</time></trkpt>`;

        expect(totals(a + hex + c).elevationGain).toBeCloseTo(totals(a + withoutEle + c).elevationGain ?? 0, 6);
        expect(totals(a + hex + c).elevationLoss).toBe(0);
    });

    it.each([
        ["an hour-only offset", "2024-05-01T12:00:00+02", "2024-05-01T13:00:00+02"],
        ["a comma fraction", "2024-05-01T10:00:00,5Z", "2024-05-01T11:00:00,5Z"],
        ["the basic format", "20240501T100000Z", "20240501T110000Z"],
    ])("reads times with %s", (_name, start, end) => {
        const first = `<trkpt lat="47.0000" lon="11.0000"><time>${start}</time></trkpt>`;
        const last = `<trkpt lat="47.0010" lon="11.0010"><time>${end}</time></trkpt>`;

        expect(totals(first + last).duration).toBe(3_600_000);
    });

    it("keeps zero coordinates", () => {
        const zero = `<trkpt lat="0" lon="0"><ele>0</ele></trkpt><trkpt lat="0.001" lon="0"><ele>0</ele></trkpt>`;

        expect(GPX.parse(gpx(zero)).trk![0].trkseg![0].trkpt).toHaveLength(2);
        expect(totals(zero).distance).toBeGreaterThan(100);
    });
});
