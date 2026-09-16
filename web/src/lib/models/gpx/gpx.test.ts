import { describe, expect, it } from "vitest";
import GPX from "./gpx";

const GPX_NS = "http://www.topografix.com/GPX/1/1";
const GARMIN_TPX_NS = "http://www.garmin.com/xmlschemas/TrackPointExtension/v1";

function track(prefix: string, extensions: string = ""): string {
    const p = prefix;
    return `<${p}trk><${p}name>t</${p}name><${p}trkseg>` +
        `<${p}trkpt lat="47.0" lon="11.0"><${p}ele>100</${p}ele><${p}time>2023-01-01T00:00:00Z</${p}time>${extensions}</${p}trkpt>` +
        `<${p}trkpt lat="47.001" lon="11.001"><${p}ele>110</${p}ele><${p}time>2023-01-01T00:01:00Z</${p}time>${extensions}</${p}trkpt>` +
        `</${p}trkseg></${p}trk>`;
}

function trackPointCount(gpx: GPX): number {
    return gpx.trk?.reduce((sum, trk) => sum + (trk.trkseg?.reduce((s, seg) => s + (seg.trkpt?.length ?? 0), 0) ?? 0), 0) ?? 0;
}

describe("GPX.parse", () => {
    it("keeps Garmin extension prefixes through a parse/toString round trip", () => {
        // Garmin Connect exports bind the TrackPointExtension namespace to an arbitrary prefix (ns3, gpxtpx, ...).
        const extensions = `<extensions><ns3:TrackPointExtension><ns3:atemp>2.0</ns3:atemp><ns3:hr>82</ns3:hr></ns3:TrackPointExtension></extensions>`;
        const xml = `<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="Garmin Connect" xmlns="${GPX_NS}" xmlns:ns3="${GARMIN_TPX_NS}">${track("", extensions)}</gpx>`;

        const gpx = GPX.parse(xml);
        const out = gpx.toString();

        expect(trackPointCount(gpx)).toBe(2);
        expect(out.match(/<ns3:hr>82<\/ns3:hr>/g)).toHaveLength(2);
        expect(out.match(/<ns3:atemp>2.0<\/ns3:atemp>/g)).toHaveLength(2);
        expect(out).not.toContain("<hr>");
        expect(out).toContain(`xmlns:ns3="${GARMIN_TPX_NS}"`);
    });

    it("accepts a file that binds the GPX namespace to a prefix", () => {
        const extensions = `<g:extensions><gpxtpx:TrackPointExtension><gpxtpx:hr>90</gpxtpx:hr></gpxtpx:TrackPointExtension></g:extensions>`;
        const xml = `<?xml version="1.0"?><g:gpx version="1.1" creator="x" xmlns:g="${GPX_NS}" xmlns:gpxtpx="${GARMIN_TPX_NS}">${track("g:", extensions)}</g:gpx>`;

        const gpx = GPX.parse(xml);
        const out = gpx.toString();

        expect(trackPointCount(gpx)).toBe(2);
        expect(gpx.features.duration).toBe(60_000);
        expect(out).toContain("<trkpt ");
        expect(out).not.toContain("<g:trkpt");
        // the foreign prefix is not stripped even though the GPX prefix is
        expect(out.match(/<gpxtpx:hr>90<\/gpxtpx:hr>/g)).toHaveLength(2);
    });

    it("accepts a prefixed GPX 1.0 file", () => {
        const xml = `<g:gpx version="1.0" creator="x" xmlns:g="http://www.topografix.com/GPX/1/0">${track("g:")}</g:gpx>`;

        expect(trackPointCount(GPX.parse(xml))).toBe(2);
    });

    it("leaves foreign namespace prefixes untouched when no GPX prefix is bound", () => {
        const extensions = `<extensions><osmand:speed>1.2</osmand:speed><locus:activity>hike</locus:activity></extensions>`;
        const xml = `<gpx version="1.1" creator="x" xmlns="${GPX_NS}" xmlns:osmand="https://osmand.net" xmlns:locus="http://www.locusmap.eu">${track("", extensions)}</gpx>`;

        const out = GPX.parse(xml).toString();

        expect(out).toContain("<osmand:speed>1.2</osmand:speed>");
        expect(out).toContain("<locus:activity>hike</locus:activity>");
    });

    it("keeps a default-namespace override on extension elements", () => {
        const extensions = `<extensions><TrackPointExtension xmlns="${GARMIN_TPX_NS}"><hr>90</hr></TrackPointExtension></extensions>`;
        const xml = `<gpx version="1.1" creator="x" xmlns="${GPX_NS}">${track("", extensions)}</gpx>`;

        const out = GPX.parse(xml).toString();

        expect(out).toContain(`<TrackPointExtension xmlns="${GARMIN_TPX_NS}">`);
    });
});
