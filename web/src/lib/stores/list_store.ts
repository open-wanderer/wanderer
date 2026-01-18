import { List, type ListFilter } from "$lib/models/list";
import type { Trail } from "$lib/models/trail";
import { APIError } from "$lib/util/api_util";
import type { Hits } from "meilisearch";
import { type AuthRecord, type ListResult, type RecordModel } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import type { ListSearchResult } from "./search_store";
import { fetchGPX } from "./trail_store";
import { currentUser } from "./user_store";
import { objectToFormData } from "$lib/util/file_util";

let lists: List[] = []
export const list: Writable<List | null> = writable(null)
export const listTrail: Writable<Trail | null> = writable(null);

export async function lists_index(filter?: ListFilter, page: number = 1, perPage: number = 5,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const user = get(currentUser)

    const filterText = filter ? buildFilterText(user, filter) : ""

    const r = await f('/api/v1/list?' + new URLSearchParams({
        sort: `${filter?.sortOrder ?? "-"}${filter?.sort ?? "name"}`,
        perPage: perPage.toString(),
        page: page.toString(),
        filter: filterText,
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedLists: ListResult<List> = await r.json();

    const result = page > 1 ? [...lists, ...fetchedLists.items] : fetchedLists.items

    lists = result;

    return { ...fetchedLists, items: result };
}


export async function lists_search_filter(filter: ListFilter, page: number = 1, perPage: number = 30, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)

    const filterText = buildSearchFilterText(user, filter)

    let r = await f("/api/v1/search/lists", {
        method: "POST",
        body: JSON.stringify({
            q: filter.q,
            options: {
                filter: filterText, sort: filter.sort && filter.sortOrder ? [`${filter.sort}:${filter.sortOrder == "+" ? "asc" : "desc"}`] : [],
                hitsPerPage: perPage,
                page: page
            }
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedLists: { page: number, totalPages: number, hits: Hits<ListSearchResult> } = await r.json();

    const resultLists: List[] = await searchResultToLists(fetchedLists.hits)

    const result = page > 1 ? [...lists, ...resultLists] : resultLists

    lists = result;

    return { items: result, ...fetchedLists };
}

export async function lists_show(id: string, handle?: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f(`/api/v1/list/${id}?` + new URLSearchParams({
        expand: "author,trails,trails.author,trails.category,trails.tags,list_share_via_list.actor",
        ...(handle ? { handle } : {})
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response = await r.json()

    for (const trail of response.expand?.trails ?? []) {
        const gpxData: string = await fetchGPX(trail, f);
        trail.expand.gpx_data = gpxData;
    }

    list.set(response);

    return response;

}

export async function lists_create(list: List, avatar?: File) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }
    list.author = user.actor;

    const formData = objectToFormData(list, ["expand"])

    if (avatar) {
        formData.append("avatar", avatar);
    }

    let r = await fetch('/api/v1/list/form', {
        method: 'PUT',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: List = await r.json();

    return model;
}

export async function lists_update(list: List, avatar?: File) {

    const formData = objectToFormData(list, [ "expand" ])

    if (avatar) {
        formData.append("avatar", avatar);
    }

    let r = await fetch('/api/v1/list/form/' + list.id, {
        method: 'POST',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: List = await r.json();
    return model;
}

export async function lists_add_trail(list: List, trail: Trail) {
    const r = await fetch('/api/v1/list/' + list.id, {
        method: 'POST',
        body: JSON.stringify({
            "trails+": trail.id
        }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: List = await r.json();

    return model;
}

export async function lists_remove_trail(list: List, trail: Trail) {
    const r = await fetch('/api/v1/list/' + list.id, {
        method: 'POST',
        body: JSON.stringify({
            "trails-": trail.id
        }),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: List = await r.json();

    return model;
}

export async function lists_delete(list: List) {
    const r = await fetch('/api/v1/list/' + list.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}


function buildFilterText(user: AuthRecord | undefined, filter: ListFilter): string {
    let filterText = `(name~"${filter.q}"||description~"${filter.q}")`
    if (filter.author?.length) {
        filterText += `&&author="${filter.author}"`
    }
    if (user) {
        if (filter.public === false && filter.shared === false) {
            filterText += `&&author="${user.id}"`
        } else if (filter.public === true && filter.shared === false) {
            filterText += `&&(public=true||list_share_via_list.user!="${user.id}"||author="${user.id}")`
        } else if (filter.public === false && filter.shared === true) {
            filterText += `&&(public=false||list_share_via_list.user="${user.id}"||author="${user.id}")`
        }
    }
    return filterText
}

function buildSearchFilterText(user: AuthRecord, filter: ListFilter): string {
    let filterText: string = "";

    if (filter.author?.length) {
        filterText += `author = ${filter.author}`
    }

    if (filter.public !== undefined || filter.shared !== undefined) {
        if (filterText.length) {
            filterText += " AND "
        }
        filterText += "("
        if (filter.public !== undefined) {
            filterText += `(public = ${filter.public}`

            if (!filter.author?.length || filter.author == user?.actor) {
                filterText += ` OR author = ${user?.actor}`
            }
            filterText += ")"
        }

        if (filter.shared !== undefined) {
            if (filter.shared === true) {
                filterText += ` OR shares = ${user?.actor}`
            } else {
                filterText += ` AND NOT shares = ${user?.actor}`

            }
        }
        filterText += ")"
    }

    return filterText
}

export async function searchResultToLists(hits: Hits<ListSearchResult>): Promise<List[]> {
    const lists: List[] = []
    for (const h of hits) {
        const l: List & RecordModel = {
            collectionId: "lists",
            collectionName: "lists",
            updated: new Date(h.created * 1000).toISOString(),
            author: h.author,
            name: h.name,
            public: h.public,
            description: h.description,
            id: h.id,
            trails: Array(h.trails).fill("000000000000000"),
            avatar: h.avatar,
            elevation_gain: h.elevation_gain,
            elevation_loss: h.elevation_loss,
            distance: h.distance,
            duration: h.duration,
            iri: h.iri,
            expand: {
                author: {
                    icon: h.author_avatar,
                    id: h.author,
                    preferred_username: h.author_name,
                } as any,
                list_share_via_list: h.shares?.map(s => ({
                    permission: "view",
                    list: h.id,
                    actor: s,
                })),
            }
        }


        lists.push(l)
    }

    return lists
}