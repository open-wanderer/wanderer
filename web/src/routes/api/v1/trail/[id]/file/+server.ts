/**
 * @swagger
 * /api/v1/trail/{id}/file:
 *   post:
 *     summary: Upload trail file
 *     deprecated: true
 *     description: >
 *       Deprecated alias of `POST /api/v1/trail/form/{id}`, which accepts the same multipart body and is the endpoint to use.
 *       Kept for compatibility; behaves exactly like the form endpoint.
 *     tags:
 *       - Trails
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         multipart/form-data:
 *           schema:
 *             $ref: '#/components/schemas/TrailUpdateInput'
 *     responses:
 *       200:
 *         description: Trail updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Trail'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export { POST } from "../../form/[id]/+server";
