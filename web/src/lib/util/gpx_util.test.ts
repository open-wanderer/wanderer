import { describe, expect, it, vi } from "vitest";
import GPX from "$lib/models/gpx/gpx";
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import GPXWaypoint from "$lib/models/gpx/waypoint";
import { Trail } from "$lib/models/trail";
import { trails_show } from "$lib/stores/trail_store";
import JSZip from "jszip";
import { FileTooLargeError, fromFile, fromKMZ, trail2gpx } from "./gpx_util";
import { joinContiguousTrackSegments, routeAnchorNamespace, routeToStoredGpx, splitTrackSegmentsAtAnchors } from "./gpx_segment_util";

vi.mock("$lib/stores/trail_store", () => ({ trails_show: vi.fn() }));

function pt(lat: number | string, lon: number | string) {
    return new GPXWaypoint({ $: { lat: lat as number, lon: lon as number } });
}

function segment(points: GPXWaypoint[], extensions?: string) {
    return new TrackSegment({ trkpt: points, extensions });
}

function coordinates(gpx: GPX) {
    return (gpx.trk ?? []).map((track) =>
        (track.trkseg ?? []).map((seg) =>
            (seg.trkpt ?? []).map((point) => [Number(point.$.lat), Number(point.$.lon)]),
        ),
    );
}

const multiLegGpx = `<gpx><trk>` +
    `<trkseg><trkpt lat="47.0" lon="11.0"/><trkpt lat="47.1" lon="11.1"/><trkpt lat="47.2" lon="11.2"/></trkseg>` +
    `<trkseg><trkpt lat="47.2" lon="11.2"/><trkpt lat="47.3" lon="11.3"/></trkseg>` +
    `<trkseg><trkpt lat="47.3" lon="11.3"/><trkpt lat="47.4" lon="11.4"/><trkpt lat="47.0" lon="11.0"/></trkseg>` +
    `</trk></gpx>`;

describe("joinContiguousTrackSegments", () => {
    it("merges meeting legs into one segment and keeps the closing point", () => {
        const gpx = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt("47.0", "11.0"), pt(47.1, 11.1), pt(47.2, 11.2)]),
                    segment([pt("47.2", "11.2"), pt(47.3, 11.3)]),
                    segment([pt("47.30", "11.30"), pt(47.4, 11.4), pt(47, 11)]),
                ],
            })],
        });

        joinContiguousTrackSegments(gpx);

        expect(coordinates(gpx)).toEqual([[
            [[47, 11], [47.1, 11.1], [47.2, 11.2], [47.3, 11.3], [47.4, 11.4], [47, 11]],
        ]]);
    });

    it("keeps segments apart when their endpoints differ", () => {
        const gpx = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt(1, 1), pt(2, 2)]),
                    segment([pt(3, 3), pt(4, 4)]),
                ],
            })],
        });

        joinContiguousTrackSegments(gpx);

        expect(coordinates(gpx)).toEqual([[[[1, 1], [2, 2]], [[3, 3], [4, 4]]]]);
    });

    it("leaves a single segment unchanged", () => {
        const gpx = new GPX({
            trk: [new Track({ trkseg: [segment([pt(1, 2), pt(3, 4), pt(5, 6)])] })],
        });
        const pointCount = gpx.trk?.[0].trkseg?.[0].trkpt?.length;

        joinContiguousTrackSegments(gpx);

        expect(gpx.trk?.[0].trkseg).toHaveLength(1);
        expect(gpx.trk?.[0].trkseg?.[0].trkpt).toHaveLength(pointCount!);
    });

    it("joins segments only inside the same track", () => {
        const gpx = new GPX({
            trk: [
                new Track({
                    name: "out",
                    trkseg: [
                        segment([pt(1, 1), pt(2, 2)]),
                        segment([pt("2", "2"), pt(3, 3)]),
                    ],
                }),
                new Track({
                    name: "back",
                    trkseg: [
                        segment([pt(3, 3), pt(4, 4)]),
                        segment([pt(4, 4), pt(5, 5)]),
                    ],
                }),
            ],
        });

        joinContiguousTrackSegments(gpx);

        expect(gpx.trk?.map((track) => track.name)).toEqual(["out", "back"]);
        expect(coordinates(gpx)).toEqual([
            [[[1, 1], [2, 2], [3, 3]]],
            [[[3, 3], [4, 4], [5, 5]]],
        ]);
    });

    it("does not absorb a segment that carries extensions", () => {
        const gpx = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt(1, 1), pt(2, 2)]),
                    segment([pt(2, 2), pt(3, 3)], "<extensions><line>1</line></extensions>"),
                    segment([pt(3, 3), pt(4, 4)]),
                ],
            })],
        });

        joinContiguousTrackSegments(gpx);

        expect(coordinates(gpx)).toEqual([
            [[[1, 1], [2, 2]], [[2, 2], [3, 3]], [[3, 3], [4, 4]]],
        ]);
    });

    it("skips an empty segment between meeting legs", () => {
        const gpx = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt(1, 1), pt(2, 2)]),
                    segment([]),
                    segment([pt("2.0", "2.0"), pt(3, 3)]),
                ],
            })],
        });

        joinContiguousTrackSegments(gpx);

        expect(coordinates(gpx)).toEqual([[[[1, 1], [2, 2], [3, 3]]]]);
    });
});

