import type { User } from "$lib/models/user";
import { Collection, upload } from "$lib/util/api_util";
import { error, json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/user/{id}/file:
 *   post:
 *     summary: Upload user file
 *     description: Uploads a file (avatar) for a user
 *     tags:
 *       - Users
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
 *             type: object
 *             properties:
 *               file:
 *                 type: string
 *                 format: binary
 *     responses:
 *       200:
 *         description: File uploaded, user updated
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<User>(event, Collection.users);
        return json(r);
    } catch (e: any) {
        throw error(e.status, e)
    }
}