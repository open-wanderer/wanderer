import { error, json, type RequestEvent } from "@sveltejs/kit";

type SearchOptions = {
    filter?: string | string[];
    sort?: string[];
    attributesToRetrieve?: string[];
    hitsPerPage?: number;
    page?: number;
    limit?: number;
    offset?: number;
};

type BBoxPayload = {
    minLat: number;
    minLon: number;
    maxLat: number;
    maxLon: number;
};

type SearchPayload = {
    mode?: "bbox" | "table";
    q?: string;
    filter?: string;
    bbox?: BBoxPayload;
    returnAll?: boolean;
    options?: SearchOptions;
    attributesToRetrieve?: string[];
};

type CountRecord = {
    id: string;
    trail_count: number;
};

export async function POST(event: RequestEvent) {
    const data: SearchPayload = await event.request.json();

    try {
        if (data.mode === "bbox") {
            return json(await searchTrailsByBBox(event, data));
        }
        if (data.mode === "table") {
            return json(await searchTrailsByTable(event, data));
        }

        const options = { ...(data.options ?? {}) } as SearchOptions;
        if (data.attributesToRetrieve && !options.attributesToRetrieve) {
            options.attributesToRetrieve = data.attributesToRetrieve;
        }

        const r = await event.locals.ms.index("trails").search(data.q ?? "", options);
        return json(r);
    } catch (e: any) {
        throw error(e.httpStatus ?? 500, e);
    }
}

async function searchTrailsByBBox(event: RequestEvent, data: SearchPayload) {
    if (!data.bbox) {
        throw error(400, "Missing bbox payload");
    }

    const bbox = data.bbox;
    const options = normalizeOptions(data);
    const hitsPerPage = options.hitsPerPage ?? 200;
    const page = options.page ?? 1;
    const perGroupLimit = Math.min(1000, hitsPerPage * page * 2);

    if (!bucketsEnabled()) {
        const filter = buildFilter([data.filter, buildTrailBBoxFilter(bbox)]);
        if (data.returnAll) {
            const hits = await fetchAllHits(event, data.q ?? "", filter, options);
            const merged = mergeAndSortHits([hits], options.sort);
            return buildSearchResponse(merged, 1, merged.length || hitsPerPage, merged.length);
        }

        const limit = hitsPerPage;
        const offset = Math.max(page - 1, 0) * limit;
        const response = await event.locals.ms.index("trails").search(data.q ?? "", {
            filter,
            sort: options.sort,
            attributesToRetrieve: options.attributesToRetrieve,
            limit,
            offset,
        });
        const hits = response.hits ?? [];
        const totalHits = (response as any).estimatedTotalHits ?? (response as any).totalHits ?? hits.length;
        return buildSearchResponse(hits, page, hitsPerPage, totalHits);
    }

    const nodes = await fetchAllCounts(event, "quad_nodes", {
        filter: buildPbBBoxFilter(bbox),
        fields: "id,trail_count",
    });

    const groups = groupByMaxTotal(nodes, 1000);
    if (groups.length === 0) {
        return buildSearchResponse([], page, hitsPerPage);
    }

    if (data.returnAll) {
        const hits = await fetchAllBBoxHits(event, groups, data, bbox, options);
        const merged = mergeAndSortHits(hits, options.sort);
        return buildSearchResponse(merged, 1, merged.length || hitsPerPage, merged.length);
    }

    const queries = groups.map((group) => ({
        indexUid: "trails",
        q: data.q ?? "",
        filter: buildFilter([
            data.filter,
            buildNodeFilter(group),
            buildTrailBBoxFilter(bbox),
        ]),
        sort: options.sort,
        attributesToRetrieve: options.attributesToRetrieve,
        limit: perGroupLimit,
    }));

    const response = await event.locals.ms.multiSearch({ queries });
    const results = response.results ?? [];
    const hits = mergeAndSortHits(results.map((r: any) => r.hits), options.sort);
    const totalHits = sumEstimatedTotalHits(results);

    return buildSearchResponse(paginate(hits, page, hitsPerPage), page, hitsPerPage, totalHits);
}