function anchors(gpx: GPX) {
    return (gpx.trk ?? []).map((track) =>
        (track.trkseg ?? []).flatMap((seg) =>
            (seg.trkpt ?? []).flatMap((point, index) =>
                point.extensions && typeof point.extensions === "object" && "wanderer:anchor" in point.extensions ? [index] : [],
            ),
        ),
    );
}

describe("route anchors", () => {
    it("marks the points where legs were joined", () => {
        const gpx = GPX.parse(multiLegGpx);

        joinContiguousTrackSegments(gpx);

        expect(anchors(gpx)).toEqual([[2, 3]]);
        expect((gpx.$ as Record<string, string>)["xmlns:wanderer"]).toBe(routeAnchorNamespace);
    });

    it("does not mark anything when no segments were joined", () => {
        const gpx = new GPX({
            trk: [new Track({ trkseg: [segment([pt(1, 1), pt(2, 2)]), segment([pt(3, 3), pt(4, 4)])] })],
        });

        joinContiguousTrackSegments(gpx);

        expect(anchors(gpx)).toEqual([[]]);
        expect((gpx.$ as Record<string, string>)["xmlns:wanderer"]).toBeUndefined();
    });

    it("keeps other extensions on a junction point", () => {
        const gpx = GPX.parse(`<gpx xmlns:ns3="http://www.garmin.com/xmlschemas/TrackPointExtension/v1"><trk>` +
            `<trkseg><trkpt lat="1" lon="1"/><trkpt lat="2" lon="2"><extensions><ns3:hr>90</ns3:hr></extensions></trkpt></trkseg>` +
            `<trkseg><trkpt lat="2" lon="2"/><trkpt lat="3" lon="3"/></trkseg>` +
            `</trk></gpx>`);

        joinContiguousTrackSegments(gpx);
        const exported = GPX.parse(gpx.toString());

        expect(exported.trk?.[0].trkseg).toHaveLength(1);
        expect(exported.trk?.[0].trkseg?.[0].trkpt?.[1].extensions).toEqual({ "ns3:hr": "90", "wanderer:anchor": "" });

        splitTrackSegmentsAtAnchors(exported);

        expect(exported.trk?.[0].trkseg?.[0].trkpt?.[1].extensions).toEqual({ "ns3:hr": "90" });
    });

    it("restores the legs of an exported route", () => {
        const gpx = GPX.parse(multiLegGpx);
        joinContiguousTrackSegments(gpx);
        const imported = GPX.parse(gpx.toString());

        splitTrackSegmentsAtAnchors(imported);

        expect(coordinates(imported)).toEqual(coordinates(GPX.parse(multiLegGpx)));
        expect(anchors(imported)).toEqual([[]]);
        expect((imported.$ as Record<string, string>)["xmlns:wanderer"]).toBeUndefined();
        const legs = imported.trk![0].trkseg!;
        expect(legs[1].trkpt![0]).not.toBe(legs[0].trkpt!.at(-1));
    });

    it("leaves a track without anchors unchanged", () => {
        const gpx = GPX.parse(multiLegGpx);

        splitTrackSegmentsAtAnchors(gpx);

        expect(coordinates(gpx)).toEqual(coordinates(GPX.parse(multiLegGpx)));
    });

    it("ignores anchors on the ends of a segment", () => {
        const gpx = GPX.parse(`<gpx><trk><trkseg>` +
            `<trkpt lat="1" lon="1"><extensions><wanderer:anchor/></extensions></trkpt>` +
            `<trkpt lat="2" lon="2"/>` +
            `<trkpt lat="3" lon="3"><extensions><wanderer:anchor/></extensions></trkpt>` +
            `</trkseg></trk></gpx>`);

        splitTrackSegmentsAtAnchors(gpx);

        expect(coordinates(gpx)).toEqual([[[[1, 1], [2, 2], [3, 3]]]]);
        expect(anchors(gpx)).toEqual([[]]);
    });
});

