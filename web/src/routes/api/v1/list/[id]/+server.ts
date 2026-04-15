import { ListUpdateSchema } from "$lib/models/api/list_schema";
import type { List } from "$lib/models/list";
import { Collection, handleError, remove, show, update } from "$lib/util/api_util";
import { objectToFormData } from "$lib/util/file_util";
import { json, type RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";

/**
 * @swagger
 * /api/v1/list/{id}:
 *   get:
 *     summary: Get list
 *     description: Retrieves a list by ID. Supports federated queries via handle parameter, fetching from remote instances and remapping file URLs
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: expand
 *         schema:
 *           type: string
 *       - in: query
 *         name: handle
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: List with optional federated data
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 *   post:
 *     summary: Update list
 *     description: Updates a list by ID
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *     responses:
 *       200:
 *         description: List
 *       400:
 *         description: Bad Request
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 *   delete:
 *     summary: Delete list
 *     description: Deletes a list by ID
 *     tags:
 *       - Lists
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *     responses:
 *       200:
 *         description: Success
 *       404:
 *         description: Not Found
 *       500:
 *         description: Internal Server Error
 */
export async function GET(event: RequestEvent) {
    try {
        if (!event.url.searchParams.has("handle")) {
            const l = await show<List>(event, Collection.lists)
            return json(l)
        } else {
            const {actor, error} = await event.locals.pb.send(`/activitypub/actor?resource=acct:${event.url.searchParams.get("handle")}`, { method: "GET", fetch: event.fetch, });
            event.url.searchParams.delete("handle")

            const origin = new URL(actor.iri).origin
            const url = `${origin}/api/v1/list/${event.params.id}`

            let dbList: List | undefined;
            try {
                dbList = await event.locals.pb.collection("lists").getFirstListItem(`iri='${url}'||id='${event.params.id}'`, {
                    ...Object.fromEntries(event.url.searchParams)
                })
            } catch (e) {
                if (!(e instanceof ClientResponseError) || e.status != 404) {
                    throw e
                }
            }

            if (actor.isLocal) {
                return json(dbList)
            } else {
                const response = await event.fetch((dbList?.iri ?? url) + '?' + event.url.searchParams, { method: 'GET' })
                if (!response.ok) {
                    const errorResponse = await response.json()
                    console.error(errorResponse)
                    const cachedList = await event.locals.pb.collection("lists").getOne(`${event.params.id}`)
                    return json(cachedList)
                }
                const l: List = await response.json()
                l.avatar = l.avatar ? `${origin}/api/v1/files/lists/${l.id}/${l.avatar}` : undefined

                l.author = actor.id!
                l.expand!.author = actor
                l.iri = dbList?.iri ?? url;

                l.expand?.trails?.forEach(t => {
                    t.gpx = t.gpx ? `${origin}/api/v1/files/trails/${t.id}/${t.gpx}` : undefined
                    t.photos = t.photos.map(p =>
                        `${origin}/api/v1/files/trails/${t.id}/${p}`
                    )
                    t.iri = t.iri || `${origin}/api/v1/trails/${t.id}`;
                })

                const formData = objectToFormData({ ...l, id: dbList?.id, expand: undefined, trails: [] })
                if (l.avatar) {
                    const avatarURL = l.avatar
                    let response = await event.fetch(avatarURL, { method: "GET" })
                    const avatar = await response.blob()


                    formData.append("avatar", avatar)
                }
                if (dbList !== undefined) {
                    dbList = await event.locals.pb.collection("lists").update(dbList.id!, formData)
                } else {
                    dbList = await event.locals.pb.collection("lists").create(formData)
                }

                l.id = dbList!.id

                return json(l)
            }

        }

    } catch (e: any) {
        return handleError(e)
    }
}

export async function POST(event: RequestEvent) {
    try {
        const r = await update<List>(event, ListUpdateSchema, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

export async function DELETE(event: RequestEvent) {
    try {
        const r = await remove(event, Collection.lists)
        return json(r);
    } catch (e: any) {
        return handleError(e)
    }
}

