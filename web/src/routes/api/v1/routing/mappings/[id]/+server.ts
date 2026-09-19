import { routingProxy } from "$lib/server/routing_proxy";
import type { RequestEvent } from "@sveltejs/kit";

export const PATCH = (event: RequestEvent) =>
    routingProxy(event, `mappings/${encodeURIComponent(event.params.id!)}`, "PATCH");
