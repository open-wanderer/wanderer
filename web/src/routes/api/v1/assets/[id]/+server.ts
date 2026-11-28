import { RecordIdSchema, RecordIdValueSchema } from "#lib/models/api/base_schema.ts";
import { Collection, handleError } from "#lib/util/api_util.ts";
import type { RequestEvent } from "@sveltejs/kit";
import { z } from "zod";

/**
 * @swagger
 * /api/v1/assets/{id}:
 *   delete:
 *     summary: Delete asset or remove it from a target
 *     description: >
 *       Without a target, deletes an asset the user is the author of, together with its links.
 *       With `trail`, `waypoint` or `summit_log`, removes the asset from the given target instead.
 *       An existing asset requires target photo-edit permission or asset authorship; authors can withdraw their photos after a share ends.
 *       An unknown targeted asset only checks that the forwarded target exists, changes nothing, and returns assetDeleted=false without edit permission.
 *       After a matching link is removed, the asset itself is deleted only if it has no links left and the user is its author.
 *       `assetDeleted` says whether that happened; an unknown or unlinked targeted asset returns false.
 *     tags:
 *       - Assets
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *       - in: query
 *         name: trail
 *         schema:
 *           type: string
 *         description: Trail to remove the asset from. Ignored together with `waypoint` or `summit_log`.
 *       - in: query
 *         name: waypoint
 *         schema:
 *           type: string
 *         description: Waypoint to remove the asset from.
 *       - in: query
 *         name: summit_log
 *         schema:
 *           type: string
 *         description: Summit log to remove the asset from.
 *     responses:
 *       200:
 *         description: Asset deleted or removed from the target. A targeted removal is idempotent and returns assetDeleted=false when the asset is unknown or is not linked to that target.
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 acknowledged:
 *                   type: boolean
 *                 assetDeleted:
 *                   type: boolean
 *                   description: Only with a target. Whether the asset itself was deleted.
 *       400:
 *         description: Invalid asset or target id
 *       401:
 *         description: Unauthorized
 *       403:
 *         description: Without a target, the user is not the asset's author; otherwise insufficient permissions for the target
 *       404:
 *         description: Target not found, or asset not found when no target is given
 *       500:
 *         description: Internal Server Error
 */
export async function DELETE(event: RequestEvent) {
    try {
        if (!event.locals.user) {
            return Response.json({ message: "Unauthorized" }, { status: 401 });
        }

        const safeParams = RecordIdSchema.parse(event.params);

        const targets = z.object({
            trail: RecordIdValueSchema.optional(),
            waypoint: RecordIdValueSchema.optional(),
            summit_log: RecordIdValueSchema.optional(),
        }).parse(Object.fromEntries(event.url.searchParams));

        const linkTargets = assetLinkTargets(targets);

        const searchParams = new URLSearchParams();
        for (const target of linkTargets) {
            searchParams.set(target.field, target.id);
        }
        const query = searchParams.toString();
        const response = await event.locals.pb.send(`/assets/${safeParams.id}${query ? `?${query}` : ""}`, {
            method: "DELETE",
            fetch: event.fetch,
        });
        return Response.json(response);
    } catch (e: any) {
        return handleError(e);
    }
}

function assetLinkTargets(targets: { trail?: string; waypoint?: string; summit_log?: string }) {
    const trailIsDirectTarget = targets.trail && !targets.waypoint && !targets.summit_log;
    return [
        { field: "trail", id: trailIsDirectTarget ? targets.trail : undefined },
        { field: "waypoint", id: targets.waypoint },
        { field: "summit_log", id: targets.summit_log },
    ].filter((target): target is { field: string; id: string } => Boolean(target.id));
}
