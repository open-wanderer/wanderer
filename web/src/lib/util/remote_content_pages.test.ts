import { render } from "svelte/server";
import { describe, expect, it, vi } from "vitest";
import { load as trailLoad } from "../../routes/trail/view/[handle]/[id]/+page";
import { load as mapLoad } from "../../routes/map/trail/[handle]/[id]/+page";
import { load as printLoad } from "../../routes/map/trail/[handle]/[id]/print/+page";
import { load as listLoad } from "../../routes/lists/[[handle]]/[[id]]/+page";
import proxyResponses from "./remote_content_proxy.fixture.json";
import ErrorPage from "../../routes/+error.svelte";

const page = vi.hoisted(() => ({ status: 422, error: { message: "This content is too large to load here.", originURL: "https://remote.example/lists/@alice/list00000000001" } as App.Error }));
vi.mock("$app/state", () => ({ page }));

describe("remote content pages", () => {
    it.each([trailLoad, mapLoad, printLoad, listLoad])("retains the message and link when the API rejects a document", async (load) => {
        const message = "This content is too large to load here. You can view it on its original instance.";
        const originURL = "https://remote.example/lists/@alice/list00000000001";
        const event = {
            params: { id: "list00000000001", handle: "@alice@remote.example" },
            url: new URL("https://local.example/lists/@alice@remote.example/list00000000001"),
            fetch: async () => Response.json({ status: 422, message, data: { code: "remote_document_limit", origin_url: originURL } }, { status: 422 }),
        } as never;
        await expect(load(event)).rejects.toMatchObject({ status: 422, body: { message, originURL } });
    });

    it.each(proxyResponses)("retains the original link from a complete proxy response $data.origin_url", async (body) => {
        const loaders = body.data.origin_url.includes("/lists/") ? [listLoad] : [trailLoad, mapLoad, printLoad];
        for (const load of loaders) {
            const event = {
                params: { id: "record000000001", handle: "@riley@remote.example" },
                url: new URL("https://local.example/lists/@riley@remote.example/record000000001"),
                fetch: async () => Response.json(body, { status: body.status }),
            } as never;
            await expect(load(event)).rejects.toMatchObject({ status: 422, body: { message: body.message, originURL: body.data.origin_url } });
        }
    });

    it("renders a clear message and the original view link", () => {
        const { body } = render(ErrorPage);
        expect(body).toContain("This content is too large to load here.");
        expect(body).toContain('href="https://remote.example/lists/@alice/list00000000001"');
        expect(body).toContain("View on the original instance");
    });
});
