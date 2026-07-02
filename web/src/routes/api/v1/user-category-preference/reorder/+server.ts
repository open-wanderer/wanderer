import {
    UserCategoryPreferenceReorderSchema,
} from "$lib/models/api/category_preference_schema";
import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/user-category-preference/reorder:
 *   post:
 *     summary: Reorder current user's category preferences
 *     tags:
 *       - Categories
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/UserCategoryPreferenceReorderInput'
 *     responses:
 *       200:
 *         description: Reorder acknowledged
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               required:
 *                 - acknowledged
 *               properties:
 *                 acknowledged:
 *                   type: boolean
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const data = await event.request.json();
        const safeData = UserCategoryPreferenceReorderSchema.parse(data);
        const response = await event.locals.pb.send(
            "/category-preferences/reorder",
            {
                method: "POST",
                body: JSON.stringify(safeData),
                fetch: event.fetch,
                requestKey: null,
            },
        );

        return json(response);
    } catch (e) {
        return handleError(e);
    }
}
