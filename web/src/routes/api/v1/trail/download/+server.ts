import { handleError } from "$lib/util/api_util";
import { DownloadError, fetchExternalFile } from "$lib/util/secure_fetch_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/trail/download:
 *   post:
 *     summary: Download file from URL
 *     description: Downloads a file from a URL and returns it as a blob
 *     tags:
 *       - Trails
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - url
 *             properties:
 *               url:
 *                 type: string
 *                 format: uri
 *     responses:
 *       200:
 *         description: File blob
 *         content:
 *           application/octet-stream:
 *             schema:
 *               type: string
 *               format: binary
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) {
        return json({ message: "Unauthorized" }, { status: 401 });
    }

    try {
        const data = await event.request.json();

        const { contentType, body } = await fetchExternalFile(data.url);

        return new Response(new Uint8Array(body), {
            headers: {
                "Content-Type": contentType ?? "application/octet-stream",
            },
        });
    } catch (e) {
        if (e instanceof DownloadError) {
            return json({ message: e.message }, { status: e.status });
        }
        return handleError(e);
    }
}
