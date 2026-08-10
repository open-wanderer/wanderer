import GPX from "$lib/models/gpx/gpx";
import { handleError } from "$lib/util/api_util";
import { fromFile } from "$lib/util/gpx_util";
import type { RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

/**
 * Rejects anything that is not a parseable GPX document with a 400.
 *
 * This endpoint is unauthenticated (`/api/v1/trail/convert` is absent from
 * `privateRoutes` in `authorization_util.ts`) and `/api/v1` is CSRF-exempt
 * (`hooks.server.ts`'s `csrf(['/api/v1'])`), so without this check a
 * cross-origin auto-submitting `enctype="text/plain"` form could drive a
 * top-level navigation whose response body is fully attacker-controlled and
 * served from the victim's origin under an active XML content type. Before
 * the endpoint became transcode-only it got this validation for free from
 * `gpx2trail`, which threw on unparseable content; that call is gone and must
 * not be reintroduced, so the parse is reused here as a pure
 * VALIDATOR - nothing derived from it is computed, persisted or returned.
 *
 * `GPX.parse` throws for non-XML input and for XML whose root element is not
 * `<gpx>`, which is exactly the acceptance boundary wanted here.
 */
function assertParsableGpx(gpxData: string): void {
    const invalid = new ClientResponseError({
        status: 400,
        response: { message: "Invalid GPX content" },
    });

    let parsed: unknown;
    try {
        parsed = GPX.parse(gpxData);
    } catch {
        throw invalid;
    }
    if (!parsed || typeof parsed !== "object") {
        throw invalid;
    }
}

/**
 * @swagger
 * /api/v1/trail/convert:
 *   post:
 *     summary: Transcode an uploaded file to GPX.
 *     description: >
 *       Accepts a GPX/KML/KMZ/TCX/FIT file (multipart), or a GPX string as a raw text body,
 *       and returns the equivalent GPX document. It does NOT compute or persist a trail -
 *       clients compute trail metrics themselves.
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *               name:
 *                 type: string
 *     responses:
 *       200:
 *         description: The transcoded GPX document.
 *         content:
 *           application/gpx+xml:
 *             schema:
 *               type: string
 *       400:
 *         description: >
 *           Bad Request - the body was missing/empty, or its content did not parse as a GPX
 *           document. The response body is never an echo of unvalidated input.
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        let gpxData: string = "";

        const contentType = event.request.headers.get("content-type") || "";

        if (contentType.includes("multipart/form-data")) {
            // 1. Handle File Upload
            const data = await event.request.formData();
            const fileBlob = data.get("file") as Blob | null;

            if (!fileBlob) {
                throw new ClientResponseError({ status: 400, response: { message: "Missing file field" } });
            }

            // Extract the text content from the file blob
            const parsed = await fromFile(fileBlob);
            gpxData = parsed.gpxData;
            // The `name` field is accepted-and-ignored rather than rejected as a 400: the
            // endpoint no longer names anything, but rejecting it would break older app
            // builds harder than necessary for no benefit.
        } else {
            // 2. Fallback: treat the raw body text as the GPX string directly.
            //
            // There is deliberately no `application/json` branch. One existed
            // (reading `body.gpx`/`body.gpxData`) from the commit that first
            // made the app call this endpoint, but no client ever sent JSON:
            // the app posts multipart FormData and the web transcodes
            // client-side. It was speculative generality, and it was never a
            // declared request media type in the generated OpenAPI spec, so
            // removing it narrows nothing that was published. A JSON request
            // now falls through here and is rejected by assertParsableGpx
            // below as unparsable GPX.
            gpxData = await event.request.text();
        }

        // Validate we actually got something
        if (!gpxData || !gpxData.trim().length) {
            throw new ClientResponseError({ status: 400, response: { message: "Empty GPX data" } });
        }

        // ...and that what we got is actually a GPX document, on every input
        // branch. Without this the handler is a verbatim echo of arbitrary
        // unauthenticated request bodies.
        assertParsableGpx(gpxData);

        return new Response(gpxData, {
            headers: {
                "Content-Type": "application/gpx+xml",
                // Defence in depth on top of the validation above: `*+xml` is
                // parsed as XML by browsers (XSLT processing instructions,
                // XHTML-namespace script), so make the body non-sniffable and
                // non-renderable as a top-level navigation.
                "X-Content-Type-Options": "nosniff",
                "Content-Disposition": "attachment; filename=\"track.gpx\"",
            },
        });
    } catch (e) {
        return handleError(e);
    }
}
