import { handleError } from "$lib/util/api_util";
import { type RequestEvent } from "@sveltejs/kit";

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
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    const data = await event.request.json();

    try {
        const response = await event.fetch(data.url);
        const blob = await response.blob();

        const contentType = 'determine the content type here';

        return new Response(blob, {
            headers: {
                'Content-Type': contentType,
            },
        });
    } catch (e) {
        return handleError(e)
    }
}