describe("routeToStoredGpx", () => {
    it("stores contiguous legs as one marked segment", () => {
        const stored = routeToStoredGpx(GPX.parse(multiLegGpx));
        const parsed = GPX.parse(stored);

        expect(parsed.trk).toHaveLength(1);
        expect(parsed.trk?.[0].trkseg).toHaveLength(1);
        expect(coordinates(parsed)).toEqual([[
            [[47, 11], [47.1, 11.1], [47.2, 11.2], [47.3, 11.3], [47.4, 11.4], [47, 11]],
        ]]);
        expect(anchors(parsed)).toEqual([[2, 3]]);
        expect(stored).toContain(`xmlns:wanderer="${routeAnchorNamespace}"`);
        expect((parsed.$ as Record<string, string>)["xmlns:wanderer"]).toBe(routeAnchorNamespace);
    });

    it("restores the legs when the stored route is loaded again", () => {
        const loaded = GPX.parse(routeToStoredGpx(GPX.parse(multiLegGpx)));

        splitTrackSegmentsAtAnchors(loaded);

        expect(coordinates(loaded)).toEqual(coordinates(GPX.parse(multiLegGpx)));
        expect(anchors(loaded)).toEqual([[]]);
        expect((loaded.$ as Record<string, string>)["xmlns:wanderer"]).toBeUndefined();
    });

    it("does not mutate the route it serializes", () => {
        const input = GPX.parse(multiLegGpx);
        const before = input.toString();

        routeToStoredGpx(input);

        expect(input.trk?.[0].trkseg).toHaveLength(3);
        expect(anchors(input)).toEqual([[]]);
        expect((input.$ as Record<string, string>)["xmlns:wanderer"]).toBeUndefined();
        expect(coordinates(input)).toEqual(coordinates(GPX.parse(multiLegGpx)));
        expect(input.toString()).toBe(before);
    });

    it("serializes legs built in memory without touching them", () => {
        const input = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt(1, 1), pt(2, 2)]),
                    segment([pt(2, 2), pt(3, 3)]),
                    segment([pt(3, 3), pt(4, 4)]),
                ],
            })],
        });

        const parsed = GPX.parse(routeToStoredGpx(input));

        expect(parsed.trk?.[0].trkseg).toHaveLength(1);
        expect(anchors(parsed)).toEqual([[1, 2]]);
        expect(input.trk?.[0].trkseg).toHaveLength(3);
    });

    it("keeps legs apart when their endpoints differ", () => {
        const input = new GPX({
            trk: [new Track({
                trkseg: [
                    segment([pt(1, 1), pt(2, 2)]),
                    segment([pt(2, 2), pt(3, 3)]),
                    segment([pt(3.0001, 3), pt(4, 4)]),
                ],
            })],
        });

        const parsed = GPX.parse(routeToStoredGpx(input));

        expect(parsed.trk?.[0].trkseg).toHaveLength(2);
        expect(coordinates(parsed)).toEqual([[
            [[1, 1], [2, 2], [3, 3]],
            [[3.0001, 3], [4, 4]],
        ]]);
        expect(anchors(parsed)).toEqual([[1]]);

        splitTrackSegmentsAtAnchors(parsed);

        expect(coordinates(parsed)).toEqual(coordinates(input));
    });

    it("serializes an empty route", () => {
        const stored = routeToStoredGpx(new GPX({ trk: [new Track({ trkseg: [] })] }));

        const parsed = GPX.parse(stored);

        expect(coordinates(parsed).flat(2)).toEqual([]);
        expect(stored).not.toContain("xmlns:wanderer");
    });
});

