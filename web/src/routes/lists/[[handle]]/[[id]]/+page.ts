import { browser } from "$app/environment";
import { type ListFilter } from "$lib/models/list";
import { lists_search_filter, lists_show } from "$lib/stores/list_store";
import { APIError } from "$lib/util/api_util";
import { error, type Load, type NumericRange } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, url }) => {
    const filter: ListFilter = {
        q: "",
        author: "",
        shared: true,
        public: true,
        sort: "created",
        sortOrder: "+",
    };

    let lists: Awaited<ReturnType<typeof lists_search_filter>>;
    if (params.handle && params.id) {
        try {
            const list = await lists_show(params.id, params.handle, false, fetch)

            lists = { items: [list], page: 1, totalPages: 1, hits: [] }
        } catch (e) {
            if (e instanceof APIError) {
                error(e.status as NumericRange<400, 599>, {
                    message: e.status == 404 ? 'Not found' : e.message
                });
            }
            throw e
        }
    } else if (browser) {
        lists = await lists_search_filter(filter, 1, undefined, fetch);        
    } else {
        lists = { items: [], page: 1, totalPages: 1, hits: [] }
    }

    return { lists, filter }
};