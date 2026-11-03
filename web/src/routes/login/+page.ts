import { users_auth_methods } from "#lib/stores/user_store.ts";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {

    const authMethods = await users_auth_methods(fetch)

    return { authMethods: authMethods }
};