import { routingProxy } from "$lib/server/routing_proxy";
import type { RequestEvent } from "@sveltejs/kit";

export const GET = (event: RequestEvent) => routingProxy(event, "settings", "GET");
export const PATCH = (event: RequestEvent) => routingProxy(event, "settings", "PATCH");
