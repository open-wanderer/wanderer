import { POCKETBASE_PROXY_SECRET } from "$app/env/private";
import { createHmac, randomBytes, timingSafeEqual } from "node:crypto";

const PREFIX = "wanderer-app";
export const APP_STATE_TTL_SECONDS = 600;

// The proxy secret is the one secret every deployment shares between replicas.
// Without it states stop verifying on a restart, which their lifetime makes harmless.
const key = createHmac("sha256", POCKETBASE_PROXY_SECRET || randomBytes(32)).update("oauth-app-state").digest();

function sign(payload: string) {
    return createHmac("sha256", key).update(payload).digest();
}

export function isAppState(state: string | null): state is string {
    return state?.startsWith(`${PREFIX}.`) ?? false;
}

/** A value for the OAuth `state` parameter that only this server can produce. */
export function issueAppState(provider: string, now = Date.now()) {
    const payload = Buffer.from(JSON.stringify({
        n: randomBytes(16).toString("base64url"),
        e: Math.floor(now / 1000) + APP_STATE_TTL_SECONDS,
        p: provider,
    })).toString("base64url");
    return `${PREFIX}.${payload}.${sign(payload).toString("base64url")}`;
}

/** The provider a state was issued for, or null when it is forged, malformed or expired. */
export function verifyAppState(state: string, now = Date.now()): { provider: string } | null {
    const [prefix, payload, signature, ...rest] = state.split(".");
    if (prefix !== PREFIX || !payload || !signature || rest.length > 0) {
        return null;
    }
    const expected = sign(payload);
    const actual = Buffer.from(signature, "base64url");
    if (actual.length !== expected.length || !timingSafeEqual(actual, expected)) {
        return null;
    }
    const { e, p } = JSON.parse(Buffer.from(payload, "base64url").toString());
    return e * 1000 > now ? { provider: p } : null;
}
