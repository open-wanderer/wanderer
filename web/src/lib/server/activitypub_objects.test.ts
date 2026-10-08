import { describe, expect, it, vi } from "vitest";
import { acceptsActivityPub, activityPubObject } from "./activitypub_objects";

const request = (accept: string) => new Request("https://local.example/api/v1/trail/abc", { headers: { accept } });

describe("acceptsActivityPub", () => {
    it.each([
        "application/activity+json",
        'application/ld+json; profile="https://www.w3.org/ns/activitystreams"',
    ])("accepts %s", (accept) => expect(acceptsActivityPub(request(accept))).toBe(true));

    it.each(["application/json", "*/*", "application/ld+json"])(
        "rejects %s", (accept) => expect(acceptsActivityPub(request(accept))).toBe(false));
});

describe("activityPubObject", () => {
    it("serves the object at path as ActivityPub", async () => {
        const fetch = vi.fn(async () => Response.json({ type: "Note" }));
        const res = await activityPubObject({ fetch } as any, "/api/v1/activitypub/trail/abc");

        expect(fetch).toHaveBeenCalledWith("/api/v1/activitypub/trail/abc");
        expect(res.headers.get("content-type")).toBe("application/activity+json");
        expect(res.headers.get("vary")).toBe("Accept");
        expect(await res.json()).toEqual({ type: "Note" });
    });
});
