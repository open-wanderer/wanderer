import { trails_show } from "$lib/stores/trail_store";
import { remoteContentError } from "$lib/util/remote_content_error";
import { APIError } from "$lib/util/api_util";
import { error, type NumericRange, type ServerLoad } from "@sveltejs/kit";

export const load: ServerLoad = async ({ params, url, fetch }) => {
    try {
        const trail = await trails_show(params.id!, params.handle, url.searchParams.get("share") ?? undefined, true, fetch)
        return { trail };
    } catch (e) {
        if (e instanceof APIError) {
            error(e.status as NumericRange<400, 599>, remoteContentError(e));
        }
        throw e
    }
};