import type { Profile } from '$lib/models/profile';
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { handleError } from '$lib/util/api_util';
import { error, json, type RequestEvent } from '@sveltejs/kit';

/**
 * @swagger
 * /api/v1/profile/{handle}:
 *   get:
 *     summary: Get user profile
 *     description: Retrieves a user's profile by handle, with optional federation support
 *     tags:
 *       - Profiles
 *     parameters:
 *       - in: path
 *         name: handle
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Profile and actor data
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    const handle = event.params.handle;
    if (!handle) {
        return error(400, { message: "Bad request" })
    }

    try {
        const { actor, error: actorError } = await getActorResponseForHandle(event, handle, { follows: true });

        const profile: Profile = {
            id: actor.id!,
            username: actor.username,
            preferredUsername: actor.preferred_username,
            acct: handle,
            createdAt: actor.published ?? "",
            bio: actor.summary ?? "",
            uri: actor.iri,
            followers: actor.followerCount ?? 0,
            following: actor.followingCount ?? 0,
            icon: actor.icon ?? "",
            error: actorError ?? undefined
        }

        return json({ profile, actor: actor })
    } catch (e) {
        if (e instanceof Error && e.message == "fetch failed") {
            return error(404, "Not found")
        }
        return handleError(e)
    }
}
