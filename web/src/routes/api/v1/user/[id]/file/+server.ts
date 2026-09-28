import type { User } from "$lib/models/user";
import { Collection, handleError, upload } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

const fileFields = ["avatar"] as const;

/**
 * @swagger
 * /api/v1/user/{id}/file:
 *   post:
 *     summary: Upload user file
 *     description: >
 *       Replaces a user's avatar. The multipart field must be named `avatar`; a request without it is rejected with 400.
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
 *               avatar:
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
 *         description: Bad Request (no `avatar` field in the body)
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const r = await upload<User>(event, Collection.users, fileFields);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}