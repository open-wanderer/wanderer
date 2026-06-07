import { List } from "$lib/models/list";
import { lists_show } from "$lib/stores/list_store";
import { APIError } from "$lib/util/api_util";
import { getFileURL } from "$lib/util/file_util";
import { error, type Load, type NumericRange } from "@sveltejs/kit";

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
            list = await lists_show(params.id, undefined, false, fetch);
            const previewURL = getFileURL(list, list.avatar);

            return { list: list, previewUrl: previewURL }

        } catch (e) {
            if (e instanceof APIError) {
                throw error(e.status as NumericRange<400, 599>, e.message)
            }
            throw e
        }
    }

};