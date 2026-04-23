import { RecordIdSchema } from '$lib/models/api/base_schema';
import { UserUpdateSchema } from '$lib/models/api/user_schema';
import type { User } from '$lib/models/user';
import { Collection, handleError, remove, show } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/user/{id}:
 *   get:
 *     summary: Get user by ID
 *     tags:
 *       - Users
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
 *         description: User details
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        const r = await show<User>(event, Collection.users)

        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/user/{id}:
 *   post:
 *     summary: Update user
 *     description: Updates a user. Handles password changes and email change requests
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
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/UserUpdateInput'
 *     responses:
 *       200:
 *         description: User updated
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
    const data = await event.request.json()
    try {
        const params = event.params
        const safeParams = RecordIdSchema.parse(params);

        const safeData = UserUpdateSchema.parse(data);

        if (safeData.email && safeData.email != event.locals.pb.authStore.record!.email) {
            const r = await event.locals.pb.collection('users').requestEmailChange(safeData.email);
            event.locals.pb.authStore.record!.email = safeData.email;
        }
        const r = await event.locals.pb.collection('users').update<User>(safeParams.id, safeData)
        if (safeData.password) {
            const r = await event.locals.pb.collection('users').authWithPassword(safeData.email ?? safeData.username!, safeData.password);
            return json(r.record)
        } else {
            return json(r);
        }
    } catch (e: any) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/user/{id}:
 *   delete:
 *     summary: Delete user
 *     tags:
 *       - Users
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: User deleted
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.users)
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}
