import type { APIToken } from '$lib/models/api_token';
import { create } from '$lib/util/api_util';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { APITokenCreateSchema } from "$lib/models/api/api_token_schema";

/**
 * @swagger
 * /api/v1/api-token:
 *   get:
 *     summary: List API tokens
 *     description: Retrieves a paginated list of API tokens with optional filtering and sorting
 *     tags:
 *       - API Tokens
 *     parameters:
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *           default: 1
 *       - in: query
 *         name: perPage
 *         schema:
 *           type: integer
 *           default: 30
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
 *         description: Success
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 page:
 *                   type: integer
 *                 perPage:
 *                   type: integer
 *                 totalItems:
 *                   type: integer
 *                 totalPages:
 *                   type: integer
 *                 items:
 *                   type: array
 *                   items:
 *                     $ref: '#/components/schemas/APIToken'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await list<APIToken>(event, Collection.api_tokens);

        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/api-token:
 *   put:
 *     summary: Create API token
 *     description: Creates a new API token
 *     tags:
 *       - API Tokens
 *     parameters:
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/APITokenInput'
 *     responses:
 *       201:
 *         description: Created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/APIToken'
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        const r = await create<APIToken>(event, APITokenCreateSchema, Collection.api_tokens)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}