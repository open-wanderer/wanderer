import type { RequestEvent } from "@sveltejs/kit";
import PocketBase, { ClientResponseError } from "pocketbase";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$app/env/private", () => ({ ORIGIN: "https://trails.example.com" }));

import { GET as getUser } from "./+server";
import { GET as getFollowers } from "./followers/+server";
import { GET as getFollowing } from "./following/+server";
import { GET as getOutbox } from "./outbox/+server";

const ORIGIN = "https://trails.example.com";
const FILTER_TEMPLATE =
    "preferred_username:lower={:username} && is_local=true && actor_type!='instance'";

// A real client only to bind parameters; no request is ever made.
const realPb = new PocketBase("http://127.0.0.1:9");

const handlers: {
    name: string;
    suffix: string;
    get: (event: RequestEvent) => Promise<Response>;
}[] = [
    { name: "user", suffix: "", get: getUser },
    { name: "followers", suffix: "/followers", get: getFollowers },
    { name: "following", suffix: "/following", get: getFollowing },
    { name: "outbox", suffix: "/outbox", get: getOutbox },
];

function makeEvent(handle: string, suffix: string) {
    const getFirstListItem = vi.fn(async (_filter: string) => {
        throw new ClientResponseError({
            status: 404,
            response: { status: 404, message: "not found", data: {} },
        });
    });
    const collection = vi.fn(() => ({ getFirstListItem }));
    const event = {
        params: { handle },
        url: new URL(
            `${ORIGIN}/api/v1/activitypub/user/${encodeURIComponent(handle)}${suffix}`,
        ),
        locals: {
            pb: {
                filter: (raw: string, params?: Record<string, unknown>) =>
                    realPb.filter(raw, params),
                collection,
            },
        },
    } as unknown as RequestEvent;
    return { event, collection, getFirstListItem };
}

describe.each(handlers)("$name handle lookup", ({ suffix, get }) => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("excludes the instance actor for handle -instance and answers 404", async () => {
        const { event, collection, getFirstListItem } = makeEvent(
            "-instance",
            suffix,
        );

        const response = await get(event);

        expect(collection).toHaveBeenCalledWith("activitypub_actors");
        expect(getFirstListItem).toHaveBeenCalledTimes(1);
        expect(getFirstListItem.mock.calls[0][0]).toBe(
            realPb.filter(FILTER_TEMPLATE, { username: "-instance" }),
        );
        expect(response.status).toBe(404);
    });

    it("strips the leading @ and lowercases, keeping the type exclusion for @Instance", async () => {
        const { event, getFirstListItem } = makeEvent("@Instance", suffix);

        await get(event);

        const filter = getFirstListItem.mock.calls[0][0];
        expect(filter).toBe(
            realPb.filter(FILTER_TEMPLATE, { username: "instance" }),
        );
        expect(filter.endsWith("&& actor_type!='instance'")).toBe(true);
    });

    it("binds a quote in the handle as a value so it cannot add an || clause", async () => {
        // The SDK wraps the value in double quotes and escapes embedded double
        // quotes, so neither quote character can leave the string literal.
        for (const handle of [
            "x'||actor_type='instance",
            'x"||actor_type="instance',
        ]) {
            const { event, getFirstListItem } = makeEvent(handle, suffix);

            await get(event);

            const filter = getFirstListItem.mock.calls[0][0];
            const username = handle.toLowerCase();
            expect(filter).toBe(realPb.filter(FILTER_TEMPLATE, { username }));
            expect(filter).toBe(
                `preferred_username:lower=${JSON.stringify(username)} && is_local=true && actor_type!='instance'`,
            );
            expect(filter.endsWith("&& actor_type!='instance'")).toBe(true);
        }
    });
});
