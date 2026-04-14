import { ListUpdateSchema } from "$lib/models/api/list_schema";
import type { List } from "$lib/models/list";
import { Collection, handleError, remove, update } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function GET(event: RequestEvent) {
    const { url, params } = event;

    try {
        let list: List = await event.locals.pb.send(`/remote/list/${params.id}?` + url.searchParams, {
            method: "GET",
            fetch: event.fetch,
        })

        return json(list)
    } catch (e: any) {
        return handleError(e);
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<List>(event, ListUpdateSchema, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

