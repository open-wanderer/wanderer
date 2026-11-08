import { RecordIdSchema } from "#lib/models/api/base_schema.ts";
import { error, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/assets/{id}/file:
 *   get:
 *     summary: Get an asset file or photo preview
 *     description: >
 *       Streams a local asset file. Access follows the asset's view rules: its author, users with access to a
 *       linked target, visitors to public targets, and holders of a valid share token can read permitted assets.
 *       Local files return the original unless a thumbnail is requested. Every request rechecks access,
 *       including conditional requests. Private browser caches may retain a loaded photo for up to one hour
 *       after access is revoked; shared caches never store it.
 *     tags:
 *       - Assets
 *     parameters:
 *       - in: path
 *         name: id
 *         schema:
 *           type: string
 *           pattern: ^[a-z0-9]{15}$
 *         required: true
 *         description: Record ID.
 *       - in: query
 *         name: share
 *         schema:
 *           type: string
 *           pattern: ^[a-z0-9]{32}$
 *         description: Share token granting access to the linked target.
 *       - in: query
 *         name: thumb
 *         schema:
 *           type: string
 *           pattern: ^[0-9]+x[0-9]+[tbf]?$
 *           example: 300x300
 *         description: Thumbnail dimensions; optional t, b or f resize suffix.
 *       - in: query
 *         name: size
 *         schema:
 *           type: string
 *           pattern: ^[0-9]+x[0-9]+[tbf]?$
 *           example: 300x300
 *         description: Alias for thumb; thumb takes precedence when both are supplied.
 *       - in: header
 *         name: If-None-Match
 *         schema:
 *           type: string
 *         description: ETag from a previous response.
 *       - in: header
 *         name: If-Modified-Since
 *         schema:
 *           type: string
 *         description: Last-Modified value from a previous local file response.
 *     responses:
 *       200:
 *         description: Asset file or preview stream
 *         headers:
 *           ETag:
 *             schema:
 *               type: string
 *             description: Validator for conditional requests.
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       304:
 *         description: Not Modified after access validation; no response body
 *       499:
 *         description: Request cancelled while downloading the file; no response body
 *       400:
 *         description: Invalid asset ID, share token or image query
 *       404:
 *         description: Asset unavailable or access denied
 */
export async function GET(event: RequestEvent) {
    const safeParams = RecordIdSchema.safeParse(event.params);

    if (!safeParams.success) {
        throw error(400, "Invalid asset id");
    }

    const safeSearchParams = z.object({
        share: z.string().regex(/^[a-z0-9]{32}$/).optional(),
        thumb: z.string().regex(/^[0-9]+x[0-9]+[tbf]?$/).optional(),
        size: z.string().regex(/^[0-9]+x[0-9]+[tbf]?$/).optional(),
    }).safeParse(Object.fromEntries(event.url.searchParams));

    if (!safeSearchParams.success) {
        throw error(400, "Invalid asset file request");
    }

    const headers: HeadersInit = {};
    const token = event.locals.pb.authStore.token;
    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }
    for (const name of ["If-None-Match", "If-Modified-Since"]) {
        const value = event.request.headers.get(name);
        if (value) headers[name] = value;
    }

    const query = new URLSearchParams();
    if (safeSearchParams.data.share) {
        query.set("share", safeSearchParams.data.share);
    }
    const thumb = safeSearchParams.data.thumb ?? safeSearchParams.data.size;
    if (thumb) {
        query.set("thumb", thumb);
    }
    const assetURL = event.locals.pb.buildURL(
        `assets/${safeParams.data.id}/file${query.size ? `?${query}` : ""}`,
    );
    let response: Response;
    try {
        response = await event.fetch(assetURL, { headers, signal: event.request.signal });
    } catch (fetchError) {
        if (event.request.signal.aborted) return new Response(null, { status: 499 });
        throw fetchError;
    }

    return new Response(response.body, {
        headers: response.headers,
        status: response.status,
    });
}
