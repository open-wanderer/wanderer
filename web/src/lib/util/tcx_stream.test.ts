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
