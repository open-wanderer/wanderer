import type { RequestEvent } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ ORIGIN: "https://trails.example.com" }));

import { GET } from "./+server";

const ORIGIN = "https://trails.example.com";

function makeEvent(getFirstListItem: ReturnType<typeof vi.fn>) {
    const collection = vi.fn(() => ({ getFirstListItem }));
    const event = {
        url: new URL(`${ORIGIN}/api/v1/activitypub/instance`),
        locals: { pb: { collection } },
    } as unknown as RequestEvent;
    return { event, collection };
}

describe("instance actor document endpoint", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("LooksUpByTypeAndLocality", async () => {
        const getFirstListItem = vi.fn(async () => ({
            preferred_username: "-instance",
            username: "Wanderer at trails.example.com",
            inbox: `${ORIGIN}/api/v1/activitypub/instance/inbox`,
            outbox: `${ORIGIN}/api/v1/activitypub/instance/outbox`,
            public_key: "PEM-1",
        }));
        const { event, collection } = makeEvent(getFirstListItem);

        await GET(event);

        expect(collection).toHaveBeenCalledWith("activitypub_actors");
        expect(getFirstListItem).toHaveBeenCalledTimes(1);
        expect(getFirstListItem).toHaveBeenCalledWith(
            "actor_type='instance'&&is_local=true",
            { sort: "@rowid" },
        );
    });

    it("ServesStoredIdentity", async () => {
        const getFirstListItem = vi.fn(async () => ({
            preferred_username: "-instance",
            username: "Wanderer at trails.example.com",
            inbox: `${ORIGIN}/api/v1/activitypub/instance/inbox`,
            outbox: `${ORIGIN}/api/v1/activitypub/instance/outbox`,
            public_key: "PEM-1",
        }));
        const { event } = makeEvent(getFirstListItem);

        const response = await GET(event);

        expect(response.status).toBe(200);
        expect(response.headers.get("Content-Type")).toBe(
            "application/activity+json",
        );
        const doc = await response.json();
        const id = `${ORIGIN}/api/v1/activitypub/instance`;
        expect(doc.id).toBe(id);
        expect(doc.type).toBe("Application");
        expect(doc.preferredUsername).toBe("-instance");
        expect(doc.name).toBe("Wanderer at trails.example.com");
        expect(doc.inbox).toBe(`${id}/inbox`);
        expect(doc.outbox).toBe(`${id}/outbox`);
        expect(doc.publicKey.id).toBe(id + "#main-key");
        expect(doc.publicKey.owner).toBe(id);
        expect(doc.publicKey.publicKeyPem).toBe("PEM-1");
    });

    it("StaysSelfConsistentWhenTheStoredRowStillHasTheOldOrigin", async () => {
        // InitInstanceActor failed to move the row after an ORIGIN change, so the
        // stored inbox and outbox still name the former origin.
        const getFirstListItem = vi.fn(async () => ({
            iri: "https://former.example.com/api/v1/activitypub/instance",
            preferred_username: "-instance",
            username: "Wanderer at trails.example.com",
            inbox: "https://former.example.com/api/v1/activitypub/instance/inbox",
            outbox: "https://former.example.com/api/v1/activitypub/instance/outbox",
            public_key: "PEM-1",
        }));
        const { event } = makeEvent(getFirstListItem);

        const response = await GET(event);

        expect(response.status).toBe(200);
        const doc = await response.json();
        const id = `${ORIGIN}/api/v1/activitypub/instance`;
        expect(doc.id).toBe(id);
        expect(doc.inbox).toBe(`${id}/inbox`);
        expect(doc.outbox).toBe(`${id}/outbox`);
        expect(doc.publicKey.owner).toBe(id);
        expect(doc.publicKey.id).toBe(id + "#main-key");
        expect(JSON.stringify(doc)).not.toContain("former.example.com");
    });

    it("NoInstanceActor404", async () => {
        const getFirstListItem = vi.fn(async () => {
            throw new ClientResponseError({
                status: 404,
                response: { status: 404, message: "not found", data: {} },
            });
        });
        const { event } = makeEvent(getFirstListItem);

        const response = await GET(event);

        expect(response.status).toBe(404);
    });
});
