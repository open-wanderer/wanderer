import { browser } from "$app/environment";
import { type ListFilter } from "$lib/models/list";
import { lists_search_filter, lists_show } from "$lib/stores/list_store";
import { APIError } from "$lib/util/api_util";
import { error, type Load, type NumericRange } from "@sveltejs/kit";
import type { AuthRecord } from "pocketbase";

export const load: Load = async ({ params, fetch, parent }) => {
    const filter: ListFilter = {
        q: "",
        author: "",
        shared: true,
        public: true,
        sort: "created",
        sortOrder: "+",
    };

    const parentData = await parent();
    const user = (parentData as { user?: AuthRecord }).user;

    let lists: Awaited<ReturnType<typeof lists_search_filter>> = {
        items: [],
        page: 1,
        totalPages: 1,
        hits: [],
    };
    if (browser) {
        lists = await lists_search_filter(filter, 1, undefined, fetch, user);
    }

    let selectedList: Awaited<ReturnType<typeof lists_show>> | null = null;
    if (params.handle && params.id) {
        try {
            selectedList = await lists_show(params.id, params.handle, fetch);
        } catch (e) {
            if (e instanceof APIError) {
                error(e.status as NumericRange<400, 599>, {
                    message: e.status == 404 ? "Not found" : e.message,
                });
            }
            throw e;
        }
    }

    return { lists, filter, selectedList };
};
