import type { Trail } from "#lib/models/trail.ts";
import type { RequestEvent } from "@sveltejs/kit";

// An export must have complete child data before trail2gpx runs. PocketBase
// omits empty backrelations and truncates saturated ones; prove/fill them
// through authorized child queries without requesting any photo assets.
export async function completeOwnerExportChildren(event: RequestEvent, trails: Trail[]): Promise<void> {
    for (const [relation, collection] of [
        ["waypoints_via_trail", "waypoints"],
        ["summit_logs_via_trail", "summit_logs"],
    ] as const) {
        const missing = trails.filter((trail) => trail.id && (
            !Array.isArray(trail.expand?.[relation]) || (trail.expand?.[relation]?.length ?? 0) >= 1000
        ));
        if (!missing.length) continue;
        const params = Object.fromEntries(missing.map((trail, index) => [`id${index}`, trail.id]));
        const children = await event.locals.pb.collection(collection).getFullList({
            batch: 500,
            filter: event.locals.pb.filter(missing.map((_, index) => `trail = {:id${index}}`).join(" || "), params),
            sort: "id", fetch: event.fetch, requestKey: null,
        });
        const byTrail = new Map<string, typeof children>();
        for (const child of children) {
            const group = byTrail.get(child.trail) ?? [];
            group.push(child);
            byTrail.set(child.trail, group);
        }
        for (const trail of missing) {
            trail.expand ??= {};
            trail.expand[relation] = (byTrail.get(trail.id!) ?? []) as never;
        }
    }
}
