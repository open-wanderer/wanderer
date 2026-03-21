import { Collection, handleError, remove } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.api_tokens)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}
