import {
    formatManeuverResponse,
    resolveManeuverLanguage,
} from "$lib/server/maneuver_formatter";
import type { RoutingInternalManeuverResponse } from "$lib/models/routing";
import { handleError } from "$lib/util/api_util";
import { json, type RequestEvent } from "@sveltejs/kit";

export async function POST(event: RequestEvent) {
    try {
        const data = await event.request.json();
        const resolved = resolveManeuverLanguage({
            explicit: typeof data.language === "string" ? data.language : undefined,
            settingsLanguage: event.locals.settings?.language,
            acceptLanguage: event.request.headers.get("accept-language"),
        });
        const request = {
            trailId: data.trailId,
            language: resolved.language,
            ...(typeof data.share === "string" && data.share ? { share: data.share } : {}),
        };
        const internal = await event.locals.pb.send<RoutingInternalManeuverResponse>(
            "/plugins/routing/maneuvers",
            {
                method: "POST",
                body: JSON.stringify(request),
                headers: { "Content-Type": "application/json" },
            },
        );
        return json(await formatManeuverResponse(internal, resolved));
    } catch (e: any) {
        return handleError(e);
    }
}
