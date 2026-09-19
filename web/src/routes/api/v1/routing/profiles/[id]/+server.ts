import { routingProxy } from "$lib/server/routing_proxy";
import type { RequestEvent } from "@sveltejs/kit";

export const PATCH = (event: RequestEvent) =>
    routingProxy(event, `profiles/${encodeURIComponent(event.params.id!)}`, "PATCH");
export const DELETE = (event: RequestEvent) =>
    routingProxy(event, `profiles/${encodeURIComponent(event.params.id!)}`, "DELETE");
