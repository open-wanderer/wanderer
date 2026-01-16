import { trails_show } from "$lib/stores/trail_store";
import { APIError } from "$lib/util/api_util";
import { error, type Load, type NumericRange } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, url }) => {
    try {
        const trail = await trails_show(params.id!, params.handle, url.searchParams.get("share") ?? undefined, true, fetch)
        return { trail };
    } catch (e) {
        if (e instanceof APIError) {
            error(e.status as NumericRange<400, 599>, {
                message: e.status == 404 ? 'Not found' : e.message
            });
        }
        console.error(e);

    }
};