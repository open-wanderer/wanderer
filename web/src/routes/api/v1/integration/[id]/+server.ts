import { IntegrationUpdateSchema } from "$lib/models/api/integration_schema";
import type { Integration } from "$lib/models/integration";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/integration/{id}:
 *   get:
 *     summary: Get integration
 *     tags:
 *       - Integrations
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Integration details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Integration'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<Integration>(event, Collection.integrations)
        return json(r)
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/integration/{id}:
 *   post:
 *     summary: Update integration
 *     tags:
 *       - Integrations
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/IntegrationUpdateInput'
 *     responses:
 *       200:
 *         description: Integration updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Integration'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await update<Integration>(event, IntegrationUpdateSchema, Collection.integrations)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/integration/{id}:
 *   delete:
 *     summary: Delete integration
 *     tags:
 *       - Integrations
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Integration deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.integrations)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
