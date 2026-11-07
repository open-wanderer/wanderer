import { describe, expect, it, vi } from "vitest";
import GPX from "#lib/models/gpx/gpx.ts";
import Track from "#lib/models/gpx/track.ts";
import TrackSegment from "#lib/models/gpx/track-segment.ts";
import GPXWaypoint from "#lib/models/gpx/waypoint.ts";
import { Trail } from "#lib/models/trail.ts";
import { trails_show } from "#lib/stores/trail_store.ts";
import { DOMParser as XMLDOMParser } from "@xmldom/xmldom";
import JSZip from "jszip";
import type { AuthRecord } from "pocketbase";
import { applyGpxToForm, FileTooLargeError, fromFile, fromKMZ, summitLogGpxFields, trail2gpx, trailGpxFields } from "./gpx_util";
import { joinContiguousTrackSegments, routeAnchorNamespace, routeToStoredGpx, splitTrackSegmentsAtAnchors } from "./gpx_segment_util";

vi.mock("#lib/stores/trail_store.ts", () => ({ trails_show: vi.fn() }));

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
    it("merges legs with more points than a call may have arguments", () => {
        const long = Array.from({ length: 200_000 }, () => pt(47.2, 11.2));
        const gpx = new GPX({
            trk: [new Track({ trkseg: [segment([pt(47.1, 11.1), pt(47.2, 11.2)]), segment(long)] })],
        });

        joinContiguousTrackSegments(gpx);

        expect(gpx.trk?.[0].trkseg?.[0].trkpt?.length).toBe(200_001);
    }, 30_000);

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

    it("writes the author's email in the GPX 1.1 attribute form", async () => {
        const trail = new Trail("Alpine loop", { gpx_data: multiLegGpx });

        const exported = await trail2gpx(trail, { email: "me.myself@mail.host.org" } as unknown as AuthRecord);

        expect(exported).toMatch(/<email id="me\.myself" domain="mail\.host\.org"\/>/);
        expect(exported).not.toContain("@");
    });

    it("writes metadata in schema order", async () => {
        const exported = await trail2gpx(new Trail("Alpine loop", { gpx_data: multiLegGpx }), { email: "a@b.de" } as unknown as AuthRecord);

        const order = ["<name>", "<desc", "<author>", "<time>", "<keywords>"].map((tag) => exported.indexOf(tag));
        expect(order).toEqual([...order].sort((a, b) => a - b));
    });

    it("leaves the email out when the user has none", async () => {
        const exported = await trail2gpx(new Trail("Alpine loop", { gpx_data: multiLegGpx }));

        expect(exported).not.toContain("<email");
    });
});

