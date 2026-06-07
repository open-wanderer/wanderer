import { Comment } from "$lib/models/comment";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";

export const comments: Writable<Comment[]> = writable([])

export async function comments_index(trailId: string) {
    let filter: string;

    let r = await fetch(`/api/v1/trail/${trailId}/comment?` + new URLSearchParams({
        expand: "author",
        sort: "-created",
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedComments: ListResult<Comment> = await r.json();

    comments.set(fetchedComments.items);

    return fetchedComments.items;
}

export async function comments_create(comment: Comment) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    comment.author = user.actor

    let r = await fetch('/api/v1/comment?' + new URLSearchParams({
        expand: "author",
    }), {
        method: 'PUT',
        body: JSON.stringify(comment),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Comment = await r.json();

    return model;
}

export async function comments_update(comment: Comment) {
    let r = await fetch('/api/v1/comment/' + comment.id, {
        method: 'POST',
        body: JSON.stringify(comment),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Comment = await r.json();

    return model;
}

export async function comments_delete(comment: Comment) {
    const r = await fetch('/api/v1/comment/' + comment.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}