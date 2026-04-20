import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/auth/login:
 *   post:
 *     summary: Authenticate user
 *     description: Authenticates a user with email or username and password
 *     tags:
 *       - Authentication
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - password
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *               username:
 *                 type: string
 *               password:
 *                 type: string
 *     responses:
 *       200:
 *         description: User authenticated with auth token
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/User'
 *       400:
 *         description: Bad Request - Invalid credentials
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const safeData = z.object({
            email: z.string().email().optional(),
            username: z.string().optional(),
            password: z.string().min(8).max(72)
        }).refine(d => d.email !== undefined || d.username !== undefined).parse(data);


        const r = await event.locals.pb.collection('users').authWithPassword(safeData.email ?? safeData.username!, data.password);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }

}