describe("author email in a stored GPX", () => {
    const stored = (email: string) => `<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1"><metadata><author><name>A</name>${email}</author></metadata></gpx>`;

    it("rewrites the plain-text form to the attribute form", () => {
        expect(GPX.parse(stored("<email>a@b.de</email>")).toString()).toContain(`<email id="a" domain="b.de"/>`);
    });

    it("writes nothing the schema does not know", () => {
        expect(GPX.parse(stored("<email>a@b.de</email>")).toString()).not.toContain("_features");
    });

    it("keeps the attribute form", () => {
        expect(GPX.parse(stored(`<email id="a" domain="b.de"/>`)).toString()).toContain(`<email id="a" domain="b.de"/>`);
    });

    it("drops a plain-text value that is no address", () => {
        expect(GPX.parse(stored("<email>nobody</email>")).toString()).not.toContain("<email");
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

    // Sets the sizes in the local header and the zip directory to `size`.
    function forgeDeclaredSize(kmz: ArrayBuffer, size: number) {
        const bytes = new Uint8Array(kmz);
        const view = new DataView(kmz);
        for (let i = 0; i + 4 <= bytes.length; i++) {
            const signature = view.getUint32(i, true);
            if (signature === 0x04034b50) view.setUint32(i + 22, size, true);
            if (signature === 0x02014b50) view.setUint32(i + 24, size, true);
        }
        return kmz;
    }

    it("counts the inflated bytes instead of trusting the zip directory", async () => {
        const zip = new JSZip();
        zip.file("doc.kml", " ".repeat(5_000_000));
        const kmz = forgeDeclaredSize(await zip.generateAsync({ type: "arraybuffer", compression: "DEFLATE" }), 10);

        await expect(fromKMZ(kmz, 1_000)).rejects.toBeInstanceOf(FileTooLargeError);
    });

    it("stops inflating at the limit", async () => {
        // Incompressible, because JSZip can only stop between 16 KiB of input.
        const content = new Uint8Array(2_000_000);
        for (let i = 0; i < content.length; i += 65_536) crypto.getRandomValues(content.subarray(i, i + 65_536));
        const zip = new JSZip();
        zip.file("doc.kml", content);
        const kmz = await zip.generateAsync({ type: "arraybuffer", compression: "DEFLATE" });

        let inflated = 0;
        const entryPrototype = Object.getPrototypeOf(zip.file("doc.kml"));
        const internalStream = entryPrototype.internalStream;
        const spy = vi.spyOn(entryPrototype, "internalStream").mockImplementation(function (this: unknown, type: unknown) {
            return internalStream.call(this, type).on("data", (chunk: Uint8Array) => {
                inflated += chunk.length;
            });
        });

        try {
            await expect(fromKMZ(kmz, 1_000)).rejects.toBeInstanceOf(FileTooLargeError);
            await new Promise((resolve) => setTimeout(resolve, 100));
            expect(inflated).toBeLessThan(200_000);
        } finally {
            spy.mockRestore();
        }
    });

    it("keeps multibyte text that a chunk boundary splits", async () => {
        const name = `${"ä".repeat(40_000)}€`;
        const zip = new JSZip();
        zip.file("doc.kml", `<kml xmlns="http://www.opengis.net/kml/2.2"><Document><Placemark><name>${name}</name><LineString><coordinates>11,47 11.001,47.001</coordinates></LineString></Placemark></Document></kml>`);

        const gpx = await fromKMZ(await zip.generateAsync({ type: "arraybuffer", compression: "DEFLATE" }));

        expect(gpx).toContain(name);
    });
});

describe("server limits of fromFile", () => {
    const MiB = 1024 * 1024;
    const kml = (body: string) => `<kml xmlns="http://www.opengis.net/kml/2.2"><Document>${body}</Document></kml>`;
    const line = (coordinates: string) => kml(`<Placemark><LineString><coordinates>${coordinates}</coordinates></LineString></Placemark>`);
    const tcxFile = (body: string) => `<TrainingCenterDatabase><Activities><Activity><Lap><Track>${body}</Track></Lap></Activity></Activities></TrainingCenterDatabase>`;
    const gpx = (body: string) => `<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg>${body}</trkseg></trk></gpx>`;

    function fit(size: number) {
        const bytes = new Uint8Array(size);
        bytes[0] = 14;
        bytes.set(new TextEncoder().encode(".FIT"), 8);
        return bytes;
    }

    async function kmz(size: number) {
        const zip = new JSZip();
        const noise = new Uint8Array(size);
        for (let i = 0; i < size; i += 65_536) crypto.getRandomValues(noise.subarray(i, Math.min(i + 65_536, size)));
        zip.file("doc.kml", line("11,47 11.001,47.001"));
        zip.file("photo.jpg", noise);
        return zip.generateAsync({ type: "uint8array" });
    }

    it.each<[string, () => BlobPart | Promise<BlobPart>]>([
        ["a file over 25 MiB", () => " ".repeat(25 * MiB + 1)],
        ["a KML over 4 MiB", () => kml(" ".repeat(4 * MiB))],
        ["a KML with too many elements", () => kml("<a/>".repeat(400_001))],
        ["a TCX with too many points", () => tcxFile(`<Trackpoint><Position><LatitudeDegrees>47</LatitudeDegrees><LongitudeDegrees>11</LongitudeDegrees></Position></Trackpoint>`.repeat(60_001))],
        ["a KML with too many points", () => line("1,2 ".repeat(60_001))],
        ["a GPX with too many points", () => gpx(`<trkpt lat="1" lon="2"/>`.repeat(150_001))],
        ["a GPX with too many elements", () => gpx("<a/>".repeat(750_001))],
        ["a route-only GPX with too many points", () => `<gpx version="1.1"><rte>${`<rtept lat="1" lon="2"/>`.repeat(60_001)}</rte></gpx>`],
        ["a FIT over 8 MiB", () => fit(8 * MiB + 1) as BlobPart],
        ["a KMZ over 8 MiB", async () => (await kmz(8 * MiB + 1)) as BlobPart],
    ])("refuses %s with a 413", async (_name, content) => {
        const error = await fromFile(new Blob([await content()])).catch((e) => e);

        expect(error).toBeInstanceOf(FileTooLargeError);
        expect(error.status).toBe(413);
    });

    it("refuses a FIT with too many records", async () => {
        const definition = new Uint8Array([0x40, 0, 0, 20, 0, 3, 253, 4, 0x86, 0, 4, 0x85, 1, 4, 0x85]);
        const record = 13;
        const data = new Uint8Array(definition.length + 60_001 * record);
        data.set(definition);
        const file = new Uint8Array(14 + data.length + 2);
        const view = new DataView(file.buffer);
        file[0] = 14;
        file[1] = 0x10;
        view.setUint32(4, data.length, true);
        file.set(new TextEncoder().encode(".FIT"), 8);
        file.set(data, 14);

        await expect(fromFile(new Blob([file]))).rejects.toBeInstanceOf(FileTooLargeError);
    });

    it("accepts a file at the limits", async () => {
        const { gpxData } = await fromFile(new Blob([line("11,47 ".repeat(59_999) + "11.001,47.001")]));

        expect(gpxData).toContain("<trkpt");
    });

    it("does not limit a browser upload that the server would refuse", async () => {
        vi.resetModules();
        vi.doMock("$app/env", () => ({ browser: true }));
        vi.stubGlobal("DOMParser", XMLDOMParser);
        try {
            const browserUtil = await import("./gpx_util");
            const zip = new JSZip();
            zip.file("doc.kml", line("11,47 11.001,47.001") + " ".repeat(5 * MiB));

            await expect(browserUtil.fromKMZ(await zip.generateAsync({ type: "arraybuffer", compression: "DEFLATE" }))).resolves.toContain("<trkpt");
        } finally {
            vi.unstubAllGlobals();
            vi.doUnmock("$app/env");
            vi.resetModules();
        }
    });
});

describe("applyGpxToForm and moving_duration", () => {
    const file = (points: string) => new File([`<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg>${points}</trkseg></trk></gpx>`], "track.gpx");
    const twoPoints = `<trkpt lat="47" lon="11"/><trkpt lat="47.001" lon="11.001"/>`;

    it("clears it when a GPX replaces the track", async () => {
        const form = new FormData();
        form.set("gpx", file(`<trkpt lat="47" lon="11"/>`));

        await applyGpxToForm(form, trailGpxFields, false);

        expect(form.get("moving_duration")).toBe("0");
        expect(form.get("distance")).toBe("0");
    });

    it("keeps an explicit value", async () => {
        const form = new FormData();
        form.set("gpx", file(twoPoints));
        form.set("moving_duration", "5000");

        await applyGpxToForm(form, trailGpxFields, false);

        expect(form.get("moving_duration")).toBe("5000");
    });

    it("leaves it alone without a GPX", async () => {
        const form = new FormData();

        await applyGpxToForm(form, trailGpxFields, false);

        expect(form.has("moving_duration")).toBe(false);
    });

    it("does not add it to a summit log", async () => {
        const form = new FormData();
        form.set("gpx", file(twoPoints));

        await applyGpxToForm(form, summitLogGpxFields, false);

        expect(form.has("moving_duration")).toBe(false);
    });
});
