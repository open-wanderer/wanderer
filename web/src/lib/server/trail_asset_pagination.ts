import type { AssetLink } from "#lib/models/asset.ts";
import type { Trail } from "#lib/models/trail.ts";
import type { Waypoint } from "#lib/models/waypoint.ts";
import type { SummitLog } from "#lib/models/summit_log.ts";
import { enrichSummitLogAssetExpands, enrichWaypointAssetExpands } from "#lib/util/asset_link_util.ts";
import { forEachConcurrent } from "#lib/util/concurrency_util.ts";
import type { RequestEvent } from "@sveltejs/kit";

const pocketBaseExpandLimit = 1000;

interface AssetLinkTarget {
    id?: string;
    collection: "trail_assets" | "waypoint_assets" | "summit_log_assets";
    field: "trail" | "waypoint" | "summit_log";
    links?: AssetLink[];
    expandAsset?: boolean;
    replace: (links: AssetLink[]) => void;
}

// PocketBase truncates backrelation expands at 1000 and omits empty ones.
// A missing expand needs an independent authorized query before it can prove
// an empty photo set; saturated relations still need pagination.
export async function completeTrailAssetExpands(event: RequestEvent, trails: Trail[]): Promise<void> {
    const targets: AssetLinkTarget[] = [];
    for (const trail of trails) {
        trail.expand ??= {};
        const expand = trail.expand;
        targets.push({
            id: trail.id, collection: "trail_assets", field: "trail",
            links: expand.trail_assets_via_trail,
            replace: (links) => { expand.trail_assets_via_trail = links; },
        });
        for (const waypoint of expand.waypoints_via_trail ?? []) {
            waypoint.expand ??= {};
            const expand = waypoint.expand;
            targets.push({
                id: waypoint.id, collection: "waypoint_assets", field: "waypoint",
                links: expand.waypoint_assets_via_waypoint,
                replace: (links) => { expand.waypoint_assets_via_waypoint = links; },
            });
        }
        for (const log of expand.summit_logs_via_trail ?? []) {
            log.expand ??= {};
            const expand = log.expand;
            targets.push({
                id: log.id, collection: "summit_log_assets", field: "summit_log",
                links: expand.summit_log_assets_via_summit_log,
                replace: (links) => { expand.summit_log_assets_via_summit_log = links; },
            });
        }
    }
    await completeAssetLinkTargets(event, targets);
}

// Call this only for requested asset paths. It also handles PocketBase's omitted
// empty backrelations without treating a failed expand as an empty photo set.
export async function completeChildAssetExpands(event: RequestEvent, records: (Waypoint | SummitLog)[], kind: "waypoint" | "summit_log", expandAsset = true): Promise<void> {
    const collection = kind === "waypoint" ? "waypoint_assets" : "summit_log_assets";
    const relation = `${collection}_via_${kind}`;
    await completeAssetLinkTargets(event, records.map((record) => {
        record.expand ??= {};
        const expand = record.expand as unknown as Record<string, AssetLink[] | undefined>;
        return { id: record.id, collection, field: kind, expandAsset, links: expand[relation], replace: (links: AssetLink[]) => { expand[relation] = links; } };
    }));
    if (!expandAsset) return;
    const options = { share: event.url.searchParams.get("share") ?? undefined };
    for (const record of records) {
        const links = (record.expand as unknown as Record<string, AssetLink[] | undefined>)?.[relation];
        if (!links || links.some((link) => !link.expand?.asset)) continue;
        if (kind === "waypoint") enrichWaypointAssetExpands(record as Waypoint, options);
        else enrichSummitLogAssetExpands(record as SummitLog, options);
    }
}

