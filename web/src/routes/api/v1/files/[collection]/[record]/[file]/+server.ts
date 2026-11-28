import { error, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/files/{collection}/{record}/{file}:
 *   get:
 *     summary: Download file
 *     description: Downloads a file from a record with optional thumbnail generation
 *     tags:
 *       - Files
 *     parameters:
 *       - in: path
 *         name: collection
 *         required: true
 *         schema:
 *           type: string
 *       - in: path
 *         name: record
 *         required: true
 *         schema:
 *           type: string
 *       - in: path
 *         name: file
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: thumb
 *         schema:
 *           type: string
 *       - in: query
 *         name: share
 *         schema:
 *           type: string
 *           pattern: ^[a-z0-9]{32}$
 *         description: Share token granting access to the record.
 *       - in: header
 *         name: If-None-Match
 *         schema:
 *           type: string
 *       - in: header
 *         name: If-Modified-Since
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: File download
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       304:
 *         description: Not Modified after access validation; no response body
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {

    const safeParams = z.object({
        collection: z.string(),
        record: z.string().length(15),
        file: z.string()
    }).safeParse(event.params)

    if (!safeParams.success) {
        throw Response.json({ message: safeParams.error }, { status: 400 })
    }

    const safeSearchParams = z.object({
        thumb: z.string().regex(/^[0-9]+x[0-9]+[tbf]?$/).optional(),
        share: z.string().regex(/^[a-z0-9]{32}$/).optional(),
    }).safeParse(Object.fromEntries(event.url.searchParams))

    if (!safeSearchParams.success) {
        throw Response.json({ message: safeSearchParams.error }, { status: 400 })
    }

    // PocketBase resolves legacy photos and current GPX files under its
    // current access rules. Fetch follows its same-origin 307 redirect;
    // a second BFF record/link lookup would duplicate work and diverge.
    const parts = [];
    parts.push("api");
    parts.push("files");
    parts.push(encodeURIComponent(safeParams.data.collection));
    parts.push(encodeURIComponent(safeParams.data.record));
    parts.push(encodeURIComponent(safeParams.data.file));

    let fileURL = event.locals.pb.buildURL(parts.join("/") + '?' + new URLSearchParams(safeSearchParams.data));

    const headers: HeadersInit = {};
    const token = event.locals.pb.authStore.token;
    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }
    for (const name of ["If-None-Match", "If-Modified-Since"]) {
        const value = event.request.headers.get(name);
        if (value) headers[name] = value;
    }

    try {
        const r = await event.fetch(fileURL, { headers, signal: event.request.signal })
        return new Response(r.body, { headers: r.headers, status: r.status });
    } catch (e: any) {
        throw error(500, e.message);
    }

}
