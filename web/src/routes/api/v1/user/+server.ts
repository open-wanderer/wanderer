import { env } from '$env/dynamic/public';
import { UserCreateSchema } from '$lib/models/api/user_schema';
import type { User } from '$lib/models/user';
import { Collection, handleError } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/user:
 *   put:
 *     summary: Create user (sign up)
 *     tags:
 *       - Users
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/UserCreateInput'
 *     responses:
 *       201:
 *         description: User created
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       400:
 *         description: Bad Request - Invalid data
 *       403:
 *         description: Signup is disabled on this instance
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    if (env.PUBLIC_DISABLE_SIGNUP === "true") {
        return json({ message: "signup_disabled" }, { status: 403 })
    }

    try {

        const data = await event.request.json();
        const safeData = UserCreateSchema.parse(data);

        const r = await event.locals.pb.collection(Collection.users).create<User>(safeData)

        await event.locals.pb.collection('users').requestVerification(safeData.email);

        return json(r);
    } catch (e: any) {
        return handleError(e);
    }
}

