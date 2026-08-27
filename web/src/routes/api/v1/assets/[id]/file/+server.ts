import { RecordIdSchema } from "$lib/models/api/base_schema";
import { error, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/assets/{id}/file:
 *   get:
 *     summary: Get remote asset file
 *     description: Streams a local or remote plugin-backed asset file when the authenticated user has access to the asset.
 *     tags:
 *       - Assets
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Asset file stream
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       400:
 *         description: Invalid asset id
 *       404:
 *         description: Asset not available
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
    const response = await event.fetch(assetURL, { headers });

    if (!response.ok) {
        throw error(response.status, "Asset not available");
    }

    return new Response(response.body, {
        headers: response.headers,
        status: response.status,
    });
}
