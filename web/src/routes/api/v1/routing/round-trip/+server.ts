import { routingProxy } from "$lib/server/routing_proxy";
import type { RequestEvent } from "@sveltejs/kit";

export const POST = (event: RequestEvent) => routingProxy(event, "round-trip", "POST");
