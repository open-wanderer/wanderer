import { error, json, type NumericRange, type RequestEvent } from "@sveltejs/kit";
import { ClientResponseError, type ListResult } from "pocketbase";
import { ZodError, type ZodSchema } from "zod";
import { RecordListOptionsSchema, RecordIdSchema, RecordOptionsSchema } from "$lib/models/api/base_schema";

export class APIError extends Error {
    status: number;
    message: string;
    detail: any;

    constructor(status: number, message: string, detail?: any) {
        super();
        this.status = status;
        this.message = message;
        this.detail = detail
    }
}


export enum Collection {
    users = "users",
    activitypub_activities = "activitypub_activities",
    activitypub_actors = "activitypub_actors",
    categories = "categories",
    comments = "comments",
    feed = "feed",
    follows = "follows",
    integrations = "integrations",
    list_share = "list_share",
    lists = "lists",
    notifications = "notifications",
    profile_feed = "profile_feed",
    settings = "settings",
    summit_logs = "summit_logs",
    trail_like = "trail_like",
    trail_share = "trail_share",
    trail_link_share = "trail_link_share",
    trails = "trails",
    tags = "tags",
    waypoints = "waypoints",
    trails_bounding_box = "trails_bounding_box",
    trails_filter = "trails_filter",
    users_anonymous = "users_anonymous",

}

export async function list<T>(event: RequestEvent, collection: Collection) {
    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordListOptionsSchema.parse(searchParams);
    const { perPage, page, ...opts } = safeSearchParams;

    let r: ListResult<T>;
    if ((safeSearchParams.perPage ?? 0) < 0) {
        const activities: T[] = await event.locals.pb.collection(Collection[collection])
            .getFullList<T>({...opts})
        r = {
            items: activities,
            perPage: -1,
            page: 1,
            totalItems: activities.length,
            totalPages: 1
        }
    } else {
        r = await event.locals.pb.collection(Collection[collection])
            .getList<T>(page, perPage, opts)
    }
    return r
}

export async function show<T>(event: RequestEvent, collection: Collection) {
    const params = event.params
    const safeParams = RecordIdSchema.parse(params);

    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordOptionsSchema.parse(searchParams);

    const r = await event.locals.pb.collection(collection.toString())
        .getOne<T>(safeParams.id, safeSearchParams)

    return r
}

export async function create<T>(event: RequestEvent, schema: ZodSchema, collection: Collection) {
    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordOptionsSchema.parse(searchParams);

    const data = await event.request.json();
    const safeData = schema.parse(data);

    const r = await event.locals.pb.collection(Collection[collection]).create<T>(safeData, { ...safeSearchParams, requestKey: null })

    return r
}

export async function update<T>(event: RequestEvent, schema: ZodSchema, collection: Collection) {
    const params = event.params
    const safeParams = RecordIdSchema.parse(params);

    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordOptionsSchema.parse(searchParams);

    const data = await event.request.json();
    const safeData = schema.parse(data);

    const r = await event.locals.pb.collection(Collection[collection]).update<T>(safeParams.id, safeData, safeSearchParams)

    return r
}

export async function uploadCreate<T>(event: RequestEvent, collection: Collection) {
    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordOptionsSchema.parse(searchParams);

    const data = await event.request.formData();


    const r = await event.locals.pb.collection(Collection[collection]).create<T>(data, safeSearchParams)

    return r
}

export async function uploadUpdate<T>(event: RequestEvent, collection: Collection) {
    const searchParams = Object.fromEntries(event.url.searchParams);
    const safeSearchParams = RecordOptionsSchema.parse(searchParams);

    const data = await event.request.formData();
    if (!data.has('id')) {
        throw new Error("data has no id")
    }


    const r = await event.locals.pb.collection(Collection[collection]).update<T>(data.get('id')!.toString(), data, safeSearchParams)

    return r
}

export async function upload<T>(event: RequestEvent, collection: Collection) {
    const params = event.params
    const safeParams = RecordIdSchema.parse(params);

    const data = await event.request.formData();

    const r = await event.locals.pb.collection(Collection[collection]).update<T>(safeParams.id, data)

    return r
}

export async function remove(event: RequestEvent, collection: Collection) {
    const params = event.params
    const safeParams = RecordIdSchema.parse(params);

    const r = await event.locals.pb.collection(Collection[collection]).delete(safeParams.id)

    return { 'acknowledged': r }
}

export function handleError(e: any) {
    if (e instanceof ZodError) {
        return json({ message: "invalid_params", detail: e.issues }, { status: 400 })
    } else if (e instanceof ClientResponseError && e.status > 0) {
        return json({ ...e.response, message: e.message, detail: e.originalError.data }, { status: e.status })
    } else if (e instanceof SyntaxError) {
        return json({ message: "invalid_json" }, { status: 400 })
    } else {
        return json({ message: e }, { status: 500 })
    }
}