async function searchTrailsByTable(event: RequestEvent, data: SearchPayload) {
    const options = normalizeOptions(data);
    const pageSize = options.hitsPerPage ?? 50;
    const page = options.page ?? 1;
    const perGroupLimit = Math.min(1000, pageSize * page * 2);

    if (!bucketsEnabled()) {
        const limit = pageSize;
        const offset = Math.max(page - 1, 0) * limit;
        const response = await event.locals.ms.index("trails").search(data.q ?? "", {
            filter: data.filter,
            sort: options.sort,
            attributesToRetrieve: options.attributesToRetrieve,
            limit,
            offset,
        });
        const hits = response.hits ?? [];
        const totalHits = (response as any).estimatedTotalHits ?? (response as any).totalHits ?? hits.length;
        return buildSearchResponse(hits, page, pageSize, totalHits);
    }
    
    const buckets = await fetchAllCounts(event, "trail_time_buckets", {
        fields: "id,trail_count",
    });

    const groups = groupByMaxTotal(buckets, 1000);
    if (groups.length === 0) {
        return buildSearchResponse([], page, pageSize);
    }

    const queries = groups.map((group) => ({
        indexUid: "trails",
        q: data.q ?? "",
        filter: buildFilter([
            data.filter,
            buildBucketFilter(group),
        ]),
        sort: options.sort,
        attributesToRetrieve: options.attributesToRetrieve,
        limit: perGroupLimit,
    }));

    const response = await event.locals.ms.multiSearch({ queries });
    const results = response.results ?? [];
    const hits = mergeAndSortHits(results.map((r: any) => r.hits), options.sort);
    const totalHits = sumEstimatedTotalHits(results);

    return buildSearchResponse(paginate(hits, page, pageSize), page, pageSize, totalHits);
}

function normalizeOptions(data: SearchPayload): SearchOptions {
    const options = { ...(data.options ?? {}) } as SearchOptions;
    if (data.attributesToRetrieve && !options.attributesToRetrieve) {
        options.attributesToRetrieve = data.attributesToRetrieve;
    }
    return options;
}

function buildPbBBoxFilter(bbox: BBoxPayload) {
    return `is_leaf = true && max_lat >= ${bbox.minLat} && min_lat <= ${bbox.maxLat} && max_lon >= ${bbox.minLon} && min_lon <= ${bbox.maxLon}`;
}

function buildTrailBBoxFilter(bbox: BBoxPayload) {
    return `min_lat <= ${bbox.maxLat} AND max_lat >= ${bbox.minLat} AND min_lon <= ${bbox.maxLon} AND max_lon >= ${bbox.minLon}`;
}

function buildNodeFilter(nodes: CountRecord[]) {
    const ids = nodes.map((node) => `"${node.id}"`).join(", ");
    return `quad_node_ids IN [${ids}]`;
}

function buildBucketFilter(buckets: CountRecord[]) {
    const ids = buckets.map((bucket) => `"${bucket.id}"`).join(", ");
    return `time_bucket_ids IN [${ids}]`;
}

function buildFilter(parts: Array<string | undefined>): string | undefined {
    const filtered = parts.filter((part) => part && part.trim().length > 0) as string[];
    if (filtered.length === 0) {
        return undefined;
    }
    return filtered.join(" AND ");
}

function groupByMaxTotal(records: CountRecord[], maxTotal: number) {
    const items = records.filter((record) => record.trail_count > 0);
    items.sort((a, b) => b.trail_count - a.trail_count || a.id.localeCompare(b.id));

    const groups: CountRecord[][] = [];
    let current: CountRecord[] = [];
    let running = 0;

    for (const record of items) {
        if (running+record.trail_count > maxTotal && current.length > 0) {
            groups.push(current);
            current = [];
            running = 0;
        }
        current.push(record);
        running += record.trail_count;
    }

    if (current.length > 0) {
        groups.push(current);
    }

    return groups;
}

