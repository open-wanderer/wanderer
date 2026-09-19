import { routingProxy } from "$lib/server/routing_proxy";
import type { RequestEvent } from "@sveltejs/kit";

export const GET = (event: RequestEvent) => routingProxy(event, "engines", "GET");
