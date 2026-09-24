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

    it("ignores real comments before the root and inside text and track elements", () => {
        const xml = `<?xml version="1.0"?><!-- exported GPX -->` +
            `<gpx xmlns="${GPX_NS}"><metadata><name>Before<!-- ignored -->After</name></metadata>` +
            `<!-- track -->${track("")}<!-- end --></gpx>`;

        const gpx = GPX.parse(xml);

        expect(gpx.metadata?.name).toBe("BeforeAfter");
        expect(trackPointCount(gpx)).toBe(2);
        expect(gpx.toString()).not.toContain("#comment");
    });

    it("preserves comment-like text and namespace declarations in CDATA and attribute values", () => {
        const description = `<scr<!-- keep -->ipt>text</script> xmlns:ns3="${GPX_NS}"`;
        const creator = `text xmlns:ns3='${GPX_NS}'`;
        const extensions = `<extensions><ns3:hr>82</ns3:hr></extensions>`;
        const xml = `<gpx xmlns="${GPX_NS}" xmlns:ns3="${GARMIN_TPX_NS}" creator="${creator}">` +
            `<metadata><desc><![CDATA[${description}]]></desc></metadata>${track("", extensions)}</gpx>`;

        const gpx = GPX.parse(xml);

        expect(gpx.metadata?.desc).toBe(description);
        expect(gpx.$.creator).toBe(creator);
        expect(gpx.toString()).toContain("<ns3:hr>82</ns3:hr>");
        expect(GPX.parse(gpx.toString()).metadata?.desc).toBe(description);
    });

    it("does not interpret namespace declarations inside comments", () => {
        const extensions = `<extensions><ns3:hr>82</ns3:hr></extensions>`;
        const xml = `<gpx xmlns="${GPX_NS}" xmlns:ns3="${GARMIN_TPX_NS}">` +
            `<!-- xmlns:ns3="${GPX_NS}" -->${track("", extensions)}</gpx>`;

        expect(GPX.parse(xml).toString()).toContain("<ns3:hr>82</ns3:hr>");
    });

    it("accepts empty default namespaces without rewriting text or attribute contents", () => {
        const xml = `<gpx xmlns="" creator='text xmlns=""'>` +
            `<metadata xmlns=""><name>text xmlns=""</name><desc><![CDATA[text xmlns=""]]></desc></metadata>${track("")}</gpx>`;

        const gpx = GPX.parse(xml);

        expect(trackPointCount(gpx)).toBe(2);
        expect(gpx.$.xmlns).toBe(GPX_NS);
        expect(gpx.$.creator).toBe('text xmlns=""');
        expect(gpx.metadata?.name).toBe('text xmlns=""');
        expect(gpx.metadata?.desc).toBe('text xmlns=""');
    });

    it("accepts an empty GPX root", () => {
        expect(trackPointCount(GPX.parse('<gpx/>'))).toBe(0);
    });

    it("preserves valid replacement characters in text, CDATA and attributes", () => {
        const xml = '<gpx creator="GPS \uFFFD"><metadata><name>Trail \uFFFD</name>' +
            '<desc><![CDATA[Description \uFFFD]]></desc></metadata></gpx>';

        const gpx = GPX.parse(xml);

        expect(gpx.$.creator).toBe('GPS \uFFFD');
        expect(gpx.metadata?.name).toBe('Trail \uFFFD');
        expect(gpx.metadata?.desc).toBe('Description \uFFFD');
        expect(GPX.parse(gpx.toString()).metadata?.name).toBe('Trail \uFFFD');
    });

    it("preserves the server parser's support for undeclared extension prefixes", () => {
        const xml = `<gpx>${track('', '<extensions><gpxtpx:hr>82</gpxtpx:hr></extensions>')}</gpx>`;
        const gpx = GPX.parse(xml);

        expect(trackPointCount(gpx)).toBe(2);
        expect(gpx.toString()).toContain('<gpxtpx:hr>82</gpxtpx:hr>');
    });

    it.each([
        ['HTML entities', '<gpx><metadata><name>A&nbsp;&copy;B</name></metadata></gpx>', 'A\u00a0©B'],
        ['whitespace before declaration', '\n <?xml version="1.0"?><gpx><metadata><name>kept</name></metadata></gpx>', 'kept'],
        ['BOM', '\uFEFF<gpx><metadata><name>kept</name></metadata></gpx>', 'kept'],
        ['trailing text', '<gpx><metadata><name>kept</name></metadata></gpx>trailing', 'kept'],
        ['uppercase numeric reference', '<gpx><metadata><name>&#X41;</name></metadata></gpx>', 'A'],
    ])("preserves server import compatibility for %s", (_name, xml, expected) => {
        expect(GPX.parse(xml).metadata?.name).toBe(expected);
    });

    it("preserves the server parser's handling of duplicate attributes", () => {
        expect(GPX.parse('<gpx creator="first" creator="second"/>').$.creator).toBe('first');
    });

    it("passes comment-like declarations to the parser without rebuilding comments", () => {
        const xml = '<gpx><!<!-- removed -->--><metadata><name>kept</name></metadata>--></gpx>';
        expect(GPX.parse(xml).metadata?.name).toBe('kept');
    });

    it("respects local namespace declarations without stripping foreign extensions", () => {
        const xml = `<gpx xmlns:p="${GARMIN_TPX_NS}">` +
            `<p:rte xmlns:p="${GPX_NS}"><p:rtept lat="47" lon="8"/><p:rtept lat="47.01" lon="8.01"/></p:rte>` +
            track('', '<extensions><p:hr>82</p:hr></extensions>') + '</gpx>';
        const gpx = GPX.parse(xml);

        expect(gpx.rte).toHaveLength(1);
        expect(gpx.toString()).toContain('<p:hr>82</p:hr>');
    });

    it("keeps track points in document order when GPX prefixes are interleaved", () => {
        const xml = `<gpx xmlns="${GPX_NS}" xmlns:a="${GPX_NS}" xmlns:b="${GPX_NS}"><trk><trkseg>` +
            '<a:trkpt lat="47.01" lon="8"/><trkpt lat="47.02" lon="8"/>' +
            '<b:trkpt lat="47.03" lon="8"/><a:trkpt lat="47.04" lon="8"/>' +
            '<trkpt lat="47.05" lon="8"/></trkseg></trk></gpx>';

        const gpx = GPX.parse(xml);

        expect(gpx.flatten().map(point => point.$.lat)).toEqual([47.01, 47.02, 47.03, 47.04, 47.05]);
        expect(GPX.parse(gpx.toString()).flatten().map(point => point.$.lat)).toEqual([47.01, 47.02, 47.03, 47.04, 47.05]);
    });

    it.each([
        '<gpx><metadata><name>unterminated</metadata></gpx>',
        '<gpx><!-- unterminated</gpx>',
        '<gpx><metadata></gpx>',
        '<gpx><metadata><name>Tom & Jerry</name></metadata></gpx>',
        '<not-gpx/>',
        ''
    ])("preserves errors from the existing XML parser: %s", (xml) => {
        expect(() => GPX.parse(xml)).toThrow();
    });

    it("evaluates features lazily upon access", () => {
        const xml = `<?xml version="1.0"?><gpx version="1.1" creator="x" xmlns="${GPX_NS}">${track("")}</gpx>`;
        const gpx = GPX.parse(xml);

        // Verify private field is not populated before access
        expect((gpx as any)._features).toBeUndefined();

        // toGeoJSON does not trigger features calculation
        const geojson = gpx.toGeoJSON();
        expect(geojson.type).toBe("FeatureCollection");
        expect((gpx as any)._features).toBeUndefined();

        // Accessing features computes and caches it
        const features = gpx.features;
        expect(features).toBeDefined();
        expect(features.duration).toBe(60_000);
        expect(features.distance).toBeDefined();
        expect((gpx as any)._features).toBe(features);
    });
});
