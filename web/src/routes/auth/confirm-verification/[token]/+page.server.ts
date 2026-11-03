import { Collection } from "#lib/util/api_util.ts";
import { error, type ServerLoad } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

export const load: ServerLoad = async ({ params, locals, url, fetch }) => {
    if (!params.token) {
        error(400, "Invalid token")
    }

    try {
        await locals.pb.collection(Collection.users).confirmVerification(params.token)
    } catch (e) {
        if (e instanceof ClientResponseError) {
            error(400, e.message);
        } else {
            throw e;
        }
    }
};