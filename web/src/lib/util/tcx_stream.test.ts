import { tcx } from "#lib/vendor/toGeoJSON/toGeoJSON.js";
import { DOMParser } from "@xmldom/xmldom";
import type { Feature, FeatureCollection } from "geojson";
import { describe, expect, it } from "vitest";
import { fromFile, fromGeoJSON } from "./gpx_util.ts";
import { tcxToGeoJSON } from "./tcx_stream.ts";

const position = (lat: unknown, lon: unknown) =>
    `<Position><LatitudeDegrees>${lat}</LatitudeDegrees><LongitudeDegrees>${lon}</LongitudeDegrees></Position>`;
const point = (i: number, extra = "<AltitudeMeters>1000.5</AltitudeMeters>") =>
    `<Trackpoint><Time>2024-01-01T10:00:0${i}.000Z</Time>${position(47 + i / 100, 11 + i / 100)}${extra}<HeartRateBpm><Value>120</Value></HeartRateBpm></Trackpoint>`;
const track = (...indexes: number[]) => `<Track>${indexes.map((i) => point(i)).join("")}</Track>`;
const activities = (body: string) =>
    `<TrainingCenterDatabase xmlns="http://www.garmin.com/xmlschemas/TrainingCenterDatabase/v2"><Activities><Activity Sport="Biking"><Id>x</Id>${body}</Activity></Activities></TrainingCenterDatabase>`;
const lap = (body: string) => `<Lap StartTime="x"><TotalTimeSeconds>10</TotalTimeSeconds>${body}</Lap>`;

// The metadata carries the time of the conversion.
const gpxOf = (geojson: unknown) => fromGeoJSON(geojson as Feature | FeatureCollection).replace(/<metadata>[\s\S]*?<\/metadata>/, "");
const viaDom = (xml: string) => gpxOf(tcx(new DOMParser().parseFromString(xml, "text/xml")));
const viaStream = (xml: string) => gpxOf(tcxToGeoJSON(xml, () => {}));

describe("tcxToGeoJSON", () => {
    it.each([
        ["one lap", activities(lap(track(0, 1, 2)))],
        ["several laps", activities(lap(track(0, 1)) + lap(track(2, 3)))],
        ["several tracks in a lap", activities(lap(track(0, 1) + track(2, 3)))],
        ["a named lap", activities(lap(`<Name>Morning &amp; more</Name>${track(0, 1)}`))],
        ["a name in CDATA", activities(lap(`<Name><![CDATA[a < b]]></Name>${track(0, 1)}`))],
        ["a track with one point", activities(lap(track(0) + track(1, 2)))],
        ["no track at all", activities(lap(""))],
        ["a point without a position", activities(lap(`<Track>${point(0)}<Trackpoint><Time>x</Time></Trackpoint>${point(1)}</Track>`))],
        ["a point with a broken position", activities(lap(`<Track>${point(0)}<Trackpoint>${position("x", 11)}</Trackpoint>${point(1)}</Track>`))],
        ["points without altitude", activities(lap(`<Track>${point(0, "")}${point(1, "")}</Track>`))],
        ["a broken altitude", activities(lap(`<Track>${point(0, "<AltitudeMeters>x</AltitudeMeters>")}${point(1)}</Track>`))],
        ["a course", `<TrainingCenterDatabase><Courses><Course><Name>Loop</Name>${track(0, 1, 2)}</Course></Courses></TrainingCenterDatabase>`],
        ["a lap inside a course", `<TrainingCenterDatabase><Courses><Course><Name>Loop</Name><Lap><TotalTimeSeconds>1</TotalTimeSeconds></Lap>${track(0, 1)}</Course></Courses></TrainingCenterDatabase>`],
        ["activities and courses", `<TrainingCenterDatabase><Courses><Course>${track(4, 5)}</Course></Courses><Activities><Activity>${lap(track(0, 1))}</Activity></Activities></TrainingCenterDatabase>`],
        ["extensions and attributes", activities(lap(`<Track><Trackpoint a="1" b="2">${position(47, 11)}<Extensions><TPX xmlns="x"><Speed>2</Speed></TPX></Extensions></Trackpoint>${point(1)}</Track>`))],
    ])("converts %s like the DOM reader", (_name, xml) => {
        expect(viaStream(xml)).toBe(viaDom(xml));
    });

    it("reads values that a read chunk splits", () => {
        const xml = activities(lap(`<Track>${Array.from({ length: 3_000 }, (_, i) => point(i % 10)).join("")}</Track>`));

        expect(xml.length).toBeGreaterThan(3 * 64 * 1024);
        expect(viaStream(xml)).toBe(viaDom(xml));
    });

    it.each([
        ["an element inside a coordinate", point(1).replace("47.01<", "47.01<a/><")],
        ["an element inside the time", point(1).replace("T10:", "T1<a/>0:")],
        ["an element inside the altitude", point(1).replace("1000.5", "10<a/>00.5")],
    ])("reads %s like the DOM reader", (_name, broken) => {
        const xml = activities(lap(`<Track>${point(0)}${broken}${point(2)}</Track>`));

        expect(xml).toContain("<a/>");
        expect(viaStream(xml)).toBe(viaDom(xml));
    });

    it("is not confused by attributes named like object methods", () => {
        const xml = activities(lap(`<a hasOwnProperty="1" b="2" constructor="3" __proto__="4"/>${track(0, 1)}`));

        expect(viaStream(xml)).toBe(viaDom(xml));
    });

    it("stops reading when onPoint throws", () => {
        const seen: number[] = [];
        const read = () => tcxToGeoJSON(activities(lap(track(0, 1, 2, 3))), (count) => {
            seen.push(count);
            if (count === 2) throw new Error("limit");
        });

        expect(read).toThrow("limit");
        expect(seen).toEqual([1, 2]);
    });
});

