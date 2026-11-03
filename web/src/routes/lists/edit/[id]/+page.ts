import { List } from "#lib/models/list.ts";
import { lists_show } from "#lib/stores/list_store.ts";
import { APIError } from "#lib/util/api_util.ts";
import { getFileURL } from "#lib/util/file_util.ts";
import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, data }) => {
    if (!params.id) {
        return error(400, "Bad Request")
    }

    let list: List;
    if (params.id === "new") {
        list = new List("", []);
        return { list: list, previewUrl: "" }
    } else {
        try {
            list = await lists_show(params.id, undefined, fetch);
            const previewURL = getFileURL(list, list.avatar, "300x300");

            return { list: list, previewUrl: previewURL }

        } catch (e) {
            if (e instanceof APIError) {
                throw error(e.status, e.message)
            }
            throw e
        }
    }

};