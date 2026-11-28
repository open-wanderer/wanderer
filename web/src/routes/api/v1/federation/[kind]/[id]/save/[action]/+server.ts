import { RecordIdSchema } from "#lib/models/api/base_schema.ts";
import { handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/federation/{kind}/{id}/save/{action}:
 *   post:
 *     summary: Begin, renew or complete a federated content save
 *     description: Requires edit permission on the locally authored target. Begin renews a bounded save lease; complete announces the pending saved state after uploads and photo links finish.
 *     tags: [Federation]
 *     parameters:
 *       - in: path
 *         name: kind
 *         required: true
 *         schema: { type: string, enum: [trail, summit-log] }
 *       - in: path
 *         name: id
 *         required: true
 *         schema: { type: string }
 *       - in: path
 *         name: action
 *         required: true
 *         schema: { type: string, enum: [begin, complete] }
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required: [batchId]
 *             properties:
 *               batchId: { type: string, minLength: 16, maxLength: 128 }
 *     responses:
 *       200:
 *         description: Save lease updated or target does not federate
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               required: [status]
 *               properties:
 *                 status: { type: string, enum: [ok, not_federated] }
 *       400: { description: 'Invalid target, action or batch identifier' }
 *       401: { description: Authentication required }
 *       403: { description: Edit permission required or target author is not local }
 *       404: { description: Target not found }
 *       429: { description: Too many concurrent saves }
 */
export async function POST(event: RequestEvent) {
    if (!event.locals.user) return Response.json({ message: "Unauthorized" }, { status: 401 });
    try {
        const { id } = RecordIdSchema.parse(event.params);
        const { kind, action } = event.params;
        if ((kind !== "trail" && kind !== "summit-log") || (action !== "begin" && action !== "complete")) {
            return Response.json({ message: "Invalid federation save action" }, { status: 400 });
        }
        const result = await event.locals.pb.send(`/federation/${kind}/${id}/save/${action}`, {
            method: "POST",
            body: await event.request.json(),
            fetch: event.fetch,
            requestKey: null,
        });
        return Response.json(result, { headers: { "Cache-Control": "private, no-store" } });
    } catch (error) {
        return handleError(error);
    }
}
