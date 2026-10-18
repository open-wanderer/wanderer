import GPX from "$lib/models/gpx/gpx";
import { handleError } from "$lib/util/api_util";
import { FileTooLargeError, fromFile } from "$lib/util/gpx_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

// Recorded tracks of a long day stay well below this; the whole body is held
// in memory and parsed, so it bounds what one request can cost.
const MAX_CONVERT_BYTES = 25 * 1024 * 1024;

function tooLarge() {
    return new ClientResponseError({ status: 413, response: { message: "File too large" } });
}

/**
 * Reads the request body, giving up as soon as it exceeds `limit` bytes
 * instead of buffering whatever the client sends.
 */
async function readLimitedBody(request: Request, limit: number): Promise<Uint8Array<ArrayBuffer>> {
    const declared = Number(request.headers.get("content-length"));
    if (Number.isFinite(declared) && declared > limit) {
        throw tooLarge();
    }
    if (!request.body) {
        return new Uint8Array();
    }

    const reader = request.body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    for (;;) {
        const { done, value } = await reader.read();
        if (done) {
            break;
        }
        size += value.byteLength;
        if (size > limit) {
            // Release rather than cancel: cancelling makes the Node adapter
            // destroy the socket, so the client would see a reset instead of
            // the 413. Node discards the rest of the body without keeping it.
            reader.releaseLock();
            throw tooLarge();
        }
        chunks.push(value);
    }

    const body = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
        body.set(chunk, offset);
        offset += chunk.byteLength;
    }
    return body;
}

/**
 * Rejects anything that is not a parseable GPX document with a 400.
 *
 * `/api/v1` is CSRF-exempt
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
 *       clients compute trail metrics themselves. Requires a logged-in user; the body may be
 *       at most 25 MB, and a KMZ document at most 100 MB once unzipped.
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
 *       401:
 *         description: Unauthorized
 *       413:
 *         description: The upload, or the document inside a KMZ, is too large
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        // Parsing an uploaded file is expensive, so only logged-in users may
        // ask for it; the app imports files only while logged in.
        if (!event.locals.pb.authStore.record) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        let gpxData: string = "";

        const contentType = event.request.headers.get("content-type") || "";
        const body = await readLimitedBody(event.request, MAX_CONVERT_BYTES);

        if (contentType.includes("multipart/form-data")) {
            // 1. Handle File Upload
            const data = await new Response(body, { headers: { "content-type": contentType } }).formData();
            const fileBlob = data.get("file");

            if (!(fileBlob instanceof Blob)) {
                throw new ClientResponseError({ status: 400, response: { message: "Missing file field" } });
            }

            // Extract the text content from the file blob
            try {
                const parsed = await fromFile(fileBlob);
                gpxData = parsed.gpxData;
            } catch (e) {
                if (e instanceof FileTooLargeError) {
                    throw tooLarge();
                }
                throw e;
            }
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
            gpxData = new TextDecoder().decode(body);
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
