import { describe, expect, it } from "vitest";
import GPX from "./gpx";
import { routeSegmentMetrics } from "./gpx-metrics-computation";
import { gpx2trail } from "#lib/util/gpx_util.ts";
import { routeToStoredGpx, splitTrackSegmentsAtAnchors } from "#lib/util/gpx_segment_util.ts";

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

describe("coordinates", () => {
    it("reads integer-valued exponents in full", () => {
        const [point] = GPX.parse(gpx(`<trkpt lat="4.7E1" lon="1.1e1"/>`)).flatten();

        expect(point.$).toEqual({ lat: 47, lon: 11 });
    });

    it("still treats hex as no reading", () => {
        expect(GPX.parse(gpx(`<trkpt lat="0x10" lon="11"/>`)).flatten()).toHaveLength(0);
    });

    it.each([
        ["a latitude", `<trkpt lat="89" lon="0"/><trkpt lat="95.5" lon="0"/>`, `<trkpt lat="89" lon="0"/><trkpt lat="90" lon="0"/>`],
        ["a negative latitude", `<trkpt lat="-89" lon="0"/><trkpt lat="-95.5" lon="0"/>`, `<trkpt lat="-89" lon="0"/><trkpt lat="-90" lon="0"/>`],
        ["a longitude", `<trkpt lat="10" lon="179"/><trkpt lat="10" lon="190"/>`, `<trkpt lat="10" lon="179"/><trkpt lat="10" lon="-170"/>`],
        ["a negative longitude", `<trkpt lat="10" lon="-179"/><trkpt lat="10" lon="-190"/>`, `<trkpt lat="10" lon="-179"/><trkpt lat="10" lon="170"/>`],
    ])("measures %s out of range like the app", (_name, outOfRange, normalized) => {
        expect(totals(outOfRange).distance).toBeCloseTo(totals(normalized).distance, 6);
    });

    it("keeps the file's values in the bounds", () => {
        expect(totals(`<trkpt lat="89" lon="0"/><trkpt lat="95.5" lon="0"/>`).boundingBox.maxLat).toBe(95.5);
    });
});

describe("points without a position", () => {
    const noPosition = (time: string) => `<trkpt><time>${time}</time></trkpt>`;

    it("still bound the duration at the ends of a segment", () => {
        const points = noPosition("2024-05-01T09:59:59Z") + a + c + noPosition("2024-05-01T11:00:01Z");

        expect(totals(points).duration).toBe(3_602_000);
    });

    it("do not survive a copy of the segment", () => {
        const file = GPX.parse(gpx(noPosition("2024-05-01T09:59:59Z") + a + c));

        expect(new GPX(file).getTotals().duration).toBe(3_600_000);
    });

    it("are not written to a stored file", () => {
        expect(GPX.parse(gpx(noPosition("2024-05-01T09:59:59Z") + a + c)).toString()).not.toContain("fileTimes");
    });

    it("set the date from the file's first and last point", async () => {
        const { trail } = await gpx2trail(gpx(noPosition("2024-04-30T23:59:59Z") + a + c));

        expect(trail.date).toBe("2024-04-30");
    });
});

describe("waypoints", () => {
    const document = (wpt: string) => `<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">${wpt}<trk><trkseg>${a}${c}</trkseg></trk></gpx>`;

    it.each([
        ["no latitude", `<wpt lon="11"/>`],
        ["no longitude", `<wpt lat="47"/>`],
        ["a text position", `<wpt lat="north" lon="11"/>`],
    ])("discards one with %s", async (_name, wpt) => {
        const { trail } = await gpx2trail(document(wpt));

        expect(trail.expand?.waypoints_via_trail).toHaveLength(0);
    });

    it("keeps one at 0/0", async () => {
        const { trail } = await gpx2trail(document(`<wpt lat="0" lon="0"/>`));

        expect(trail.expand?.waypoints_via_trail).toHaveLength(1);
    });
});

