import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ ORIGIN: "https://trails.example.com" }));

import { GET } from "./+server";

const ORIGIN = "https://trails.example.com";

function makeEvent() {
    const getFirstListItem = vi.fn(async () => ({
        user: "user1",
        username: "demo",
        preferred_username: "demo",
        public_key: "PEM-1",
    }));
    const getOne = vi.fn(async () => ({
        id: "user1",
        bio: "",
        created: "2025-05-05T15:07:59.943Z",
    }));
    const collection = vi.fn((name: string) =>
        name === "activitypub_actors" ? { getFirstListItem } : { getOne },
    );
    const event = {
        params: { handle: "demo" },
        url: new URL(`${ORIGIN}/api/v1/activitypub/user/demo`),
        locals: { pb: { filter: (raw: string) => raw, collection } },
    } as unknown as RequestEvent;
    return { event };
}

describe("user actor document endpoint", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("AnnouncesThatFollowsAreAcceptedAutomatically", async () => {
        // Follows of a Person are accepted right away (ProcessFollowActivity).
        // Pixelfed treats a missing field as true and shows the profile as private.
        const { event } = makeEvent();

        const response = await GET(event);

        expect(response.status).toBe(200);
        const doc = await response.json();
        expect(doc.type).toBe("Person");
        expect(doc.manuallyApprovesFollowers).toBe(false);
    });
});
