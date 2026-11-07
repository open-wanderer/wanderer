import type { RequestEvent } from "@sveltejs/kit";
import { beforeEach, describe, expect, it, vi } from "vitest";

const external = vi.hoisted(() => ({ fetchExternalJson: vi.fn() }));

vi.mock("#lib/util/activitypub_server_util.ts", () => ({
    getActorResponseForHandle: vi.fn(async () => ({
        actor: { id: "actor", is_local: false, iri: "https://remote.example/ap/users/bob", preferred_username: "bob", domain: "remote.example" },
    })),
}));
vi.mock("#lib/util/secure_fetch_util.ts", async (original) => ({
    ...(await original<typeof import("#lib/util/secure_fetch_util.ts")>()),
    fetchExternalJson: external.fetchExternalJson,
}));

import { DownloadError } from "#lib/util/secure_fetch_util.ts";
import { GET as feed } from "./feed/+server";
import { GET as stats } from "./stats/+server";

function event(path: string) {
    return {
        params: { handle: "bob@remote.example" },
        url: new URL(`http://localhost/api/v1/profile/bob/${path}?page=1`),
        fetch: vi.fn(),
        locals: {},
    } as unknown as RequestEvent;
}

describe.each([
    ["feed", feed, { items: [] }],
    ["stats", stats, []],
] as const)("federated %s", (path, get, body) => {
    beforeEach(() => {
        external.fetchExternalJson.mockReset();
    });

    it("goes through the guarded request and never through event.fetch", async () => {
        external.fetchExternalJson.mockResolvedValue({ status: 200, ok: true, json: body });
        const request = event(path);

        const response = await get(request);

        expect(response.status).toBe(200);
        const [url, options] = external.fetchExternalJson.mock.calls[0];
        expect(url).toContain(`https://remote.example/api/v1/profile/bob/${path}?`);
        expect(options.maxBytes).toBeGreaterThan(0);
        expect(request.fetch).not.toHaveBeenCalled();
    });

    it("passes on the remote instance's status", async () => {
        external.fetchExternalJson.mockResolvedValue({ status: 404, ok: false, json: { message: "nope" } });

        expect((await get(event(path))).status).toBe(404);
    });

    it("answers 502 when the instance cannot be reached safely", async () => {
        vi.spyOn(console, "warn").mockImplementation(() => {});
        external.fetchExternalJson.mockRejectedValue(new DownloadError("blocked_host", 400));

        const response = await get(event(path));

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual({ message: "remote_instance_unavailable" });
    });
});
