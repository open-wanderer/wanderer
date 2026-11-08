import type { Trail } from "#lib/models/trail.ts";
import type { RequestEvent } from "@sveltejs/kit";

// PocketBase omits empty reverse relations. Make their emptiness explicit
// only after a successful relation query confirms it. Missing expands may
// also come from a failed relation query and cannot establish a deletion.
export async function completeRequestedTrailChildren(event: RequestEvent, trail: Trail, params: URLSearchParams): Promise<void> {
    if (params.has("fields") || !trail.id) return;
    const requested = params.get("expand")?.split(",").map((path) => path.trim()) ?? [];
    for (const relation of ["waypoints_via_trail", "summit_logs_via_trail"] as const) {
        if (!requested.some((path) => path === relation || path.startsWith(relation + "."))) continue;
        if (trail.expand?.[relation] !== undefined) continue;
        const collection = relation === "waypoints_via_trail" ? "waypoints" : "summit_logs";
        try {
            const children = await event.locals.pb.collection(collection).getList(1, 1, {
                filter: event.locals.pb.filter("trail = {:trail}", { trail: trail.id }),
                fields: "id", skipTotal: false,
                query: { share: params.get("share") ?? undefined },
                fetch: event.fetch, requestKey: null,
            });
            if (children.totalItems !== 0 || children.items.length !== 0) continue;
            trail.expand ??= {};
            trail.expand[relation] = [];
        } catch {
            // Retain an inconclusive missing key if this independent proof
            // also fails or its collection is unavailable to the reader.
        }
    }
}
