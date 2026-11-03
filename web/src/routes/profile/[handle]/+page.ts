import type { ListFilter } from "#lib/models/list.ts";
import { profile_lists_index, profile_feed_index } from "#lib/stores/profile_store.ts";
import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, parent }) => {
    if (!params.handle) {
        error(404, "Not found")
    }

    const filter: ListFilter = {
        q: "",
        author: "",
        shared: true,
        public: true,
        sort: "created",
        sortOrder: "+",
    };


    try {
        const lists = await profile_lists_index(params.handle, filter, 1, 6, fetch)
        const feed = await profile_feed_index(params.handle, 1, 10, fetch);
        return { lists: lists.items, feed }
    } catch(e) {
        return {lists: [], feed: {items: [], page: 1, perPage: 1, totalItems: 0, totalPages: 0}}
    } 
    
    
};