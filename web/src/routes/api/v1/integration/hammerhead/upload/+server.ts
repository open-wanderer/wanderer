import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/integration/hammerhead/upload:
 *   post:
 *     summary: Upload via Hammerhead integration
 *     description: Proxies file upload to backend Hammerhead integration
 *     tags:
 *       - Integrations
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             type: object
 *             required:
 *               - file
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *     responses:
 *       200:
 *         description: Upload result
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const formData = await event.request.formData();
        const file = formData.get("file");

        if (!(file instanceof Blob)) {
            return json({ message: "missing_file" }, { status: 400 });
        }

        const r = await event.locals.pb.send("/integration/hammerhead/upload", {
            method: "POST",
            body: formData,
            fetch: event.fetch,
        });
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
