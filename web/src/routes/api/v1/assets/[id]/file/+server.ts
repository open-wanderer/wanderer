import { RecordIdSchema } from "$lib/models/api/base_schema";
import { error, type RequestEvent } from "@sveltejs/kit";

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

    const headers: HeadersInit = {};
    const token = event.locals.pb.authStore.token;
    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }

    const assetURL = event.locals.pb.buildURL(`assets/${safeParams.data.id}/file`);
    const response = await event.fetch(assetURL, { headers });

    if (!response.ok) {
        throw error(response.status, "Asset not available");
    }

    return new Response(response.body, {
        headers: response.headers,
        status: response.status,
    });
}