describe("trail2gpx", () => {
    it("exports a multi-leg route as one segment without fetching", async () => {
        const fetchSpy = vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("unexpected fetch"));
        const trail = new Trail("Alpine loop", { gpx_data: multiLegGpx });

        const exported = GPX.parse(await trail2gpx(trail));

        expect(exported.trk).toHaveLength(1);
        expect(exported.trk?.[0].name).toBe("Alpine loop");
        expect(exported.trk?.[0].trkseg).toHaveLength(1);
        expect(coordinates(exported)).toEqual([[
            [[47, 11], [47.1, 11.1], [47.2, 11.2], [47.3, 11.3], [47.4, 11.4], [47, 11]],
        ]]);
        expect(trails_show).not.toHaveBeenCalled();
        expect(fetchSpy).not.toHaveBeenCalled();
        fetchSpy.mockRestore();
    });
});

describe("fromFile", () => {
    const routeOnlyGpx = `<?xml version="1.0"?><gpx version="1.1" creator="GaiaGPS" xmlns="http://www.topografix.com/GPX/1/1">` +
        `<metadata><name>Tinker Cliffs</name></metadata>` +
        `<rte><name>Tinker Cliffs</name>` +
        `<rtept lat="37.4" lon="-80.0"><ele>400</ele></rtept>` +
        `<rtept lat="37.41" lon="-80.01"><ele>450</ele></rtept>` +
        `<rtept lat="37.42" lon="-80.02"><ele>520</ele></rtept>` +
        `</rte></gpx>`;

    it("stores a route-only file as a track", async () => {
        const original = new Blob([routeOnlyGpx]);

        const { gpxData, gpxFile } = await fromFile(original);

        expect(gpxFile).not.toBe(original);
        expect(await gpxFile.text()).toBe(gpxData);
        const stored = GPX.parse(gpxData);
        expect(stored.rte).toBeUndefined();
        expect(stored.trk?.[0].name).toBe("Tinker Cliffs");
        expect(coordinates(stored)).toEqual([[[[37.4, -80], [37.41, -80.01], [37.42, -80.02]]]]);
        expect(stored.trk?.[0].trkseg?.[0].trkpt?.map((point) => Number(point.ele))).toEqual([400, 450, 520]);
        expect(stored.metadata?.name).toBe("Tinker Cliffs");
        expect(stored.features.distance).toBeGreaterThan(0);
    });

    it("converts a route that uses a GPX namespace prefix", async () => {
        const prefixed = `<g:gpx xmlns:g="http://www.topografix.com/GPX/1/1"><g:rte>` +
            `<g:rtept lat="1" lon="1"/><g:rtept lat="2" lon="2"/></g:rte></g:gpx>`;

        const { gpxData } = await fromFile(new Blob([prefixed]));

        expect(coordinates(GPX.parse(gpxData))).toEqual([[[[1, 1], [2, 2]]]]);
    });

    it("keeps a file with a track as uploaded", async () => {
        const original = new Blob([multiLegGpx]);

        const { gpxData, gpxFile } = await fromFile(original);

        expect(gpxFile).toBe(original);
        expect(gpxData).toBe(multiLegGpx);
    });

    it("keeps a file with both a route and a track as uploaded", async () => {
        const content = `<gpx><rte><rtept lat="1" lon="1"/><rtept lat="2" lon="2"/></rte>` +
            `<trk><trkseg><trkpt lat="3" lon="3"/><trkpt lat="4" lon="4"/></trkseg></trk></gpx>`;
        const original = new Blob([content]);

        const { gpxFile } = await fromFile(original);

        expect(gpxFile).toBe(original);
    });
});

describe("fromKMZ", () => {
    it("refuses a document that unzips beyond the limit", async () => {
        const zip = new JSZip();
        zip.file("doc.kml", `<kml xmlns="http://www.opengis.net/kml/2.2">${" ".repeat(10_000)}</kml>`);
        const kmz = await zip.generateAsync({ type: "arraybuffer", compression: "DEFLATE" });

        await expect(fromKMZ(kmz, 1_000)).rejects.toBeInstanceOf(FileTooLargeError);
    });
});