describe("hasPositions", () => {
    const document = (body: string) => `<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">${body}</gpx>`;

    it.each([
        ["an empty file", document(""), false],
        ["only points without a position", gpx(`<trkpt lat="x" lon="11"/>`), false],
        ["only waypoints", document(`<wpt lat="47" lon="11"/>`), false],
        ["only invalid route points", document(`<rte><rtept lat="x" lon="11"/></rte>`), false],
        ["a track point", gpx(a), true],
        ["a route point", document(`<rte><rtept lat="47" lon="11"/></rte>`), true],
    ])("with %s", (_name, file, expected) => {
        expect(GPX.parse(file).hasPositions()).toBe(expected);
    });

    it("drops invalid route points when converting to a track", () => {
        const file = GPX.parse(document(`<rte><rtept lat="x" lon="11"/><rtept lat="47" lon="11"/></rte>`));

        file.convertRoutesToTracks();

        expect(file.flatten()).toHaveLength(1);
    });
});

describe("a drawn route with anchors", () => {
    const time = (minute: number) => `2024-05-01T10:${String(minute).padStart(2, "0")}:00Z`;
    const pt = (lat: number, minute: number, anchor = false) =>
        `<trkpt lat="${lat}" lon="11"><time>${time(minute)}</time>${anchor ? "<extensions><wanderer:anchor/></extensions>" : ""}</trkpt>`;
    const stored = `<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1" xmlns:wanderer="https://wanderer.to/xmlns/gpx/1"><trk><trkseg>` +
        pt(47, 0) + pt(47.001, 10) + pt(47.002, 30, true) + pt(47.003, 40) + `</trkseg></trk></gpx>`;

    it("measures the duration of its legs", () => {
        expect(GPX.parse(stored).getTotals().duration).toBe(20 * 60_000);
    });

    it("ignores a mark on the first or last point", () => {
        const marked = stored.replace(pt(47, 0), pt(47, 0, true)).replace(pt(47.003, 40), pt(47.003, 40, true)).replace(pt(47.002, 30, true), pt(47.002, 30));

        expect(GPX.parse(marked).getTotals().duration).toBe(40 * 60_000);
    });

    it("gives an upload the same duration as the editor's stored file", async () => {
        const { trail } = await gpx2trail(stored);
        const editor = GPX.parse(stored);
        splitTrackSegmentsAtAnchors(editor);

        expect(GPX.parse(routeToStoredGpx(editor)).getTotals().duration / 1000).toBe(trail.duration);
    });
});

describe("routeSegmentMetrics", () => {
    const legs = (points: string[]) => GPX.parse(gpx(points.join("</trkseg><trkseg>"))).trk![0].trkseg!;

    it("ends on the totals of getTotals", () => {
        const file = GPX.parse(gpx(`${a}${b}</trkseg><trkseg>${b}${c}`));
        const { cumulativeMetrics } = routeSegmentMetrics(file.trk![0].trkseg!);
        const totals = file.getTotals();

        expect(cumulativeMetrics.at(-1)).toEqual({ distance: totals.distance, elevationGain: totals.elevationGain, elevationLoss: totals.elevationLoss });
    });

    it("gives a 2-point first segment its distance", () => {
        expect(routeSegmentMetrics(legs([a + b, b + c])).segmentMetrics[0].distance).toBeGreaterThan(100);
    });

    it("counts a climb that the route ends on", () => {
        const { segmentMetrics } = routeSegmentMetrics(legs([a + b, b + c]));

        expect(segmentMetrics.reduce((sum, leg) => sum + leg.elevationGain, 0)).toBe(30);
    });

    it("never reports a negative leg", () => {
        const noisy = legs([a + b, b + a, a + c]);

        for (const leg of routeSegmentMetrics(noisy).segmentMetrics) {
            expect(leg.elevationGain).toBeGreaterThanOrEqual(0);
            expect(leg.elevationLoss).toBeGreaterThanOrEqual(0);
        }
    });

    it("has nothing to report without segments", () => {
        expect(routeSegmentMetrics([])).toEqual({ segmentMetrics: [], cumulativeMetrics: [] });
    });
});
