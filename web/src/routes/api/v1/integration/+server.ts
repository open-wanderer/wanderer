import { IntegrationCreateSchema } from "$lib/models/api/integration_schema";
import type { Integration } from "$lib/models/integration";
import { Collection, create, handleError, list } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/integration:
 *   get:
 *     summary: List integrations
 *     tags:
 *       - Integrations
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *       - in: query
 *         name: sort
 *         schema:
 *           type: string
 *       - in: query
 *         name: filter
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List of integrations
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/ListResult'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<Integration>(event, Collection.integrations);

        return json(r)
    } catch (e) {
        return handleError(e)
    }
}

/**
 * @swagger
 * /api/v1/integration:
 *   put:
 *     summary: Create integration
 *     tags:
 *       - Integrations
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/IntegrationInput'
 *     responses:
 *       201:
 *         description: Integration created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/Integration'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<Integration>(event, IntegrationCreateSchema, Collection.integrations)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}