describe("a file built to be expensive", () => {
    const points = (read: () => unknown) => gpxOf(read()).match(/<trkpt/g)?.length ?? 0;
    const stream = (xml: string) => () => tcxToGeoJSON(xml, () => {});
    const large = 300 * 1024;

    it.each([
        ["a long comment", activities(lap(`<!--${"x".repeat(large)}-->${track(0, 1)}`))],
        ["a long attribute value", activities(lap(`<a b="${"x".repeat(large)}"/>${track(0, 1)}`))],
        ["a long text", activities(lap(`<Notes>${"x".repeat(large)}</Notes>${track(0, 1)}`))],
        ["long CDATA", activities(lap(`<Notes><![CDATA[${"x".repeat(large)}]]></Notes>${track(0, 1)}`))],
        ["a long declaration", `<!DOCTYPE ${"x".repeat(large)}>${activities(lap(track(0, 1)))}`],
        ["a long processing instruction", `<?x ${"x".repeat(large)}?>${activities(lap(track(0, 1)))}`],
    ])("still reads a file with %s", (_name, xml) => {
        expect(points(stream(xml))).toBe(2);
    });

    it("stops at the end of the root, whatever follows it", () => {
        expect(points(stream(`${activities(lap(track(0, 1)))}${"<&x ".repeat(100_000)}`))).toBe(2);
    });

    it("gets past some recoverable errors", () => {
        expect(points(stream(activities(lap(`<Notes>${"& ".repeat(5_000)}</Notes>${track(0, 1)}`))))).toBe(2);
    });

    it.each([
        ["ampersands", `<Notes>${"&".repeat(100_000)}</Notes>`],
        ["attributes without a value", `<a ${"b ".repeat(100_000)}/>`],
        ["stray brackets", `<Notes>${"< ".repeat(100_000)}</Notes>`],
    ])("stops at too many %s", (_name, noise) => {
        expect(stream(activities(lap(noise + track(0, 1))))).toThrow("Too many errors");
    });

    it("refuses deep nesting, and leaves ordinary depth alone", () => {
        const nested = (depth: number) => activities(lap(`${"<a>".repeat(depth)}${"</a>".repeat(depth)}${track(0, 1)}`));

        expect(points(stream(nested(900)))).toBe(2);
        expect(stream(nested(1_000))).toThrow("nested too deeply");
    });
});

describe("fromFile with a TCX", () => {
    it("accepts a recording beyond what a DOM could hold", async () => {
        const trackpoint = `<Trackpoint><Time>2024-01-01T10:00:00.000Z</Time>${position(47, 11)}<AltitudeMeters>1000.5</AltitudeMeters><DistanceMeters>12.5</DistanceMeters><HeartRateBpm><Value>120</Value></HeartRateBpm><Cadence>80</Cadence><Extensions><ns3:TPX><ns3:Speed>2.5</ns3:Speed></ns3:TPX></Extensions></Trackpoint>`;
        const xml = activities(lap(`<Track>${trackpoint.repeat(20_000)}</Track>`)).replace("<TrainingCenterDatabase ", `<TrainingCenterDatabase xmlns:ns3="x" `);

        const { gpxData } = await fromFile(new Blob([xml]));

        expect(gpxData.match(/<trkpt/g)).toHaveLength(20_000);
    });

    it("is not stopped by elements and attributes it does not read", async () => {
        const noise = `<a b="" c="" d="" e="" f="" g="" h="" i=""/>`.repeat(200_000);
        const { gpxData } = await fromFile(new Blob([activities(lap(noise + track(0, 1)))]));

        expect(gpxData.match(/<trkpt/g)).toHaveLength(2);
    });
});

describe("a malformed file", () => {
    const good = activities(lap(`<Name>a</Name>${track(0, 1)}`));

    it.each([
        ["a bare ampersand", good.replace("<Name>a", "<Name>a & b")],
        ["text after the root", `${good}x`],
        ["an attribute without quotes", good.replace(`Sport="Biking"`, `Sport=Biking`)],
    ])("is read like the DOM reader with %s", (_name, xml) => {
        expect(viaStream(xml)).toBe(viaDom(xml));
    });

    it("keeps the points with an unknown entity", () => {
        const xml = good.replace("<Name>a", "<Name>a &nbsp; b");

        expect(viaStream(xml).match(/<trkpt/g)).toHaveLength(2);
    });

    it.each([
        ["a mismatched tag", good.replace("</Name>", "</Nam>")],
        ["a cut-off end", good.slice(0, good.length - 60)],
        ["a cut inside a tag", good.slice(0, good.indexOf("</Track>") + 3)],
        ["no XML at all", "TrainingCenterDatabase"],
    ])("is refused like by the DOM reader with %s", (_name, xml) => {
        expect(() => viaDom(xml)).toThrow();
        expect(() => viaStream(xml)).toThrow();
    });
});