export async function completeChildAssetResponse(event: RequestEvent, record: Waypoint | SummitLog, kind: "waypoint" | "summit_log"): Promise<void> {
    const collection = kind === "waypoint" ? "waypoint_assets" : "summit_log_assets";
    const relation = `${collection}_via_${kind}`;
    const paths = event.url.searchParams.get("expand")?.split(",").map((path) => path.trim()) ?? [];
    const requested = paths.some((path) => path === relation || path.startsWith(relation + "."));
    const expandAsset = paths.some((path) => path === relation + ".asset" || path.startsWith(relation + ".asset."));
    if (!requested) return;
    await completeChildAssetExpands(event, [record], kind, expandAsset);
}

async function completeAssetLinkTargets(event: RequestEvent, targets: AssetLinkTarget[]): Promise<void> {
    for (const target of targets) {
        if (target.links) target.replace([...target.links].sort(compareAssetLinks));
    }
    if (!event.url?.searchParams.has("fields")) {
        await proveMissingAssetLinkTargets(event, targets.filter((target) => target.id && target.links === undefined));
    }
    await forEachConcurrent(targets.filter((target) => target.id && (target.links?.length ?? 0) >= pocketBaseExpandLimit), 4, async (target) => {
        const links = await event.locals.pb.collection(target.collection).getFullList<AssetLink>({
            batch: 500,
            expand: target.expandAsset === false ? undefined : "asset",
            filter: event.locals.pb.filter(`${target.field} = {:id}`, { id: target.id }),
            sort: "+created,+id",
            query: { share: event.url.searchParams.get("share") ?? undefined },
            fetch: event.fetch,
            requestKey: null,
        });
        target.replace(links);
    });
}

async function proveMissingAssetLinkTargets(event: RequestEvent, targets: AssetLinkTarget[]): Promise<void> {
    const groups = new Map<string, AssetLinkTarget[]>();
    for (const target of targets) {
        const key = `${target.collection}:${target.expandAsset !== false}`;
        const group = groups.get(key) ?? [];
        group.push(target);
        groups.set(key, group);
    }
    const encode = (group: AssetLinkTarget[]) => event.locals.pb.filter(
        group.map((target, index) => `${target.field} = {:id${index}}`).join(" || "),
        Object.fromEntries(group.map((target, index) => [`id${index}`, target.id])),
    );
    const queries: { group: AssetLinkTarget[]; filter: string }[] = [];
    for (const targets of groups.values()) {
        try {
            let group: AssetLinkTarget[] = [];
            let filter = "";
            for (const target of targets) {
                const candidate = [...group, target];
                const candidateFilter = encode(candidate);
                if (group.length && candidateFilter.length > 3000) {
                    queries.push({ group, filter });
                    group = [target];
                    filter = encode(group);
                } else {
                    group = candidate;
                    filter = candidateFilter;
                }
            }
            if (group.length) queries.push({ group, filter });
        } catch {
            // An unavailable filter cannot establish an empty relation.
        }
    }
    await forEachConcurrent(queries, 4, async ({ group, filter }) => {
        const target = group[0];
        try {
            const links = await event.locals.pb.collection(target.collection).getFullList<AssetLink>({
                batch: 500, expand: target.expandAsset === false ? undefined : "asset",
                filter, sort: "+created,+id",
                query: { share: event.url.searchParams.get("share") ?? undefined },
                fetch: event.fetch, requestKey: null,
            });
            const byTarget = new Map<string, AssetLink[]>();
            for (const link of links) {
                const id = link[target.field];
                if (!id) continue;
                const list = byTarget.get(id) ?? [];
                list.push(link);
                byTarget.set(id, list);
            }
            for (const target of group) target.replace((byTarget.get(target.id!) ?? []).sort(compareAssetLinks));
        } catch {
            // Failure or denied access is inconclusive. Keeping the key absent
            // prevents a remote copy from interpreting it as confirmed deletion.
        }
    });
}

function compareAssetLinks(a: AssetLink, b: AssetLink): number {
    return (a.created ?? "").localeCompare(b.created ?? "") || (a.id ?? "").localeCompare(b.id ?? "");
}
