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
vi.mock("#lib/server/category_preference_filter.ts", () => ({ withTrailPreferenceMeiliFilter: vi.fn() }));

import { DownloadError } from "#lib/util/secure_fetch_util.ts";
import { POST as searchLists } from "./lists/+server";
import { POST as searchTrails } from "./trails/+server";

function event() {
    return {
        params: { handle: "bob@remote.example" },
        url: new URL("http://localhost/api/v1/profile/bob/trails?a=b"),
        request: new Request("http://localhost/x", { method: "POST", body: `{"q":"a"}` }),
        fetch: vi.fn(),
        locals: {},
    } as unknown as RequestEvent;
}

describe.each([
    ["trails", searchTrails],
    ["lists", searchLists],
] as const)("federated %s search", (kind, search) => {
    beforeEach(() => {
        external.fetchExternalJson.mockReset();
    });

    it("goes through the guarded request with a size limit", async () => {
        external.fetchExternalJson.mockResolvedValue({ status: 200, ok: true, json: { hits: [] } });
        const request = event();

        const response = await search(request);

        expect(response.status).toBe(200);
        expect(request.fetch).not.toHaveBeenCalled();
        const [url, options] = external.fetchExternalJson.mock.calls[0];
        expect(url).toBe(`https://remote.example/api/v1/profile/bob/${kind}?a=b`);
        expect(options).toMatchObject({ method: "POST", body: { q: "a" }, maxBytes: 8 * 1024 * 1024 });
    });

    it("passes on the remote error", async () => {
        external.fetchExternalJson.mockResolvedValue({ status: 404, ok: false, json: { message: "gone" } });

        expect((await search(event())).status).toBe(404);
    });

    it("answers 502 when the guard refuses the request", async () => {
        const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
        external.fetchExternalJson.mockRejectedValue(new DownloadError("invalid_url"));

        const response = await search(event());

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual({ message: "remote_instance_unavailable" });
        warn.mockRestore();
    });
});