function mergeAndSortHits(hitsLists: any[][], sort?: string[]) {
    const map = new Map<string, any>();
    for (const hits of hitsLists) {
        for (const hit of hits ?? []) {
            if (!map.has(hit.id)) {
                map.set(hit.id, hit);
            }
        }
    }
    const merged = Array.from(map.values());
    if (!sort || sort.length === 0) {
        return merged;
    }

    const [field, order] = sort[0].split(":");
    const direction = order === "desc" ? -1 : 1;

    merged.sort((a, b) => compareValues(a[field], b[field]) * direction);
    return merged;
}

function sumEstimatedTotalHits(results: any[]) {
    let total = 0;
    for (const result of results) {
        const value = result?.estimatedTotalHits ?? result?.totalHits ?? 0;
        total += typeof value === "number" ? value : 0;
    }
    return total;
}

function compareValues(a: any, b: any) {
    if (a === b) {
        return 0;
    }
    if (a === undefined || a === null) {
        return -1;
    }
    if (b === undefined || b === null) {
        return 1;
    }
    if (typeof a === "number" && typeof b === "number") {
        return a < b ? -1 : 1;
    }
    return String(a).localeCompare(String(b));
}

function paginate<T>(items: T[], page: number, pageSize: number) {
    const start = Math.max(page - 1, 0) * pageSize;
    return items.slice(start, start + pageSize);
}

function buildSearchResponse(hits: any[], page: number, hitsPerPage: number, totalHits?: number) {
    const total = totalHits ?? hits.length;
    const totalPages = hitsPerPage > 0 ? Math.ceil(total / hitsPerPage) : 0;
    return {
        page,
        hitsPerPage,
        totalHits: total,
        totalPages,
        hits,
    };
}

function bucketsEnabled() {
    const value = (process.env.ENABLE_TRAIL_BUCKETS ?? "").trim().toLowerCase();
    return value === "true" || value === "1" || value === "yes";
}

async function fetchAllCounts(event: RequestEvent, collection: string, params: { filter?: string; fields?: string }) {
    const items: CountRecord[] = [];
    const perPage = 200;
    let page = 1;

    while (true) {
        const result = await event.locals.pb.collection(collection).getList<CountRecord>(page, perPage, {
            filter: params.filter,
            fields: params.fields,
        });
        items.push(...result.items);
        if (result.items.length < perPage) {
            break;
        }
        page += 1;
    }

    return items;
}

async function fetchAllBBoxHits(
    event: RequestEvent,
    groups: CountRecord[][],
    data: SearchPayload,
    bbox: BBoxPayload,
    options: SearchOptions,
) {
    const hitsLists: any[][] = [];
    const limit = 1000;

    for (const group of groups) {
        const filter = buildFilter([
            data.filter,
            buildNodeFilter(group),
            buildTrailBBoxFilter(bbox),
        ]);

        let offset = 0;
        while (true) {
            const response = await event.locals.ms.index("trails").search(data.q ?? "", {
                filter,
                sort: options.sort,
                attributesToRetrieve: options.attributesToRetrieve,
                limit,
                offset,
            });
            const hits = response.hits ?? [];
            hitsLists.push(hits);
            if (hits.length < limit) {
                break;
            }
            offset += limit;
        }
    }

    return hitsLists;
}

async function fetchAllHits(
    event: RequestEvent,
    query: string,
    filter: string | undefined,
    options: SearchOptions,
) {
    const hits: any[] = [];
    const limit = 1000;
    let offset = 0;

    while (true) {
        const response = await event.locals.ms.index("trails").search(query, {
            filter,
            sort: options.sort,
            attributesToRetrieve: options.attributesToRetrieve,
            limit,
            offset,
        });
        const pageHits = response.hits ?? [];
        hits.push(...pageHits);
        if (pageHits.length < limit) {
            break;
        }
        offset += limit;
    }

    return hits;
}
