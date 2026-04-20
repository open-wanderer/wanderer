import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/auth/confirm-reset:
 *   post:
 *     summary: Confirm password reset
 *     description: Confirms and applies a password reset with a valid token
 *     tags:
 *       - Authentication
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *     responses:
 *       200:
 *         description: Password reset completed
 *       400:
 *         description: Bad Request
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.json()
        const safeData = z.object({
            token: z.string(),
            password: z.string().min(8).max(72),
            passwordConfirm: z.string().min(8).max(72)
        }).refine(d => d.password === d.passwordConfirm).parse(data)
        const r = await event.locals.pb.collection('users').confirmPasswordReset(safeData.token, safeData.password, safeData.passwordConfirm);
        return json(r);
    } catch (e: any) {
        return handleError(e);
    }

}
