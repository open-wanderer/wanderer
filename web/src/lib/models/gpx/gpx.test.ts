import { describe, expect, it } from "vitest";
import GPX from "./gpx";
import Track from "./track";
import TrackSegment from "./track-segment";
import Waypoint from "./waypoint";
import { haversineDistance } from "./utils";

describe("GPX.getTotals — first point of a segment", () => {
  it("reports the full hop length of a 2-point segment instead of 0", () => {
    const a = waypointAt(47.0, 11.0);
    const b = waypointAt(47.001, 11.001);
    const gpx = gpxFromSegments([[a, b]]);

    // Pre-fix value was exactly 0 — the loop started at i = 1, so the only
    // point of the segment ever fed to metrics.addAndFilter() was `b`, which
    // is the metrics instance's very first call and only initializes its
    // anchors (no distance is added on that call).
    expect(gpx.features.distance).toBeCloseTo(hopMetres(a, b), 1);
    expect(gpx.features.distance).toBeCloseTo(134.592, 1);
  });
});

describe("GPX.getTotals — centroid and bounding box", () => {
  it("includes every point, including the geographic-extreme first point, in the bounding box", () => {
    const first = waypointAt(40.0, 10.0);
    const second = waypointAt(47.0, 11.0);
    const third = waypointAt(48.0, 12.0);
    const gpx = gpxFromSegments([[first, second, third]]);

    // Pre-fix, `first` (the segment's own first point, and also the
    // geographic extreme) was skipped by the i = 1 loop bound, so the
    // bounding box reported 47.0 / 11.0 instead of 40.0 / 10.0.
    expect(gpx.features.boundingBox.minLat).toBe(40.0);
    expect(gpx.features.boundingBox.minLon).toBe(10.0);
    expect(gpx.features.boundingBox.maxLat).toBe(48.0);
    expect(gpx.features.boundingBox.maxLon).toBe(12.0);

    expect(gpx.features.centroid.lat).toBeCloseTo(45.0, 6);
    expect(gpx.features.centroid.lon).toBeCloseTo(11.0, 6);
  });

  it("divides the centroid by exactly the number of points it summed", () => {
    const points = [
      waypointAt(40.0, 10.0),
      waypointAt(47.0, 11.0),
      waypointAt(48.0, 12.0),
    ];
    const gpx = gpxFromSegments([points]);

    const meanLat = points.reduce((sum, p) => sum + (p.$.lat ?? 0), 0) / points.length;
    const meanLon = points.reduce((sum, p) => sum + (p.$.lon ?? 0), 0) / points.length;

    expect(gpx.features.centroid.lat).toBeCloseTo(meanLat, 6);
    expect(gpx.features.centroid.lon).toBeCloseTo(meanLon, 6);
  });
});

describe("GPX.getTotals — multi-leg planner route", () => {
  it("reports the full polyline across both legs instead of dropping the opening hop", () => {
    // Shaped like valhalla_store.svelte.ts's insertIntoRoute() output: each
    // planner leg is its own TrackSegment, and the shared anchor point is
    // deliberately repeated as the next leg's first point.
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    const expectedTotal =
      hopMetres(leg1[0], leg1[1]) +
      hopMetres(leg1[1], leg1[2]) +
      hopMetres(leg2[0], leg2[1]) +
      hopMetres(leg2[1], leg2[2]);

    // Pre-fix value was 333.585 — the i = 1 loop bound dropped each
    // segment's own first point (leg1's opening hop and leg2's zero-length
    // anchor duplicate), losing one real hop's worth of distance.
    expect(gpx.features.distance).toBeCloseTo(expectedTotal, 1);
    expect(gpx.features.distance).toBeCloseTo(444.78, 1);
  });

  it("includes leg 1's opening point in the bounding box", () => {
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    expect(gpx.features.boundingBox.minLat).toBe(47.0);
  });
});

describe("GPX.getTotals — zero-point regression guard", () => {
  it("keeps the pre-existing sentinel behavior for a track with no points (not a fix)", () => {
    const gpx = gpxFromSegments([[]]);

    expect(Number.isNaN(gpx.features.centroid.lat)).toBe(true);
    expect(gpx.features.boundingBox.minLat).toBe(Infinity);
    expect(gpx.features.boundingBox.maxLat).toBe(-Infinity);
    expect(gpx.features.distance).toBe(0);
    expect(gpx.flatten()).toHaveLength(0);
  });
});

describe("GPX.getTotals — reports the raw accumulator", () => {
  it("reports the raw jitter-inflated sum, not the smoothed forward travel, for a jittery track", () => {
    // Single segment: forward hop (~20 m) then a jitter out-and-back
    // (~1 m each way) that never clears the 5 m threshold, repeated 5
    // times. The raw haversine sum over every consecutive pair (what
    // cumulativeDistance's last entry holds) is ~110.083 m — this is what
    // getTotals() reports as `distance`
    // (2026-08-01). The now-unreported smoothed accumulator, which held the
    // real forward travel with jitter suppressed, is ~100.075 m. Pre-33-01
    // (i = 1 loop bug), the reported value was 90.068 m — a different
    // defect entirely.
    const points = [waypointAt(47.0, 11.0)];
    let lat = 47.0;
    for (let i = 0; i < 5; i++) {
      lat += 0.00018;
      points.push(waypointAt(lat, 11.0));
      lat += 0.000009;
      points.push(waypointAt(lat, 11.0));
      lat -= 0.000009;
      points.push(waypointAt(lat, 11.0));
    }
    const gpx = gpxFromSegments([points]);

    expect(gpx.features.distance).toBeCloseTo(110.083, 0);
  });

  it("equals the last cumulativeDistance entry — same accumulator, by construction", () => {
    const points = [waypointAt(47.0, 11.0)];
    let lat = 47.0;
    for (let i = 0; i < 5; i++) {
      lat += 0.00018;
      points.push(waypointAt(lat, 11.0));
      lat += 0.000009;
      points.push(waypointAt(lat, 11.0));
      lat -= 0.000009;
      points.push(waypointAt(lat, 11.0));
    }
    const gpx = gpxFromSegments([points]);

    const rawTotal =
      gpx.features.cumulativeDistance[gpx.features.cumulativeDistance.length - 1];

    // Executable invariant, inverted (not deleted) now that smoothing is
    // superseded: addAndFilter pushes this.totalDistance onto
    // cumulativeDistance immediately after every totalDistance += call, so
    // the reported distance and the raw cumulative array's last entry are
    // provably the same accumulator — strict equality, not a closeness
    // matcher.
    expect(gpx.features.distance).toBe(rawTotal);
  });
});

