import { isAppState, verifyAppState } from "#lib/server/oauth_app_state.ts";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = ({ url }) => {
    const state = url.searchParams.get("state");
    if (!isAppState(state)) {
        return { appLogin: null, appLoginRefused: false };
    }
    const appLogin = verifyAppState(state);
    return { appLogin, appLoginRefused: appLogin === null };
};
