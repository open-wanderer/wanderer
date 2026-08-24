import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function routingProxy(
    event: RequestEvent,
    path: string,
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
) {
    try {
        const options: { method: string; body?: string; headers?: Record<string, string> } = {
            method,
        };
        if (method !== "GET" && method !== "DELETE") {
            options.body = JSON.stringify(await event.request.json());
            options.headers = { "Content-Type": "application/json" };
        }
        return json(await event.locals.pb.send(`/plugins/routing/${path}`, options));
    } catch (error: unknown) {
        return handleError(error);
    }
}
