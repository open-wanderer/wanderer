import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it } from "vitest";
import { POST } from "./+server";

// The handler as left by Task 1 reads only `event.request` - no `event.fetch`, `locals` or
// `params` - so a plain object holding a `Request` is a sufficient stand-in for `RequestEvent`.
function makeEvent(request: Request): RequestEvent {
    return { request } as unknown as RequestEvent;
}

const gpxDoc = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer">
  <trk>
    <trkseg>
      <trkpt lat="47.0" lon="11.0"><ele>500</ele></trkpt>
      <trkpt lat="47.001" lon="11.001"><ele>510</ele></trkpt>
    </trkseg>
  </trk>
</gpx>`;

const kmlDoc = `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <Placemark>
      <LineString>
        <coordinates>
          11.0,47.0,500
          11.001,47.001,510
        </coordinates>
      </LineString>
    </Placemark>
  </Document>
</kml>`;

describe("POST /api/v1/trail/convert - raw-text branch", () => {
    // The byte-for-byte round trip is only the contract for a body that
    // ALREADY validated as a GPX document - see the content-validation suite
    // below for the bodies that never reach this path.
    it("returns a *valid* submitted GPX string unchanged with an application/gpx+xml content type", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: gpxDoc,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toBe(gpxDoc);
    });

    it("hardens the success response against being rendered as a top-level navigation", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: gpxDoc,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("x-content-type-options")).toBe("nosniff");
        expect(res.headers.get("content-disposition")).toContain("attachment");
    });
});

// Regression: the transcode-only rewrite dropped the endpoint's only
// content validation (previously a side effect of `gpx2trail` throwing), which
// turned an unauthenticated, CSRF-exempt route into a verbatim reflector of
// arbitrary request bodies under an active XML content type.
describe("POST /api/v1/trail/convert - content validation", () => {
    const rejected: Array<[string, string]> = [
        ["arbitrary plain text", "just some attacker-controlled text"],
        ["an HTML document", "<html><body><script>alert(1)</script></body></html>"],
        [
            "a well-formed XML document that is not GPX",
            "<?xml version=\"1.0\"?><kml><Document/></kml>",
        ],
        [
            "an XHTML payload carrying script",
            "<?xml version=\"1.0\"?><html xmlns=\"http://www.w3.org/1999/xhtml\"><script>alert(1)</script></html>",
        ],
        ["malformed XML claiming to be GPX", "<gpx><trk></gpx>"],
        ["a JSON blob", JSON.stringify({ hello: "world" })],
    ];

    for (const [label, body] of rejected) {
        it(`400s on ${label} submitted as a raw body, and does not echo it`, async () => {
            const request = new Request("http://localhost/api/v1/trail/convert", {
                method: "POST",
                body,
            });

            const res = await POST(makeEvent(request));

            expect(res.status).toBe(400);
            expect(await res.text()).not.toContain(body);
        });
    }

    // Guard preserved through the removal of the `application/json`
    // request branch: the endpoint must never echo an attacker-controlled
    // body, and an `application/json` content-type must not be a way around
    // that. With no JSON branch left, such a request falls through to the
    // raw-text path and is rejected there — this asserts the outcome, not the
    // route it took.
    it("400s on an attacker-controlled body sent with an application/json content-type, and does not echo it", async () => {
        const payload = JSON.stringify({ gpx: "<html><script>alert(1)</script></html>" });
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            headers: { "content-type": "application/json" },
            body: payload,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(400);
        expect(await res.text()).not.toContain("alert(1)");
    });
});

describe("POST /api/v1/trail/convert - multipart branch", () => {
    it("transcodes an uploaded .gpx file and returns it as application/gpx+xml", async () => {
        const formData = new FormData();
        formData.set("file", new Blob([gpxDoc], { type: "application/gpx+xml" }), "track.gpx");
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: formData,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toContain("<gpx");
    });

    it("transcodes an uploaded KML file into a GPX document - transcoding still happens server-side", async () => {
        const formData = new FormData();
        formData.set("file", new Blob([kmlDoc], { type: "application/vnd.google-earth.kml+xml" }), "track.kml");
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: formData,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toContain("<trkpt");
    });

    it("400s when no file field is present", async () => {
        const formData = new FormData();
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: formData,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(400);
    });
});

describe("POST /api/v1/trail/convert - empty body guard", () => {
    it("400s on an empty raw body", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: "",
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(400);
    });
});

describe("POST /api/v1/trail/convert - regression guard", () => {
    it("the 200 response body is not a Trail - it does not parse as JSON and contains neither `expand` nor `elevation_gain`", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: gpxDoc,
        });

        const res = await POST(makeEvent(request));
        const text = await res.text();

        expect(() => JSON.parse(text)).toThrow();
        expect(text).not.toContain("\"expand\"");
        expect(text).not.toContain("elevation_gain");
    });
});