describe("GPX.getTotals — cumulativeDistance index alignment", () => {
  it("index-aligns a 2-point segment with a leading 0 entry", () => {
    const a = waypointAt(47.0, 11.0);
    const b = waypointAt(47.001, 11.001);
    const gpx = gpxFromSegments([[a, b]]);

    expect(gpx.features.cumulativeDistance).toHaveLength(gpx.flatten().length);
    expect(gpx.features.cumulativeDistance).toHaveLength(2);
    expect(gpx.features.cumulativeDistance[0]).toBe(0);
    expect(gpx.features.cumulativeDistance[1]).toBeCloseTo(134.592, 1);
  });

  it("index-aligns the two-leg planner route, staying non-decreasing across the shared anchor", () => {
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);
    const cumulative = gpx.features.cumulativeDistance;

    expect(cumulative).toHaveLength(gpx.flatten().length);
    expect(cumulative).toHaveLength(6);
    expect(cumulative[0]).toBe(0);
    expect(cumulative[cumulative.length - 1]).toBeCloseTo(444.78, 1);

    for (let i = 1; i < cumulative.length; i++) {
      expect(cumulative[i]).toBeGreaterThanOrEqual(cumulative[i - 1]);
    }
  });

  it("stays empty for a track with no points", () => {
    const gpx = gpxFromSegments([[]]);

    expect(gpx.features.cumulativeDistance).toHaveLength(gpx.flatten().length);
    expect(gpx.features.cumulativeDistance).toEqual([]);
  });
});

describe("GPX.getTotals — smoothing does not regress the planner route", () => {
  it("still reports the full polyline distance when every hop clears the smoothing threshold", () => {
    // Every hop in this fixture exceeds the 5 m threshold, so raw and
    // smoothed totals coincide — this value is unaffected by the
    // supersession (the reported distance is now raw, but raw and smoothed
    // agree here) and stays at 33-01's baseline.
    const leg1 = [
      waypointAt(47.0, 11.0),
      waypointAt(47.001, 11.0),
      waypointAt(47.002, 11.0),
    ];
    const leg2 = [
      waypointAt(47.002, 11.0),
      waypointAt(47.003, 11.0),
      waypointAt(47.004, 11.0),
    ];
    const gpx = gpxFromSegments([leg1, leg2]);

    expect(gpx.features.distance).toBeCloseTo(444.78, 1);
  });
});

// An unparseable-but-non-empty <time> body used to become an
// `Invalid Date`, which is a truthy object — so getTotals()'s
// `startTime && endTime` guard passed and produced a NaN duration, while the
// Dart port reported 0 for the same document. Times are now parsed with the
// grammar Dart's DateTime.parse accepts.
describe("Waypoint.time — Dart-aligned <time> parsing", () => {
  const rejected: Array<[string, string]> = [
    ["a non-numeric body", "N/A"],
    ["a whitespace-only body", "   "],
    ["a legacy US format V8 accepts but Dart does not", "Jan 1 2024"],
    ["a slash-separated date V8 accepts but Dart does not", "2024/01/01"],
    ["a bare year", "2024"],
  ];

  for (const [label, raw] of rejected) {
    it(`leaves time undefined for ${label}`, () => {
      // @ts-expect-error xml2js hands the constructor a raw string here.
      expect(new Waypoint({ $: { lat: 47, lon: 11 }, time: raw }).time).toBeUndefined();
    });
  }

  it("still accepts a conforming ISO-8601 instant", () => {
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const wpt = new Waypoint({ $: { lat: 47, lon: 11 }, time: "2024-01-01T10:30:00Z" });
    expect(wpt.time?.toISOString()).toBe("2024-01-01T10:30:00.000Z");
  });

  it("reports a 0 duration - never NaN - when the start time is unparseable", () => {
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const start = new Waypoint({ $: { lat: 47.0, lon: 11.0 }, time: "N/A" });
    // @ts-expect-error xml2js hands the constructor a raw string here.
    const end = new Waypoint({ $: { lat: 47.001, lon: 11.001 }, time: "2024-01-01T10:30:00Z" });
    const gpx = gpxFromSegments([[start, end]]);

    expect(Number.isNaN(gpx.features.duration)).toBe(false);
    expect(gpx.features.duration).toBe(0);
  });
});

function waypointAt(lat: number, lon: number, ele?: number): Waypoint {
  return new Waypoint({ $: { lat, lon }, ele });
}

function gpxFromSegments(segments: Waypoint[][]): GPX {
  return new GPX({
    trk: [
      new Track({
        trkseg: segments.map(points => new TrackSegment({ trkpt: points })),
      }),
    ],
  });
}

function hopMetres(a: Waypoint, b: Waypoint): number {
  return haversineDistance(a.$.lat ?? 0, a.$.lon ?? 0, b.$.lat ?? 0, b.$.lon ?? 0);
}

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
});
