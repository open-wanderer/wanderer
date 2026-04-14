import { follows_index } from "$lib/stores/follow_store";
import { profile_follows_index } from "$lib/stores/profile_store";
import { APIError } from "$lib/util/api_util";
import { error, type Load } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

export const load: Load = async ({ params, fetch }) => {
    if (!params.type || (params.type !== "followers" && params.type !== "following")) {
        throw error(400, "Missing or wrong type");
    }

    try {
        const follows = await profile_follows_index(
            params.handle!,
            params.type as "followers" | "following",
            1,
            fetch,
        );
        return { follows }

    } catch (e) {
        if (e instanceof APIError) {
            return error(e.status, e.message)
        }
        throw e
    }
};