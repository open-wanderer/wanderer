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
    it("returns the submitted GPX string unchanged with an application/gpx+xml content type", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            body: gpxDoc,
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toBe(gpxDoc);
    });
});

describe("POST /api/v1/trail/convert - JSON branch", () => {
    it("accepts the `gpx` key", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            headers: { "content-type": "application/json" },
            body: JSON.stringify({ gpx: gpxDoc }),
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toBe(gpxDoc);
    });

    it("accepts the `gpxData` key", async () => {
        const request = new Request("http://localhost/api/v1/trail/convert", {
            method: "POST",
            headers: { "content-type": "application/json" },
            body: JSON.stringify({ gpxData: gpxDoc }),
        });

        const res = await POST(makeEvent(request));

        expect(res.status).toBe(200);
        expect(res.headers.get("content-type")).toContain("application/gpx+xml");
        expect(await res.text()).toBe(gpxDoc);
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

    it("transcodes an uploaded KML file into a GPX document - PORT-05 proof transcoding still happens server-side", async () => {
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

describe("POST /api/v1/trail/convert - D-06 regression guard", () => {
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
