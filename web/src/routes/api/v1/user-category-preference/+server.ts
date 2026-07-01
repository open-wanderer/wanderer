import {
    UserCategoryPreferenceUpsertSchema,
} from "$lib/models/api/category_preference_schema";
import type { UserCategoryPreference } from "$lib/models/category_preference";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/user-category-preference:
 *   get:
 *     summary: List current user's category preferences
 *     tags:
 *       - Categories
 *     responses:
 *       200:
 *         description: Category preferences for the authenticated user, or an empty list for anonymous requests
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/UserCategoryPreference'
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json([]);
        }

        const preferences = await event.locals.pb
            .collection(Collection.user_category_preferences)
            .getFullList<UserCategoryPreference>({
                filter: event.locals.pb.filter("user = {:user}", {
                    user: event.locals.user.id,
                }),
                requestKey: null,
            });

        return json(preferences);
    } catch (e) {
        return handleError(e);
    }
}

/**
 * @swagger
 * /api/v1/user-category-preference:
 *   put:
 *     summary: Create or update a category preference
 *     tags:
 *       - Categories
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/UserCategoryPreferenceUpsertInput'
 *     responses:
 *       200:
 *         description: Saved category preference
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/UserCategoryPreference'
 *       400:
 *         description: Bad Request
 *       401:
 *         description: Unauthorized
 *       500:
 *         description: Internal Server Error
 */
export async function PUT(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json({ message: "Unauthorized" }, { status: 401 });
        }

        const data = await event.request.json();
        const safeData = UserCategoryPreferenceUpsertSchema.parse(data);

        const payload = {
            ...safeData,
            user: event.locals.user.id,
        };

        let preference: UserCategoryPreference | undefined;
        try {
            preference = await event.locals.pb
                .collection(Collection.user_category_preferences)
                .getFirstListItem<UserCategoryPreference>(
                    event.locals.pb.filter("user = {:user} && category = {:category}", {
                        user: event.locals.user.id,
                        category: safeData.category,
                    }),
                    { requestKey: null },
                );
        } catch {
            preference = undefined;
        }

        const saved = preference?.id
            ? await event.locals.pb
                  .collection(Collection.user_category_preferences)
                  .update<UserCategoryPreference>(preference.id, payload, {
                      requestKey: null,
                  })
            : await event.locals.pb
                  .collection(Collection.user_category_preferences)
                  .create<UserCategoryPreference>(payload, {
                      requestKey: null,
                  });

        return json(saved);
    } catch (e) {
        return handleError(e);
    }
}
