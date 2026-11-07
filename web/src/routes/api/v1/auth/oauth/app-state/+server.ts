import { clientAddress } from "#lib/server/http.ts";
import { APP_STATE_TTL_SECONDS, issueAppState } from "#lib/server/oauth_app_state.ts";
import { createRateLimiter } from "#lib/server/rate_limit.ts";
import { handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

// The app asks once per login attempt, and the endpoint needs no login.
const limiter = createRateLimiter(30, 60_000);

/**
 * @swagger
 * /api/v1/auth/oauth/app-state:
 *   post:
 *     summary: Issue the OAuth state for a login from the mobile app
 *     description: >
 *       Returns a signed, expiring value for the `state` parameter of the provider's
 *       authorization URL. `/login/redirect` hands a login to the app only with such a state
 *       that is valid when the provider redirects back, and only after the user confirms.
 *     tags:
 *       - Authentication
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - provider
 *             properties:
 *               provider:
 *                 type: string
 *                 description: The `name` of a provider from GET /api/v1/auth/oauth.
 *     responses:
 *       200:
 *         description: The state to send to the provider, unchanged
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 state:
 *                   type: string
 *                 expiresIn:
 *                   type: number
 *                   description: Seconds until `/login/redirect` refuses the state.
 *       400:
 *         description: Bad Request - malformed body or unknown provider
 *       429:
 *         description: Too many requests from this address; see the Retry-After header
 *       500:
 *         description: Internal Server Error
 */
export async function POST(event: RequestEvent) {
    const retryAfter = limiter(clientAddress(event) ?? "unknown");
    if (retryAfter !== undefined) {
        return Response.json({ message: "too_many_requests" }, { status: 429, headers: { "Retry-After": String(retryAfter) } });
    }

    try {
        const { provider: name } = z.object({ provider: z.string() }).parse(await event.request.json());
        const methods = await event.locals.pb.collection("users").listAuthMethods();
        const provider = methods.oauth2.providers.find((p) => p.name === name);
        if (!provider) {
            return Response.json({ message: "invalid_provider" }, { status: 400 });
        }
        return Response.json({ state: issueAppState(provider.displayName), expiresIn: APP_STATE_TTL_SECONDS });
    } catch (e) {
        return handleError(e);
    }
}
