import type { RequestEvent } from "@sveltejs/kit";

export function acceptsActivityPub(request: Request) {
    const accept = request.headers.get("accept")?.toLowerCase() ?? "";
    return accept.includes("application/activity+json") ||
        (accept.includes("application/ld+json") && accept.includes("https://www.w3.org/ns/activitystreams"));
}

/** Answers an ActivityPub request for an object id with the object at path. */
export async function activityPubObject(event: RequestEvent, path: string) {
    const response = await event.fetch(path);
    return new Response(response.body, {
        status: response.status,
        headers: { "content-type": "application/activity+json", vary: "Accept" },
    });
}
