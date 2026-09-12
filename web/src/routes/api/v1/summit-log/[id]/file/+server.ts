/**
 * @swagger
 * /api/v1/summit-log/{id}/file:
 *   post:
 *     summary: Upload summit log file
 *     deprecated: true
 *     description: >
 *       Deprecated alias of `POST /api/v1/summit-log/form/{id}`, which accepts the same multipart body and is the endpoint to use.
 *       Kept for compatibility; behaves exactly like the form endpoint.
 *     tags:
 *       - Summit Logs
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
 *             $ref: '#/components/schemas/SummitLogUpdateInput'
 *     responses:
 *       200:
 *         description: Summit log updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/SummitLog'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export { POST } from "../../form/[id]/+server";
