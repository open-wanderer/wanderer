import { UserSubcategoryPreferenceUpsertSchema } from "$lib/models/api/subcategory_preference_schema";
import type { UserSubcategoryPreference } from "$lib/models/subcategory_preference";
import { Collection, handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

/**
 * @swagger
 * /api/v1/user-subcategory-preference:
 *   get:
 *     summary: List current user's subcategory preferences
 *     tags:
 *       - Categories
 *     responses:
 *       200:
 *         description: Subcategory preferences for the authenticated user, or an empty list for anonymous requests
 *         content:
 *           application/json:
 *             schema:
 *               type: array
 *               items:
 *                 $ref: '#/components/schemas/UserSubcategoryPreference'
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return json([]);
        }

        const preferences = await event.locals.pb
            .collection(Collection.user_subcategory_preferences)
            .getFullList<UserSubcategoryPreference>({
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
 * /api/v1/user-subcategory-preference:
 *   put:
 *     summary: Create or update a subcategory preference
 *     tags:
 *       - Categories
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/UserSubcategoryPreferenceUpsertInput'
 *     responses:
 *       200:
 *         description: Saved subcategory preference
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/UserSubcategoryPreference'
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
        const safeData = UserSubcategoryPreferenceUpsertSchema.parse(data);

        const payload = {
            ...safeData,
            user: event.locals.user.id,
        };

        let preference: UserSubcategoryPreference | undefined;
        try {
            preference = await event.locals.pb
                .collection(Collection.user_subcategory_preferences)
                .getFirstListItem<UserSubcategoryPreference>(
                    event.locals.pb.filter(
                        "user = {:user} && subcategory = {:subcategory}",
                        {
                            user: event.locals.user.id,
                            subcategory: safeData.subcategory,
                        },
                    ),
                    { requestKey: null },
                );
        } catch {
            preference = undefined;
        }

        const saved = preference?.id
            ? await event.locals.pb
                  .collection(Collection.user_subcategory_preferences)
                  .update<UserSubcategoryPreference>(preference.id, payload, {
                      requestKey: null,
                  })
            : await event.locals.pb
                  .collection(Collection.user_subcategory_preferences)
                  .create<UserSubcategoryPreference>(payload, {
                      requestKey: null,
                  });

        return json(saved);
    } catch (e) {
        return handleError(e);
    }
}
